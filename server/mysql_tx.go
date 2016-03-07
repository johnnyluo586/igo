package server

import "igo/mysql"

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
	Exec(args []byte) ([]byte, error)

	// Query executes a query that may return rows, such as a
	// SELECT.
	Query(args []byte) ([][]byte, error)
}
