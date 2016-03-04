package main

import (
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"
)
import (
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

var defaultdb *sql.DB

func Test_Conn(t *testing.T) {
	db, err := sql.Open("mysql", "root:root@tcp(172.20.4.11:3306)/testdb")
	if err != nil {
		t.Error(err)
	}
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(512)
	defaultdb = db
}

func Test_Query(t *testing.T) {
	rows, err := defaultdb.Query("select * from test_table limit 1, 100")
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

func Test_Insert(t *testing.T) {
	res, err := defaultdb.Exec(fmt.Sprintf("insert into test_table(name, lang ) values('%v','%v')", "name_aaa", "name_bbb"))
	if err != nil {
		t.Log(err)
		return
	}
	insertID, _ := res.LastInsertId()
	affRows, _ := res.RowsAffected()
	t.Log("res: ", insertID, affRows)

}

func Benchmark_Query(b *testing.B) {

	wg := new(sync.WaitGroup)
	for i := 1; i <= b.N; i++ {
		wg.Add(1)
		go query(wg, i)
	}
	wg.Wait()

}

func Benchmark_Delay(b *testing.B) {
	defer defaultdb.Close()
	<-time.After(1 * time.Second)
}

func query(wg *sync.WaitGroup, i int) {
	defer wg.Done()
	rows, err := defaultdb.Query(fmt.Sprintf("select * from test_table where id= %v", i))
	if err != nil {
		log.Println(err)
		return
	}
	defer rows.Close()

	var id int
	var name string
	var lang string
	for rows.Next() {
		err := rows.Scan(&id, &name, &lang)
		if err != nil {
			log.Println(err)
			return
		}
		//log.Println(id, name, lang)
	}
	err = rows.Err()
	if err != nil {
		log.Println(err)
		return
	}

}
