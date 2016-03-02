package main

import (
	"database/sql"
	"testing"
)

//_ "github.com/go-sql-driver/mysql"

func Test_Conn(t *testing.T) {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:6603)/testdb")
	defer db.Close()
	rows, err := db.Query("select * from test_table")
	if err != nil {
		t.Log(err)
		return
	}
	t.Log("OK")
	_ = rows
}
