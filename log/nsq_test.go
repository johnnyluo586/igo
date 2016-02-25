package log

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/bitly/go-nsq"
)

func TestNsq(t *testing.T) {
	log := NewLogger(4096)
	log.SetLogger("nsq", `{"prefix":"game","addr":"http://127.0.0.1:4151/mpub?topic=LOG&binary=true"}`)
	log.Debug("debug")
	log.Informational("info")
	log.Notice("notice")
	log.Warning("warning")
	log.Error("error")
	log.Alert("alert")
	log.Critical("critical")
	log.Emergency("emergency")
	time.Sleep(time.Second * 4)
	readNsq()
}

type NsqHandle struct{}

func (n *NsqHandle) HandleMessage(msg *nsq.Message) error {
	fmt.Println(string(msg.Body))
	return nil
}

func readNsq() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	cfg := nsq.NewConfig()
	// dail timeout
	cfg.DialTimeout = time.Duration(5) * time.Second
	cfg.UserAgent = fmt.Sprint("go-nsq version:%v", nsq.VERSION)
	consumer, err := nsq.NewConsumer("LOG", "readNsq", cfg)
	if err != nil {
		fmt.Printf("error %v\n", err)
		os.Exit(0)
	}

	consumer.AddHandler(&NsqHandle{})
	err = consumer.ConnectToNSQDs([]string{"localhost:4150"})
	if err != nil {
		fmt.Printf("error %v\n", err)
		os.Exit(0)

	}
	err = consumer.ConnectToNSQLookupds([]string{"localhost:4160"})
	if err != nil {
		fmt.Printf("error %v\n", err)
		os.Exit(0)
	}
	for {
		select {
		case <-consumer.StopChan:
			return
		case <-sigChan:
			consumer.Stop()
		}
	}
}
