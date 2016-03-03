package server

import (
	"igo/config"
	"igo/log"
	"igo/mysql"
	"net"
	"runtime"
	"sync"
	"time"
)

var nowFunc = time.Now

type MysqlDB struct {
	mu sync.Mutex

	addr   string
	user   string
	passwd string
	db     string
	state  mysql.StatusFlag

	maxLifetime time.Duration
	freeConn    chan *mysqlConn
	openCh      chan struct{}
	maxIdle     int
	maxOpen     int
	numOpen     int
}

func Open(conf *config.ServerConfig) (*MysqlDB, error) {
	m := &MysqlDB{
		addr:        conf.Addr,
		user:        conf.User,
		passwd:      conf.Passwd,
		db:          conf.DBName,
		maxIdle:     conf.MaxIdleConn,
		maxOpen:     conf.MaxConnNum,
		maxLifetime: time.Duration(conf.MaxLifeTime),
	}
	m.freeConn = make(chan *mysqlConn, m.maxOpen)
	m.openCh = make(chan struct{}, m.maxOpen)

	go m.opener()
	for i := 0; i < m.maxIdle; i++ {
		m.openCh <- struct{}{}

	}

	return m, nil
}

//NewConnect open the database connect.
func (m *MysqlDB) newConn() (*mysqlConn, error) {
	var err error

	// New mysqlConn
	mc := &mysqlConn{
		maxPacketAllowed: mysql.MaxPacketSize,
		maxWriteSize:     mysql.MaxPacketSize - 1,
		writeTimeout:     defaultWriteTimeout,
		createdAt:        nowFunc(),
	}
	mc.cfg = &config.ServerConfig{
		Addr:   m.addr,
		User:   m.user,
		Passwd: m.passwd,
		DBName: m.db,
	}

	if err != nil {
		return nil, err
	}
	mc.strict = mc.cfg.Strict

	// Connect to Server
	mc.netConn, err = net.Dial("tcp", mc.cfg.Addr)
	if err != nil {
		return nil, err
	}

	// Enable TCP Keepalives on TCP connections
	if tc, ok := mc.netConn.(*net.TCPConn); ok {
		if err := tc.SetKeepAlive(true); err != nil {
			// Don't send COM_QUIT before handshake.
			mc.netConn.Close()
			mc.netConn = nil
			return nil, err
		}
	}

	mc.buf = newBuffer(mc.netConn)

	// Set I/O timeouts
	mc.buf.timeout = time.Duration(mc.cfg.ReadTimeout)
	mc.writeTimeout = time.Duration(mc.cfg.WriteTimeout)

	// Reading Handshake Initialization Packet
	cipher, err := mc.readInitPacket()
	if err != nil {
		mc.cleanup()
		return nil, err
	}
	// Send Client Authentication Packet
	if err = mc.writeAuthPacket(cipher); err != nil {
		mc.cleanup()
		return nil, err
	}

	// Handle response to auth packet, switch methods if possible
	if err := mc.readInitOK(); err != nil {
		mc.cleanup()
		return nil, err
	}
	m.numOpen++
	return mc, nil
}

func (m *MysqlDB) opener() {
	for range m.openCh {
		conn, err := m.newConn()
		if err == nil {
			m.freeConn <- conn
			continue
		}
		log.Error(err)
	}
}

func (m *MysqlDB) maybeOpenNew() {
	if can := m.maxOpen - m.numOpen; can > 0 {
		for can > 0 {
			can--
			m.openCh <- struct{}{}
		}
	}
}

func (m *MysqlDB) getConn() *mysqlConn {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maybeOpenNew()
	select {
	case v, ok := <-m.freeConn:
		if ok {
			return v
		}
	default:
		log.Error("getConn case default, freeConn is empty and maxOpen at max.\n", stack())
		return nil
	}
	// log.Debugf("m.freeConn: cap:%v, len:%v", cap(m.freeConn), len(m.freeConn))
	// v, ok := <-m.freeConn
	// if ok {
	// 	return v
	// }
	return nil
}

func (m *MysqlDB) putConn(mc *mysqlConn) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.maxLifetime > 0 {
		if m.maxIdle < m.numOpen && mc.expired(m.maxLifetime) {
			m.numOpen--
			mc.Close()
			log.Debugf("Close mc by expired, maxIdle:%v, open:%v", m.maxIdle, m.numOpen)
			return nil
		}
	}
	select {
	case m.freeConn <- mc:
		log.Debug("put back mc in m.freeConn")
		return nil
	default:
		log.Debug("m.freeConn is full.", stack())
		m.numOpen--
		mc.Close()
	}

	return nil
}

func stack() string {
	var buf [2 << 10]byte
	return string(buf[:runtime.Stack(buf[:], false)])
}
