package server

import (
	"igo/config"
	"igo/log"
)

type nodeType byte

var (
	autoNode   nodeType
	masterNode nodeType = 1
	slaveNode  nodeType = 2
)

var (
	_defaultDB = make(map[nodeType]*MysqlDB)
)

//InitDB init the db connection
func InitDB(conf *config.ServerConfig) {
	db, err := Open(conf)
	if err != nil {
		log.Error(err)
		return
	}
	_defaultDB[masterNode] = db
}

//GetDB get the database
func GetDB(s string) *MysqlDB {
	//TODO parse the s , and choose a node for

	return _defaultDB[masterNode]
}
