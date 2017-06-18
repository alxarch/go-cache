package cache

import (
	"container/list"
	"sync"
	"time"
)

// FIFO implements Interface with a first-in-first-out eviction policy.
type FIFO struct {
	cache *Cache
	list  *list.List
	index map[interface{}]*list.Element
	mu    sync.Mutex
}

func NewFIFO(size int) *FIFO {
	if size <= 0 {
		return nil
	}
	return &FIFO{
		cache: New(size),
		index: make(map[interface{}]*list.Element),
		list:  list.New(),
	}

}

func (c *FIFO) Get(k interface{}) (v interface{}, exp *time.Time, err error) {
	return c.cache.Get(k)
}

// Set assigns a value to a key and sets the expiration time.
// If the size limit is reached the oldest item stored is evicted to insert the new one
func (c *FIFO) Set(k, v interface{}, exp *time.Time) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el := c.index[k]
	if el == nil {
		el = c.list.PushFront(k)
		c.index[k] = el
	}
	for {
		if err = c.cache.Set(k, v, exp); err == MaxSizeError {
			if el = c.list.Back(); el != nil {
				c.list.Remove(el)
				k := el.Value
				delete(c.index, k)
				c.cache.Evict(k)
			} else {
				break
			}
		}
	}
	return
}

func (c *FIFO) Evict(keys ...interface{}) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range keys {
		if el := c.index[k]; el != nil {
			c.list.Remove(el)
			delete(c.index, k)
		}
	}

	return c.cache.Evict(keys...)

}

func (c *FIFO) Trim(now time.Time) []interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	expired := c.cache.Trim(now)
	for _, k := range expired {
		if el := c.index[k]; el != nil {
			delete(c.index, k)
			c.list.Remove(el)
		}
	}
	return expired
}

func (c *FIFO) Metrics() Metrics {
	return c.cache.Metrics()
}
