package main

import (
	"database/sql"
	"log"
	"testing"
)
import (
	_ "github.com/go-sql-driver/mysql"
)

func Test_Conn(t *testing.T) {
	db, err := sql.Open("mysql", "root:root@tcp(172.20.4.11:3306)/testdb")
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)

	rows, err := db.Query("select * from test_table")
	if err != nil {
		t.Log(err)
		return
	}
	defer rows.Close()
	var id int
	var name string
	var lang string
	for rows.Next() {
		err := rows.Scan(&id, &name, &lang)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(id, name, lang)
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
}
