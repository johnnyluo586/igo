package server

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

import (
	"igo/config"
	"igo/log"
	"igo/mysql"
)

type mysqlConn struct {
	buf              buffer
	netConn          net.Conn
	affectedRows     uint64
	insertID         uint64
	dbname           string
	cfg              *config.ServerConfig
	maxPacketAllowed int
	maxWriteSize     int
	writeTimeout     time.Duration
	flags            mysql.ClientFlag
	status           mysql.StatusFlag
	sequence         uint8
	strict           bool
	createdAt        time.Time
}

func (mc *mysqlConn) Close() {
	if mc.netConn != nil {
		mc.netConn.Close()
	}
	mc.buf.nc = nil
	mc.netConn = nil
}

func (mc *mysqlConn) expired(timeout time.Duration) bool {
	if timeout <= 0 {
		return false
	}
	return mc.createdAt.Add(timeout).Before(nowFunc())
}

//Exec execute the cmd,and return the read all the  packet.
func (mc *mysqlConn) Exec(data []byte) ([][]byte, error) {
	cmd := data[0]
	arg := string(data[1:])
	if err := mc.writeCommandPacketStr(cmd, arg); err != nil {
		return nil, err
	}

	reshd, resLen, err := mc.readResultSetHeaderPacket()
	if err != nil {
		return nil, err
	}

	result := make([][]byte, 0, resLen*2+1)
	result = append(result, reshd)

	if resLen > 0 {
		// columns
		if result, err = mc.readResultSetPacket(result, resLen); err != nil {
			return nil, err
		}
		// rows
		if result, err = mc.readUntilEOF(result); err != nil {
			return nil, err
		}
	}
	//log.Debug("result:", result)
	return result, err
}

func (mc *mysqlConn) readResultSetPacket(res [][]byte, count int) ([][]byte, error) {

	for i := 0; ; i++ {
		data, err := mc.readPacket()
		if err != nil {
			return nil, err
		}
		res = append(res, data)
		//log.Debugf("res len:%v, last:%v", len(res), data)

		// EOF Packet
		if data[0] == mysql.HeaderEOF && (len(data) == 5 || len(data) == 1) {
			if i == count {
				return res, nil
			}
			return res, fmt.Errorf("column count mismatch n:%d len:%d", count, i)
		}
	}
}

// Reads Packets until EOF-Packet or an Error appears. Returns count of Packets read
func (mc *mysqlConn) readUntilEOF(res [][]byte) ([][]byte, error) {
	for {
		data, err := mc.readPacket()
		res = append(res, data)
		// No Err and no EOF Packet
		if err == nil && data[0] != mysql.HeaderEOF {
			continue
		}
		if err == nil && data[0] == mysql.HeaderEOF && len(data) == 5 {
			mc.status = readStatus(data[3:])
		}

		return res, err // Err or EOF
	}
}

// Result Set Header Packet
// http://dev.mysql.com/doc/internals/en/com-query-response.html#packet-ProtocolText::Resultset
func (mc *mysqlConn) readResultSetHeaderPacket() ([]byte, int, error) {
	data, err := mc.readPacket()
	if err == nil {
		switch data[0] {

		case mysql.HeaderOK:
			return data, 0, mc.handleOkPacket(data)

		case mysql.HeaderERR:
			return data, 0, mc.handleErrorPacket(data)

		case mysql.HeaderLocalInFile:
			return data, 0, fmt.Errorf("not support")
		}

		// column count
		num, _, n := readLengthEncodedInteger(data)
		if n-len(data) == 0 {
			return data, int(num), nil
		}

		return data, 0, mysql.ErrMalformPkt
	}
	return data, 0, err
}

// Error Packet
// http://dev.mysql.com/doc/internals/en/generic-response-packets.html#packet-ERR_Packet
func (mc *mysqlConn) handleErrorPacket(data []byte) error {
	if data[0] != mysql.HeaderERR {
		return mysql.ErrMalformPkt
	}

	// 0xff [1 byte]

	// Error Number [16 bit uint]
	errno := binary.LittleEndian.Uint16(data[1:3])

	pos := 3

	// SQL State [optional: # + 5bytes string]
	if data[3] == 0x23 {
		//sqlstate := string(data[4 : 4+5])
		pos = 9
	}

	// Error Message [string]
	return &mysql.MySQLError{
		Number:  errno,
		Message: string(data[pos:]),
	}
}

// Handshake Initialization Packet
// http://dev.mysql.com/doc/internals/en/connection-phase-packets.html#packet-Protocol::Handshake
func (mc *mysqlConn) readInitPacket() ([]byte, error) {
	data, err := mc.readPacket()
	if err != nil {
		return nil, err
	}

	if data[0] == mysql.HeaderERR {
		return nil, mc.handleErrorPacket(data)
	}

	// protocol version [1 byte]
	if data[0] < mysql.MinProtocolVersion {
		return nil, fmt.Errorf(
			"unsupported protocol version %d. Version %d or higher is required",
			data[0],
			mysql.MinProtocolVersion,
		)
	}

	// server version [null terminated string]
	// connection id [4 bytes]
	pos := 1 + bytes.IndexByte(data[1:], 0x00) + 1 + 4

	// first part of the password cipher [8 bytes]
	cipher := data[pos : pos+8]

	// (filler) always 0x00 [1 byte]
	pos += 8 + 1

	// capability flags (lower 2 bytes) [2 bytes]
	mc.flags = mysql.ClientFlag(binary.LittleEndian.Uint16(data[pos : pos+2]))
	if mc.flags&mysql.ClientProtocol41 == 0 {
		return nil, mysql.ErrOldProtocol
	}
	// if mc.flags&mysql.ClientSSL == 0 && mc.cfg.tls != nil {
	// 	return nil, mysql.ErrNoTLS
	// }
	pos += 2

	if len(data) > pos {
		// character set [1 byte]
		// status flags [2 bytes]
		// capability flags (upper 2 bytes) [2 bytes]
		// length of auth-plugin-data [1 byte]
		// reserved (all [00]) [10 bytes]
		pos += 1 + 2 + 2 + 1 + 10

		// second part of the password cipher [mininum 13 bytes],
		// where len=MAX(13, length of auth-plugin-data - 8)
		//
		// The web documentation is ambiguous about the length. However,
		// according to mysql-5.7/sql/auth/sql_authentication.cc line 538,
		// the 13th byte is "\0 byte, terminating the second part of
		// a scramble". So the second part of the password cipher is
		// a NULL terminated string that's at least 13 bytes with the
		// last byte being NULL.
		//
		// The official Python library uses the fixed length 12
		// which seems to work but technically could have a hidden bug.
		cipher = append(cipher, data[pos:pos+12]...)

		// TODO: Verify string termination
		// EOF if version (>= 5.5.7 and < 5.5.10) or (>= 5.6.0 and < 5.6.2)
		// \NUL otherwise
		//
		//if data[len(data)-1] == 0 {
		//	return
		//}
		//return ErrMalformPkt

		// make a memory safe copy of the cipher slice
		var b [20]byte
		copy(b[:], cipher)
		return b[:], nil
	}

	// make a memory safe copy of the cipher slice
	var b [8]byte
	copy(b[:], cipher)
	return b[:], nil
}

// Client Authentication Packet
// http://dev.mysql.com/doc/internals/en/connection-phase-packets.html#packet-Protocol::HandshakeResponse
func (mc *mysqlConn) writeAuthPacket(cipher []byte) error {
	// Adjust client flags based on server support
	clientFlags := mysql.ClientProtocol41 |
		mysql.ClientSecureConn |
		mysql.ClientLongPassword |
		mysql.ClientTransactions |
		//mysql.ClientLocalFiles |
		//mysql.ClientPluginAuth |
		//mysql.ClientMultiResults |
		mc.flags&mysql.ClientLongFlag

	// if mc.cfg.ClientFoundRows {
	// 	clientFlags |= mysql.ClientFoundRows
	// }

	// To enable TLS / SSL
	// if mc.cfg.tls != nil {
	// 	clientFlags |= ClientSSL
	// }

	// if mc.cfg.MultiStatements {
	// 	clientFlags |= ClientMultiStatements
	// }

	// User Password
	scrambleBuff := mysql.ScramblePassword(cipher, []byte(mc.cfg.Passwd))

	pktLen := 4 + 4 + 1 + 23 + len(mc.cfg.User) + 1 + 1 + len(scrambleBuff) + 21 + 1

	// To specify a db name
	if n := len(mc.cfg.DBName); n > 0 {
		clientFlags |= mysql.ClientConnectWithDB
		pktLen += n + 1
	}

	// Calculate packet length and get buffer with that size
	data := mc.buf.takeSmallBuffer(pktLen + 4)
	if data == nil {
		// can not take the buffer. Something must be wrong with the connection
		log.Error(mysql.ErrBusyBuffer)
		return driver.ErrBadConn
	}

	// ClientFlags [32 bit]
	data[4] = byte(clientFlags)
	data[5] = byte(clientFlags >> 8)
	data[6] = byte(clientFlags >> 16)
	data[7] = byte(clientFlags >> 24)

	// MaxPacketSize [32 bit] (none)
	data[8] = 0x00
	data[9] = 0x00
	data[10] = 0x00
	data[11] = 0x00

	// Charset [1 byte]
	var found bool
	data[12], found = mysql.Collations[mysql.DefaultCollation]
	if !found {
		// Note possibility for false negatives:
		// could be triggered  although the collation is valid if the
		// collations map does not contain entries the server supports.
		return errors.New("unknown collation")
	}

	// SSL Connection Request Packet
	// http://dev.mysql.com/doc/internals/en/connection-phase-packets.html#packet-Protocol::SSLRequest
	// if mc.cfg.tls != nil {
	// 	// Send TLS / SSL request packet
	// 	if err := mc.writePacket(data[:(4+4+1+23)+4]); err != nil {
	// 		return err
	// 	}

	// 	// Switch to TLS
	// 	tlsConn := tls.Client(mc.netConn, mc.cfg.tls)
	// 	if err := tlsConn.Handshake(); err != nil {
	// 		return err
	// 	}
	// 	mc.netConn = tlsConn
	// 	mc.buf.nc = tlsConn
	// }

	// Filler [23 bytes] (all 0x00)
	pos := 13
	for ; pos < 13+23; pos++ {
		data[pos] = 0
	}

	// User [null terminated string]
	if len(mc.cfg.User) > 0 {
		pos += copy(data[pos:], mc.cfg.User)
	}
	data[pos] = 0x00
	pos++

	// ScrambleBuffer [length encoded integer]
	data[pos] = byte(len(scrambleBuff))
	pos += 1 + copy(data[pos+1:], scrambleBuff)

	// Databasename [null terminated string]
	if len(mc.cfg.DBName) > 0 {
		pos += copy(data[pos:], mc.cfg.DBName)
		data[pos] = 0x00
		pos++
	}
	mc.dbname = mc.cfg.DBName

	// Assume native client during response
	pos += copy(data[pos:], "mysql_native_password")
	data[pos] = 0x00

	// Send Auth packet
	return mc.writePacket(data)
}

func (mc *mysqlConn) readInitOK() error {
	data, err := mc.readPacket()
	if err == nil {
		// packet indicator
		switch data[0] {

		case mysql.HeaderOK:
			return nil
		case mysql.HeaderEOF:
			if len(data) > 1 {
				plugin := string(data[1:bytes.IndexByte(data, 0x00)])
				if plugin == "mysql_old_password" {
					// using old_passwords
					return mysql.ErrOldPassword
				} else if plugin == "mysql_clear_password" {
					// using clear text password
					return mysql.ErrCleartextPassword
				} else {
					return mysql.ErrUnknownPlugin
				}
			} else {
				return mysql.ErrOldPassword
			}

		default: // Error otherwise
			return mc.handleErrorPacket(data)
		}
	}
	return err
}

func (mc *mysqlConn) cleanup() {

}

// returns the number read, whether the value is NULL and the number of bytes read
func readLengthEncodedInteger(b []byte) (uint64, bool, int) {
	// See issue #349
	if len(b) == 0 {
		return 0, true, 1
	}
	switch b[0] {

	// 251: NULL
	case 0xfb:
		return 0, true, 1

	// 252: value of following 2
	case 0xfc:
		return uint64(b[1]) | uint64(b[2])<<8, false, 3

	// 253: value of following 3
	case 0xfd:
		return uint64(b[1]) | uint64(b[2])<<8 | uint64(b[3])<<16, false, 4

	// 254: value of following 8
	case 0xfe:
		return uint64(b[1]) | uint64(b[2])<<8 | uint64(b[3])<<16 |
				uint64(b[4])<<24 | uint64(b[5])<<32 | uint64(b[6])<<40 |
				uint64(b[7])<<48 | uint64(b[8])<<56,
			false, 9
	}

	// 0-250: value of first byte
	return uint64(b[0]), false, 1
}

func readStatus(b []byte) mysql.StatusFlag {
	return mysql.StatusFlag(b[0]) | mysql.StatusFlag(b[1])<<8
}
