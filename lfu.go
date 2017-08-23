package cache

import (
	"sort"
	"sync"
	"time"
)

const DefaultLFUQueueSize = 100

type LFU struct {
	cache    *Cache
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
		cache:    New(size),
		requests: make(map[interface{}]uint64, size),
		pending:  make(chan interface{}, size),
	}
}

func (c *LFU) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flush()
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

func (c *LFU) Get(x interface{}) (y interface{}, exp *time.Time, err error) {
	y, exp, err = c.cache.Get(x)
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

func (c *LFU) lfus() []lfu {
	c.flush()
	lfus := make([]lfu, len(c.requests))
	i := 0
	for k, r := range c.requests {
		lfus[i].Key = k
		lfus[i].Requests = r
		i++
	}
	sort.Slice(lfus, func(i int, j int) bool {
		return lfus[i].Requests < lfus[j].Requests
	})
	return lfus
}

func (c *LFU) Set(x, y interface{}, exp *time.Time) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer func() {
		if _, ok := c.requests[x]; !ok {
			c.requests[x] = 0
		}
	}()
	if err = c.cache.Set(x, y, exp); err != ErrMaxSize {
		return
	}
	lfus := c.lfus()
	for _, lfu := range lfus {
		delete(c.requests, lfu.Key)
		c.cache.Evict(lfu.Key)
		if err = c.cache.Set(x, y, exp); err != ErrMaxSize {
			return
		}
	}

	return
}

func (c *LFU) Evict(keys ...interface{}) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flush()
	for _, k := range keys {
		delete(c.requests, k)
	}
	return c.cache.Evict(keys...)
}

func (c *LFU) Trim(now time.Time) []interface{} {
	expired := c.cache.Trim(now)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flush()
	for _, k := range expired {
		delete(c.requests, k)
	}
	return expired
}

func (c *LFU) Metrics() Metrics {
	return c.cache.Metrics()
}
