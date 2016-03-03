package main

import (
	"database/sql"
	"testing"
)
import (
	_ "github.com/go-sql-driver/mysql"
)

func Test_Conn(t *testing.T) {
	db, err := sql.Open("mysql", "root:root@tcp(172.20.4.11:3306)/testdb")
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)

	defer db.Close()
	rows, err := db.Query("select * from test_table")
	if err != nil {
		t.Log(err)
		return
	}
	t.Log("OK")
	_ = rows
}
