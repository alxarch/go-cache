package cache

import (
	"math"
	"sort"
	"sync"
	"time"
)

// TTL implements Interface with eviction of sooner-to-expire elements
type TTL struct {
	cache *Cache
	index map[interface{}]int64
	mu    sync.Mutex
}

func NewTTL(size int) *TTL {
	if size <= 0 {
		return nil
	}
	return &TTL{
		cache: New(size),
		index: make(map[interface{}]int64, size),
	}

}

func (c *TTL) Get(k interface{}) (v interface{}, exp *time.Time, err error) {
	return c.cache.Get(k)
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

// Set assigns a value to a key and sets the expiration time.
// If the size limit is reached the oldest item stored is evicted to insert the new one
func (c *TTL) Set(x, y interface{}, exp *time.Time) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer func() {
		score := int64(math.MaxInt64)
		if exp != nil {
			score = exp.UnixNano()
		}
		c.index[x] = score
	}()
	if err = c.cache.Set(x, y, exp); err != ErrMaxSize {
		return
	}
	ttls := c.ttls()
	for _, ttl := range ttls {
		delete(c.index, ttl.Key)
		c.cache.Evict(ttl.Key)
		if err = c.cache.Set(x, y, exp); err != ErrMaxSize {
			return
		}
	}

	return
}

func (c *TTL) Evict(keys ...interface{}) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range keys {
		delete(c.index, k)
	}

	return c.cache.Evict(keys...)

}

func (c *TTL) Trim(now time.Time) []interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	expired := c.cache.Trim(now)
	for _, k := range expired {
		delete(c.index, k)
	}
	return expired
}

func (c *TTL) Metrics() Metrics {
	return c.cache.Metrics()
}
