package cache

import (
	"math"
	"sort"
	"sync"
	"time"
)

// TTL implements Interface with eviction of sooner-to-expire elements
type TTL struct {
	*Cache
	index map[interface{}]int64
	mu    sync.Mutex
}

func NewTTL(size int) *TTL {
	if size <= 0 {
		return nil
	}
	return &TTL{
		Cache: New(size),
		index: make(map[interface{}]int64, size),
	}

}

func (c *TTL) Get(k interface{}) (v interface{}, exp time.Time, err error) {
	return c.Cache.Get(k)
}

type ttl struct {
	Key interface{}
	Exp int64
}

func (c *TTL) ttls() []ttl {
	ttls := make([]ttl, len(c.index))
	i := 0
	for k, e := range c.index {
		ttls[i].Key = k
		ttls[i].Exp = e
		i++
	}
	sort.Slice(ttls, func(i int, j int) bool {
		return ttls[i].Exp < ttls[j].Exp
	})
	return ttls
}

func (c *TTL) set(x interface{}, exp time.Time) {
	score := int64(math.MaxInt64)
	if !exp.IsZero() {
		score = exp.UnixNano()
	}
	c.index[x] = score

}

// Set assigns a value to a key and sets the expiration time.
// If the size limit is reached the oldest item stored is evicted to insert the new one
func (c *TTL) Set(x, y interface{}, exp time.Time) (err error) {
	c.mu.Lock()
	if err = c.Cache.Set(x, y, exp); err != ErrMaxSize {
		c.set(x, exp)
		c.mu.Unlock()
		return
	}
	ttls := c.ttls()
	for _, ttl := range ttls {
		delete(c.index, ttl.Key)
		c.Cache.Evict(ttl.Key)
		if err = c.Cache.Set(x, y, exp); err != ErrMaxSize {
			c.set(x, exp)
			c.mu.Unlock()
			return
		}
	}

	return
}

func (c *TTL) Evict(keys ...interface{}) (n int) {
	c.mu.Lock()
	for _, k := range keys {
		delete(c.index, k)
	}
	n = c.Cache.Evict(keys...)
	c.mu.Unlock()
	return

}

func (c *TTL) Trim(now time.Time) []interface{} {
	c.mu.Lock()
	expired := c.Cache.Trim(now)
	for _, k := range expired {
		delete(c.index, k)
	}
	c.mu.Unlock()
	return expired
}
