package log

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// NsqLogWriter implements LoggerInterface and writes messages to terminal.
type NsqLogWriter struct {
	ch     chan []byte
	Prefix string `json:"prefix"`
	Addr   string `json:"addr"`
	Level  int    `json:"level"`
}

// create NsqLogWriter returning as LoggerInterface.
func NewNsqWriter() LoggerInterface {
	nw := &NsqLogWriter{
		ch:    make(chan []byte, 4096),
		Level: LevelDebug,
	}
	go nw.publish()
	return nw
}

// aggregate & mpub
func (c *NsqLogWriter) publish() {
	ticker := time.NewTicker(10 * time.Millisecond)
	size := make([]byte, 4)
	for {
		select {
		case <-ticker.C:
			n := len(c.ch)
			if n == 0 {
				continue
			}
			// [ 4-byte num messages ]
			// [ 4-byte message #1 size ][ N-byte binary data ]
			//    ... (repeated <num_messages> times)
			buf := new(bytes.Buffer)
			binary.BigEndian.PutUint32(size, uint32(n))
			buf.Write(size)
			for i := 0; i < n; i++ {
				bts := <-c.ch
				binary.BigEndian.PutUint32(size, uint32(len(bts)))
				buf.Write(size)
				buf.Write(bts)
			}

			// http part
			resp, err := http.Post(c.Addr, "application/octet-stream", buf)
			if err != nil {
				fmt.Println(err, buf.String())
				continue
			}
			//fmt.Printf("Post nsq: %#v \n", resp)
			if _, err := ioutil.ReadAll(resp.Body); err != nil {
				fmt.Println(err)
			}
			resp.Body.Close()
		}
	}
}

// init console logger.
// jsonconfig like '{"level":LevelTrace}'.
func (c *NsqLogWriter) Init(jsonconfig string) error {
	if len(jsonconfig) == 0 {
		return nil
	}
	if err := json.Unmarshal([]byte(jsonconfig), c); err != nil {
		return err
	}
	if c.Addr == "" {
		return errors.New("jsonconfig must have addr, addr is empty")
	}
	return nil
}

// write message in console.
func (c *NsqLogWriter) WriteMsg(msg string, level int) error {
	if level > c.Level {
		return nil
	}
	m := time.Now().Format("2006-01-02 15:04:05 ") + colors[level](c.Prefix+" "+msg)
	//fmt.Println(m)
	c.ch <- []byte(m)
	return nil
}

// implementing method. empty.
func (c *NsqLogWriter) Destroy() {

}

// implementing method. empty.
func (c *NsqLogWriter) Flush() {

}
