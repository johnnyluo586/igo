package server

import (
	"sync/atomic"
)

var _ Counter = &ChanCount{}

//ChanCount chan count will bolck when at max.
type ChanCount struct {
	max int64
	ch  chan struct{}
}

func (c ChanCount) Max(m int64) {
	c.max = m
	if c.ch == nil {
		c.ch = make(chan struct{}, c.max)
	}
}

func (c *ChanCount) Incr() { c.ch <- struct{}{} }

func (c *ChanCount) Decr() { <-c.ch }

func (c *ChanCount) Size() int64 {
	return int64(len(c.ch))
}

//IntCount int count will when return false when at max.
type IntCount struct {
	max int64
	cur int64
}

func (c *IntCount) Max(m int64) {
	c.max = m
}

func (c *IntCount) Incr() bool {
	if c.cur > c.max {
		return false
	}
	c.cur = atomic.AddInt64(&c.cur, 1)
	return true
}

func (c *IntCount) Decr() {
	c.cur = atomic.AddInt64(&c.cur, -1)
}
