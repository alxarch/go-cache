package cache

import (
	"container/list"
	"sync"
	"time"
)

// FIFO implements Interface with a first-in-first-out eviction policy.
type FIFO struct {
	*Cache
	list  *list.List
	index map[interface{}]*list.Element
	mu    sync.Mutex
}

func NewFIFO(size int) *FIFO {
	if size <= 0 {
		return nil
	}
	return &FIFO{
		Cache: New(size),
		index: make(map[interface{}]*list.Element, size),
		list:  list.New(),
	}

}

func (c *FIFO) Get(k interface{}) (v interface{}, exp time.Time, err error) {
	return c.Cache.Get(k)
}

// Set assigns a value to a key and sets the expiration time.
// If the size limit is reached the oldest item stored is evicted to insert the new one
func (c *FIFO) Set(k, v interface{}, exp time.Time) (err error) {
	c.mu.Lock()
	for {
		if err = c.Cache.Set(k, v, exp); err != ErrMaxSize {
			break
		}
		if el := c.list.Back(); el != nil {
			c.list.Remove(el)
			key := el.Value
			delete(c.index, key)
			c.Cache.Evict(key)
		} else {
			break
		}
	}
	if el, ok := c.index[k]; !ok {
		c.index[k] = c.list.PushFront(k)
	} else {
		c.list.PushFront(el)
	}
	c.mu.Unlock()
	return
}

func (c *FIFO) Evict(keys ...interface{}) int {
	c.mu.Lock()
	for _, k := range keys {
		if el := c.index[k]; el != nil {
			c.list.Remove(el)
			delete(c.index, k)
		}
	}
	c.mu.Unlock()

	return c.Cache.Evict(keys...)

}

func (c *FIFO) Trim(now time.Time) []interface{} {
	c.mu.Lock()
	expired := c.Cache.Trim(now)
	for _, k := range expired {
		if el := c.index[k]; el != nil {
			delete(c.index, k)
			c.list.Remove(el)
		}
	}
	c.mu.Unlock()
	return expired
}
