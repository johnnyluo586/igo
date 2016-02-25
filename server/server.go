package server

import (
	"fmt"
	"net"
)

import (
	"igo/config"
	"igo/log"
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
func (s *Server) Run() error {
	if s.cfg.Server.Addr == "" {
		return fmt.Errorf("addr is not set")
	}
	addr, err := net.ResolveTCPAddr("tcp", s.cfg.Server.Addr)
	if err != nil {
		return err
	}
	ls, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	log.Info("Server Running on addr: ", addr.String())
	for {
		conn, err := ls.AcceptTCP()
		if err != nil {
			log.Error(err)
			continue
		}
		go s.handleClient(conn)
	}

}

func (s *Server) Close() {

}

func (s *Server) handleClient(conn *net.TCPConn) {
	defer utils.PrintPanicStack()

	//set conn
	conn.SetReadBuffer(rcvBuffer)
	conn.SetWriteBuffer(sndBuffer)
	//conn.SetKeepAlive(true)
	//conn.SetNoDelay(false)

	//new Client
	client, die := newClient(conn, &s.cfg.Server)
	defer func() {
		log.Info("Client Close: ", client.ConnectID())
	}()
	log.Infof("New Connect from addr: %v, id: %v", client.Addr(), client.ConnectID())
	if err := client.Handshake(); err != nil {
		log.Error(err)
		return
	}
	log.Info("Auth OK: ", client.ConnectID())

	for {
		client.Accept()

		//handle the connection close
		select {
		case <-die:

			return
		}
	}
}
