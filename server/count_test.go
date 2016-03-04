package server

import (
	"testing"
	"time"
)

func Test_ChanIncr(t *testing.T) {
	c := new(ChanCount)
	c.SetMax(10000)
	for i := 0; i < 1000; i++ {
		go c.Incr()
	}
	<-time.After(5 * time.Second)
	t.Log(c.Size())
}

func Test_IntIncr(t *testing.T) {
	c := new(IntCount)
	c.SetMax(10000)
	for i := 0; i < 1000; i++ {
		go c.Incr()
	}
	<-time.After(5 * time.Second)
	t.Log(c.Size())
}
