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

//Counter count the client connection and limit
type Counter interface {
	Max(int64)   //set the max count.
	Size() int64 //get the current size of counter.
	Incr()       //incr will block when out of max count.
	Decr()
}

//Server the server.
type Server struct {
	cfg   *config.Config
	count Counter
}

//NewServer new server
func NewServer(conf *config.Config) *Server {
	s := new(Server)
	s.cfg = conf

	//set counter
	cnt := new(ChanCount)
	cnt.Max(conf.Server.MaxClientConn)
	s.count = cnt

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
	log.Alertf("Server Running on addr: %v, max client: %v", addr.String(), s.cfg.Server.MaxClientConn)
	for {
		conn, err := ls.AcceptTCP()
		if err != nil {
			log.Error(err)
			continue
		}
		s.count.Incr()
		go s.handleClient(conn)
	}

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
		s.count.Decr()
		log.Warnf("Client Close: %v, Current client: %v", client.ConnectID(), s.count.Size())
	}()
	log.Warnf("New Connect from addr: %v, id: %v, Current client: %v", client.Addr(), client.ConnectID(), s.count.Size())
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

func (s *Server) Close() {}
