package server

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
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
	cnt.Max(conf.Server.MaxClient)
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
	log.Alertf("Server Running on addr: %v, max client: %v", addr.String(), s.cfg.Server.MaxClient)
	s.signal()

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
	// if the connect not send data to server out of readDeadline, it will cut the connect.
	conn.SetReadDeadline(time.Now().Add(time.Second * readDeadline))
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

func (s *Server) close() {}

func (s *Server) signal() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sc
		log.Warnf("Got signal [%d] to exit.", sig)
		s.close()
		os.Exit(0)
	}()
}
