package server

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

import (
	"igo/config"
	"igo/log"
	"igo/mysql"
)

var (
	baseConnectID = uint32(1000) //atomic
)

//Client the client connection object
type Client struct {
	cfg *config.ServerConfig
	buf buffer

	netConn   *net.TCPConn
	die       chan struct{}
	user      string
	dbname    string
	connectID uint32

	status           uint16
	capability       uint32
	collation        byte
	sequence         uint8
	maxPacketAllowed int
	writeTimeout     time.Duration
	salt             []byte
}

func newClient(conn *net.TCPConn) (*Client, chan struct{}) {
	c := &Client{
		die:              make(chan struct{}),
		buf:              newBuffer(conn),
		connectID:        atomic.AddUint32(&baseConnectID, 1),
		salt:             make([]byte, 20),
		maxPacketAllowed: mysql.MaxPacketSize,
		writeTimeout:     time.Duration(10),
	}

	return c, c.die
}

func (c *Client) close() { close(c.die) }
func (c *Client) Close() { close(c.die) }

/******************************************************************************
*                           Handshake Process                                 *
******************************************************************************/

//Handshake handshake
func (c *Client) Handshake() error {
	if err := c.writeInitHandshake(); err != nil {
		return err
	}
	if err := c.readHandshakeRespone(); err != nil {
		//TODO write error.
		c.writeError(err)
		return err
	}
	c.writeOK()
	c.sequence = 0
	return nil
}

func (c *Client) writeInitHandshake() error {
	data := make([]byte, 4, 128)

	// min version 10
	data = append(data, 10)
	// server version[00]
	data = append(data, serverVersion...)
	data = append(data, 0)
	// connection id
	data = append(data, byte(c.connectID), byte(c.connectID>>8), byte(c.connectID>>16), byte(c.connectID>>24))
	// auth-plugin-data-part-1
	data = append(data, c.salt[0:8]...)
	// filler [00]
	data = append(data, 0)
	// capability flag lower 2 bytes, using default capability here
	data = append(data, byte(mysql.DefaultCapability), byte(mysql.DefaultCapability>>8))
	// charset, utf-8 default
	data = append(data, uint8(mysql.Collations[mysql.DefaultCollation]))
	//status
	data = append(data, byte(mysql.StatusInAutocommit), byte(mysql.StatusInAutocommit>>8))
	// below 13 byte may not be used
	// capability flag upper 2 bytes, using default capability here
	data = append(data, byte(mysql.DefaultCapability>>16), byte(mysql.DefaultCapability>>24))
	// filler [0x15], for wireshark dump, value is 0x15
	data = append(data, 0x15)
	// reserved 10 [00]
	data = append(data, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	// auth-plugin-data-part-2
	data = append(data, c.salt[8:]...)
	// filler [00]
	data = append(data, 0)
	err := c.writePacket(data)

	return err
}

func (c *Client) readHandshakeRespone() error {
	data, err := c.readPacket()
	if err != nil {
		return err
	}
	pos := 0

	//capability
	c.capability = binary.LittleEndian.Uint32(data[:4])
	pos += 4

	//skip max packet size
	pos += 4

	//charset, skip, if you want to use another charset, use set names
	//c.collation = CollationId(data[pos])
	pos++

	//skip reserved 23[00]
	pos += 23

	//user name
	c.user = string(data[pos : pos+bytes.IndexByte(data[pos:], 0)])

	pos += len(c.user) + 1

	//auth length and auth
	authLen := int(data[pos])
	pos++
	auth := data[pos : pos+authLen]

	checkAuth := mysql.ScramblePassword(c.salt, []byte(c.cfg.Passwd))
	if c.user != c.cfg.User || !bytes.Equal(auth, checkAuth) {
		log.Error("ClientConn", "readHandshakeResponse", "error", 0, "auth", auth, "checkAuth", checkAuth, "client_user", c.user, "config_set_user", c.cfg.User, "passworld", c.cfg.Passwd)
		return mysql.NewErrf(mysql.ErrAccessDenied, c.user, c.netConn.RemoteAddr().String(), "Yes")
	}

	pos += authLen

	var db string
	if c.capability&uint32(mysql.ClientConnectWithDB) > 0 {
		if len(data[pos:]) == 0 {
			return nil
		}

		db = string(data[pos : pos+bytes.IndexByte(data[pos:], 0)])
		pos += len(c.dbname) + 1

	} else {
		//if connect without database, use default db
		db = c.cfg.Schema
	}

	if err := c.useDB(db); err != nil {
		return err
	}

	return nil

}
func (c *Client) writeOK() error {
	data := make([]byte, 4, 32)
	data = append(data, mysql.HeaderOK)
	//afactrows, insertid.
	data = mysql.AppendLengthEncodedInteger(data, 0)
	data = mysql.AppendLengthEncodedInteger(data, 0)

	if c.capability&uint32(mysql.ClientProtocol41) > 0 {
		data = append(data, byte(c.status), byte(c.status>>8))
		data = append(data, 0, 0)
	}

	return c.writePacket(data)
}
func (c *Client) writeError(e error) error {
	var m *mysql.SQLError
	var ok bool
	if m, ok = e.(*mysql.SQLError); !ok {
		m = mysql.NewErr(mysql.ErrUnknown, e.Error())
	}
	data := make([]byte, 4, 16+len(m.Message))

	data = append(data, mysql.HeaderERR)
	data = append(data, byte(m.Code), byte(m.Code>>8))

	if c.capability&uint32(mysql.ClientProtocol41) > 0 {
		data = append(data, '#')
		data = append(data, m.State...)
	}
	data = append(data, m.Message...)
	return c.writePacket(data)
}

/******************************************************************************
*                           Packets Process                                   *
******************************************************************************/
// Packets documentation:
// http://dev.mysql.com/doc/internals/en/client-server-protocol.html

// Read packet to buffer 'data'
func (c *Client) readPacket() ([]byte, error) {
	var payload []byte
	for {
		// Read packet header
		data, err := c.buf.readNext(4)
		if err != nil {
			log.Error(err)
			c.Close()
			return nil, driver.ErrBadConn
		}

		// Packet Length [24 bit]
		pktLen := int(uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16)

		if pktLen < 1 {
			log.Error(mysql.ErrMalformPkt)
			c.Close()
			return nil, driver.ErrBadConn
		}

		// Check Packet Sync [8 bit]
		if data[3] != c.sequence {
			if data[3] > c.sequence {
				return nil, mysql.ErrPktSyncMul
			}
			return nil, mysql.ErrPktSync
		}
		c.sequence++

		// Read packet body [pktLen bytes]
		data, err = c.buf.readNext(pktLen)
		if err != nil {
			log.Error(err)
			c.Close()
			return nil, driver.ErrBadConn
		}

		isLastPacket := (pktLen < mysql.MaxPacketSize)

		// Zero allocations for non-splitting packets
		if isLastPacket && payload == nil {
			return data, nil
		}

		payload = append(payload, data...)

		if isLastPacket {
			return payload, nil
		}
	}
}

// Write packet buffer 'data'
func (c *Client) writePacket(data []byte) error {
	log.Debug(data)
	pktLen := len(data) - 4
	log.Debug("pktLen", pktLen)
	if pktLen > c.maxPacketAllowed {
		return mysql.ErrPktTooLarge
	}

	for {
		var size int
		if pktLen >= mysql.MaxPacketSize {
			data[0] = 0xff
			data[1] = 0xff
			data[2] = 0xff
			size = mysql.MaxPacketSize
		} else {
			data[0] = byte(pktLen)
			data[1] = byte(pktLen >> 8)
			data[2] = byte(pktLen >> 16)
			size = pktLen
		}
		data[3] = c.sequence

		// Write packet
		if c.writeTimeout > 0 {
			if err := c.netConn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
				return err
			}
		}

		n, err := c.netConn.Write(data[:4+size])
		if err == nil && n == 4+size {
			c.sequence++
			if size != mysql.MaxPacketSize {
				return nil
			}
			pktLen -= size
			data = data[size:]
			continue
		}

		// Handle error
		if err == nil { // n != len(data)
			log.Error(mysql.ErrMalformPkt)
		} else {
			log.Error(err)
		}
		return driver.ErrBadConn
	}
}

/******************************************************************************
*                          Dispatch Process                                   *
******************************************************************************/

//Accept call Accept and loop proccess the packet
func (c *Client) Accept() {
	data, err := c.readPacket()
	if err != nil {
		log.Error(err)
		c.close()
		return
	}
	if err := c.dispatch(data[0], data[1:]); err != nil {
		log.Error(err)
		c.close()
		return
	}
	c.sequence = 0
}

func (c *Client) dispatch(cmd byte, data []byte) error {
	switch cmd {
	case mysql.ComQuery:
		c.handleQuery(data)

	case mysql.ComInitDB:
		c.handleUseDB(data)

	case mysql.ComFieldList:
		c.handleFieldList(data)

	case mysql.ComStmtClose,
		mysql.ComStmtExecute,
		mysql.ComStmtFetch,
		mysql.ComStmtPrepare,
		mysql.ComStmtReset,
		mysql.ComStmtSendLongData:
		c.handleStmt(data)

	default:
		return fmt.Errorf("unsurport cmd %v  %v", cmd, string(data))
	}
	return nil
}

//----------------------------------------------------------
// dispatch handle
//----------------------------------------------------------

//handleQuery
func (c *Client) handleQuery(data []byte) error { return nil }

//handleUseDB
func (c *Client) handleUseDB(data []byte) error { return nil }

//handleFieldList
func (c *Client) handleFieldList(data []byte) error { return nil }

//handleStmt
func (c *Client) handleStmt(data []byte) error {
	return nil
}

func (c *Client) useDB(db string) error {

	return nil
}
