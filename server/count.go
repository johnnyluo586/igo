package server

import "sync"

var (
	_defaultMax = int64(1024)
)

//Counter count the client connection and limit
type Counter interface {
	SetMax(int64) //set the max count.
	Size() int64  //get the current size of counter.
	Incr() bool   //incr will block when out of max count.
	Decr()
}

var _ Counter = &ChanCount{}

//ChanCount chan count will bolck when at max.
type ChanCount struct {
	max int64
	ch  chan struct{}
}

//Max max
func (c *ChanCount) SetMax(m int64) {
	if m <= 0 {
		m = _defaultMax
	}
	c.max = m
	if c.ch == nil {
		c.ch = make(chan struct{}, c.max)
	}
}

//Incr incr
func (c *ChanCount) Incr() bool {
	select {
	case c.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

//Decr decr
func (c *ChanCount) Decr() { <-c.ch }

//Size size
func (c *ChanCount) Size() int64 {
	return int64(len(c.ch))
}

//IntCount int count will when return false when at max.
type IntCount struct {
	sync.Mutex
	max int64
	cur int64
}

var _ Counter = &IntCount{}

//Max max
func (c *IntCount) SetMax(m int64) {
	c.max = m
}

//Incr incr
func (c *IntCount) Incr() bool {
	c.Lock()
	if c.cur > c.max {
		c.Unlock()
		return false
	}
	c.cur++
	c.Unlock()
	return true
}

//Decr decrs
func (c *IntCount) Decr() {
	c.Lock()
	c.cur--
	c.Unlock()
}

//Size size
func (c *IntCount) Size() (s int64) {
	c.Lock()
	s = c.cur
	c.Unlock()
	return
}
