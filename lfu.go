package cache

import (
	"sort"
	"sync"
	"time"
)

const DefaultLFUQueueSize = 100

type LFU struct {
	*Cache
	pending  chan interface{}
	requests map[interface{}]uint64
	mu       sync.Mutex
}

type lfu struct {
	Key      interface{}
	Requests uint64
}

func NewLFU(size int) *LFU {
	if size <= 0 {
		return nil
	}
	return &LFU{
		Cache:    New(size),
		requests: make(map[interface{}]uint64, size),
		pending:  make(chan interface{}, size),
	}
}

func (c *LFU) Flush() {
	c.mu.Lock()
	c.flush()
	c.mu.Unlock()
}

func (c *LFU) flush() {
	for {
		select {
		case p := <-c.pending:
			c.requests[p] += 1
		default:
			return
		}
	}
}

func (c *LFU) Get(x interface{}) (y interface{}, exp time.Time, err error) {
	y, exp, err = c.Cache.Get(x)
	if err == nil {
		select {
		case c.pending <- x:
			// pass
		default:
			c.Flush()
			c.pending <- x
		}
	}
	return
}

type lfus []lfu

func (lfs lfus) Less(i, j int) bool {
	return lfs[i].Requests < lfs[j].Requests
}
func (lfs lfus) Len() int {
	return len(lfs)
}

func (lfs lfus) Swap(i, j int) {
	k, n := lfs[i].Key, lfs[i].Requests
	lfs[i].Requests, lfs[i].Key = lfs[j].Requests, lfs[j].Key
	lfs[j].Requests, lfs[j].Key = n, k
}

func (c *LFU) lfus() []lfu {
	c.flush()
	ls := lfus(make([]lfu, len(c.requests)))
	i := 0
	for k, r := range c.requests {
		ls[i].Key = k
		ls[i].Requests = r
		i++
	}
	sort.Sort(ls)
	return ls
}

func (c *LFU) Set(x, y interface{}, exp time.Time) (err error) {
	c.mu.Lock()
	if err = c.Cache.Set(x, y, exp); err != ErrMaxSize {
		if err == nil {
			if _, ok := c.requests[x]; !ok {
				c.requests[x] = 0
			}
		}
		c.mu.Unlock()
		return
	}
	lfus := c.lfus()
	for _, lfu := range lfus {
		delete(c.requests, lfu.Key)
		c.Cache.Evict(lfu.Key)
		if err = c.Cache.Set(x, y, exp); err != ErrMaxSize {
			if err == nil {
				if _, ok := c.requests[x]; !ok {
					c.requests[x] = 0
				}
			}
			c.mu.Unlock()
			return
		}
	}

	c.mu.Unlock()
	return
}

func (c *LFU) Evict(keys ...interface{}) int {
	c.mu.Lock()
	c.flush()
	for _, k := range keys {
		delete(c.requests, k)
	}
	n := c.Cache.Evict(keys...)
	c.mu.Unlock()
	return n
}

func (c *LFU) Trim(now time.Time) []interface{} {
	expired := c.Cache.Trim(now)
	c.mu.Lock()
	c.flush()
	for _, k := range expired {
		delete(c.requests, k)
	}
	c.mu.Unlock()
	return expired
}
