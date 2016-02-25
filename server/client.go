package server

import (
	"fmt"
	"igo/mysql"
	"net"
	"time"
)

//Client the client connection object
type Client struct {
	con *net.TCPConn
	die chan struct{}
	seq uint8
}

func newClient(conn *net.TCPConn) (*Client, chan struct{}) {
	c := new(Client)
	c.die = make(chan struct{})

	return c, c.die
}

//----------------------------------------------------------
// Handshake
//----------------------------------------------------------

//Handshake handshake
func (c *Client) Handshake() error {
	if err := c.writeInitHandshake(); err != nil {
		return err
	}
	if err := c.readHandshakeRespone(); err != nil {
		//TODO write error.
		c.writeError()
		return err
	}
	c.writeOK()
	c.seq = 0

	return nil
}
func (c *Client) writeInitHandshake() error   { return nil }
func (c *Client) readHandshakeRespone() error { return nil }
func (c *Client) writeOK() error              { return nil }
func (c *Client) writeError() error           { return nil }

//----------------------------------------------------------
// packet
//----------------------------------------------------------

//Accept proccess the packet
func (c *Client) Accept() {
	data, err := c.readPacket()
	if err != nil {
		fmt.Println(err)
		c.close()
		return
	}
	if err := c.dispatch(data[0], data[1:]); err != nil {
		fmt.Println(err)
		c.close()
		return
	}
	c.seq = 0
}

func (c *Client) readPacket() ([]byte, error) {
	c.con.SetReadDeadline(time.Now().Add(readDeadline * time.Second))

	return nil, nil
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

func (c *Client) close() { close(c.die) }

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
