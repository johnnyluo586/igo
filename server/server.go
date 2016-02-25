package server

import (
	"fmt"
	"net"
	"os"
)

import (
	"igo/config"
	"igo/utils"
)

const (
	rcvBuffer    = 32767
	sndBuffer    = 65535
	readDeadline = 30 //s
)

type Serverer interface {
	Run()
}

//Server the server.
type Server struct {
	cfg *config.Config
}

//NewServer new server
func NewServer(conf *config.Config) *Server {
	s := new(Server)
	s.cfg = conf
	return s
}

//Run  run the server
func (s *Server) Run() {
	if s.cfg.Server.Addr == "" {
		fmt.Println("addr is not set")
		os.Exit(-1)
	}
	addr, err := net.ResolveTCPAddr("tcp", s.cfg.Server.Addr)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	ls, err := net.ListenTCP("tcp", addr)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	for {
		conn, err := ls.AcceptTCP()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go s.handleClient(conn)
	}
}

func (s *Server) handleClient(conn *net.TCPConn) {
	defer utils.PrintPanicStack()

	//set conn
	conn.SetReadBuffer(rcvBuffer)
	conn.SetWriteBuffer(sndBuffer)
	conn.SetNoDelay(false)

	//new Client
	client, die := newClient(conn)
	if err := client.Handshake(); err != nil {
		fmt.Println(err)
		return
	}

	for {
		client.Accept()

		select {
		case <-die:
			fmt.Println("client die")
			return
		}
	}
}
