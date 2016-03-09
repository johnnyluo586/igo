package server

import (
	"encoding/binary"
	"fmt"
	"igo/mysql"
)

//Tx is the transaction interface.
type Tx interface {
	Commit() error
	Rollback() error
}

type mysqlTx struct {
	mc *mysqlConn
}

var _ Tx = &mysqlTx{}

func (tx *mysqlTx) Commit() (err error) {
	if tx.mc == nil || tx.mc.netConn == nil {
		return mysql.ErrInvalidConn
	}
	err = tx.mc.writeCommandPacketStr(mysql.ComQuery, "COMMIT")
	tx.mc = nil
	return
}

func (tx *mysqlTx) Rollback() (err error) {
	if tx.mc == nil || tx.mc.netConn == nil {
		return mysql.ErrInvalidConn
	}
	err = tx.mc.writeCommandPacketStr(mysql.ComQuery, "ROLLBACK")
	tx.mc = nil
	return
}

//Stmt is the prepare stmt
type Stmt interface {
	Close() error

	// NumInput returns the number of placeholder parameters.
	//
	// If NumInput returns >= 0, the sql package will sanity check
	// argument counts from callers and return errors to the caller
	// before the statement's Exec or Query methods are called.
	//
	// NumInput may also return -1, if the driver doesn't know
	// its number of placeholders. In that case, the sql package
	// will not sanity check Exec or Query argument counts.
	NumInput() int

	// Exec executes a query that doesn't return rows, such
	// as an INSERT or UPDATE.
	Exec(data []byte) ([]byte, error)

	// Query executes a query that may return rows, such as a
	// SELECT.
	Query(data []byte) ([][]byte, error)
}

type mysqlField struct {
	tableName string
	name      string
	flags     mysql.FieldFlag
	fieldType byte
	decimals  byte
}

type mysqlStmt struct {
	id         uint32
	mc         *mysqlConn
	paramCount int
	columns    []mysqlField
}

var _ Stmt = &mysqlStmt{}

func (ms *mysqlStmt) Close() error {
	if ms.mc == nil || ms.mc.netConn == nil {
		return mysql.ErrBadConn
	}
	err := ms.mc.writeCommandPacketUint32(mysql.ComStmtClose, ms.id)
	ms.mc = nil
	return err
}

func (ms *mysqlStmt) NumInput() int {
	return ms.paramCount
}

func (ms *mysqlStmt) Exec(data []byte) ([]byte, error) {
	mc := ms.mc
	if mc.netConn == nil {
		return nil, mysql.ErrBadConn
	}
	// Send command
	result, err := mc.Exec(data)
	if err != nil {
		return nil, err
	}

	return result, err
}

func (ms *mysqlStmt) Query(data []byte) ([][]byte, error) {
	mc := ms.mc
	if mc.netConn == nil {
		return nil, mysql.ErrBadConn
	}
	// Send command
	result, err := mc.Query(data)
	if err != nil {
		return nil, err
	}

	return result, err
}

// Prepare Result Packets
// http://dev.mysql.com/doc/internals/en/com-stmt-prepare-response.html
func (ms *mysqlStmt) readPrepareResultPacket() ([]byte, uint16, error) {
	data, err := ms.mc.readPacket()
	if err == nil {
		// packet indicator [1 byte]
		if data[0] != mysql.HeaderOK {
			return data, 0, ms.mc.handleErrorPacket(data)
		}

		// statement id [4 bytes]
		ms.id = binary.LittleEndian.Uint32(data[1:5])

		// Column count [16 bit uint]
		columnCount := binary.LittleEndian.Uint16(data[5:7])

		// Param count [16 bit uint]
		ms.paramCount = int(binary.LittleEndian.Uint16(data[7:9]))

		// Reserved [8 bit]

		// Warning count [16 bit uint]
		if !ms.mc.strict {
			return data, columnCount, nil
		}

		// Check for warnings count > 0, only available in MySQL > 4.1
		if len(data) >= 12 && binary.LittleEndian.Uint16(data[10:12]) > 0 {
			return data, columnCount, fmt.Errorf("mc.getWarnings()")
		}
		return data, columnCount, nil
	}
	return data, 0, err
}
