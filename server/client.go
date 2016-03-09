package server

import (
	"bytes"
	"encoding/binary"
	"errors"
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
	baseConnectID    = uint32(1000) //atomic
	errNotfoundDB    = errors.New("not found db")
	errCannotGetConn = errors.New("can not get conn")
)

//Client the client connection object
type Client struct {
	cfg       *config.ServerConfig
	buf       buffer
	dbConn    *mysqlConn
	netConn   *net.TCPConn
	stmt      *mysqlStmt
	die       chan struct{}
	user      string
	dbname    string
	connectID uint32

	salt             []byte
	status           uint16
	capability       uint32
	collation        byte
	sequence         uint8
	maxPacketAllowed int
	writeTimeout     time.Duration
}

func newClient(conn *net.TCPConn, conf *config.ServerConfig) (*Client, chan struct{}) {
	c := &Client{
		netConn:          conn,
		die:              make(chan struct{}),
		buf:              newBuffer(conn),
		connectID:        atomic.AddUint32(&baseConnectID, 1),
		salt:             mysql.RandomBuf(20),
		maxPacketAllowed: mysql.MaxPacketSize,
		writeTimeout:     time.Duration(defaultWriteTimeout * time.Second),
		cfg:              conf,
	}

	return c, c.die
}

//Addr addr
func (c *Client) Addr() string {
	return c.netConn.RemoteAddr().String()
}

//ConnectID connect id
func (c *Client) ConnectID() uint32 {
	return c.connectID
}

func (c *Client) close() {
	close(c.die)
}

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
	// if authLen == 0 {
	// 	return mysql.NewErr(mysql.ErrAccessDenied, c.user, c.netConn.RemoteAddr().String(), "NO")
	// }

	// auth := data[pos : pos+authLen]
	// checkAuth := mysql.ScramblePassword(c.salt, []byte(c.cfg.Passwd))
	// if c.user != c.cfg.User || !bytes.Equal(auth, checkAuth) {
	// 	log.Error("readHandshakeResponse error: 0, auth: ", auth,
	// 		",checkAuth: ", checkAuth, ",client_user: ", c.user, ",config_set_user: ", c.cfg.User, ",passworld: ", c.cfg.Passwd)
	// 	return mysql.NewErr(mysql.ErrAccessDenied, c.user, c.netConn.RemoteAddr().String(), "Yes")
	// }

	pos += authLen

	var db string
	if c.capability&uint32(mysql.ClientConnectWithDB) > 0 {
		if len(data[pos:]) == 0 {
			return nil
		}

		db = string(data[pos : pos+bytes.IndexByte(data[pos:], 0)])
		pos += len(db) + 1

	} else {
		//if connect without database, use default db
		db = c.cfg.DBName
	}
	log.Debug("db ", db)
	c.dbname = db
	// if err := c.useDB(db); err != nil {
	// 	return err
	// }

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
*                          Dispatch Process                                   *
******************************************************************************/

//Accept call Accept and loop proccess the packet
func (c *Client) Accept() error {
	data, err := c.readPacket()
	if err != nil {
		return err
	}
	if err := c.dispatch(data); err != nil {
		return err
	}
	c.sequence = 0
	return nil
}

func (c *Client) dispatch(data []byte) error {
	log.Debugf("dispatch cmd:%v, data: %v", data[0], string(data[1:]))
	var err error
	switch data[0] {
	case mysql.ComQuit:
		err = c.handleQuit()
	case mysql.ComQuery:
		err = c.handleQuery(data)

	case mysql.ComInitDB:
		err = c.handleUseDB(data)

	case mysql.ComFieldList:
		err = c.handleFieldList(data)
	case
		mysql.ComStmtPrepare:
		err = c.handleStmtPrepare(data)

	case mysql.ComStmtExecute:
		err = c.handlestmtExec(data)

	case mysql.ComStmtClose:
		err = c.handleStmtClose(data)
	case mysql.ComStmtFetch,
		mysql.ComStmtReset,
		mysql.ComStmtSendLongData:
		err = c.handleStmt(data)

	default:
		return fmt.Errorf("unsurport cmd %v  %v", data[0], string(data))
	}

	return err
}

//----------------------------------------------------------
// dispatch handle
//----------------------------------------------------------

func (c *Client) handleQuit() error {
	log.Debug("handleQuit")
	c.close()
	return nil
}

func (c *Client) handleStmtClose(data []byte) error {
	stmtID := binary.LittleEndian.Uint32(data[1:])
	if stmtID != c.stmt.id {
		return mysql.NewErr(mysql.ErrUnknownStmtHandler, stmtID)
	}
	err := c.stmt.Close()
	c.stmt = nil
	return err
}

func (c *Client) handleStmtPrepare(data []byte) error {
	db := GetDB(string(data))
	if db == nil {
		return errNotfoundDB
	}
	conn := db.getConn()
	if conn == nil {
		return errCannotGetConn
	}
	// defer db.putConn(conn)

	useCmd := []byte(string(mysql.ComInitDB) + c.dbname)
	_, err := conn.Exec(useCmd)
	if err != nil {
		return err
	}

	res, stmt, err := conn.Prepare(string(data[1:]))
	log.Debugf("handleQuery:data:%v, res:%v", string(data[1:]), res)
	if err != nil {
		return err
	}
	c.stmt = stmt
	err = c.writeResultPackets(res)
	return err
}

func (c *Client) handlestmtExec(data []byte) error {
	if c.stmt == nil || c.stmt.mc == nil {
		return errCannotGetConn
	}
	res, err := c.stmt.Query(data)
	log.Debugf("handleQuery:data:%v, res:%v", string(data[1:]), res)
	err = c.writeResultPackets(res)
	return err
}

//handleQuery
func (c *Client) handleQuery(data []byte) error {
	db := GetDB(string(data))
	if db == nil {
		return errNotfoundDB
	}
	conn := db.getConn()
	if conn == nil {
		return errCannotGetConn
	}
	defer db.putConn(conn)

	useCmd := []byte(string(mysql.ComInitDB) + c.dbname)
	_, err := conn.Exec(useCmd)
	if err != nil {
		return err
	}

	res, err := conn.Query(data)
	//log.Debugf("handleQuery:data:%v, res:%v", string(data[1:]), res)
	if err != nil {
		return err
	}
	err = c.writeResultPackets(res)
	return err
}

//handleUseDB
func (c *Client) handleUseDB(data []byte) error {
	log.Debug("handleUseDB", c.dbname)
	return c.useDB(string(data[4:]))
}

//handleFieldList
func (c *Client) handleFieldList(data []byte) error {
	db := GetDB(string(data))
	if db == nil {
		return errNotfoundDB
	}
	conn := db.getConn()
	if conn == nil {
		return errCannotGetConn
	}
	defer db.putConn(conn)
	res, err := conn.Query(data)
	if err != nil {
		return err
	}
	err = c.writeResultPackets(res)
	return err
}

//handleStmt
func (c *Client) handleStmt(data []byte) error {
	return nil
}

func (c *Client) useDB(name string) error {
	data := []byte("use " + name)
	db := GetDB(string(data))
	if db == nil {
		return errNotfoundDB
	}
	conn := db.getConn()
	if conn == nil {
		return errCannotGetConn
	}
	defer db.putConn(conn)

	res, err := conn.Query(data)
	if err != nil {
		return err
	}
	err = c.writeResultPackets(res)
	if err == nil {
		c.dbname = string(data)
	}
	return err

}
