package cache

import (
	"container/list"
	"sync"
	"time"
)

const DefaultLRUQueueSize = 100

type LRU struct {
	cache   *Cache
	list    *list.List
	pending chan interface{}
	index   map[interface{}]*list.Element
	mu      sync.Mutex
}

func NewLRU(size int) *LRU {
	if size <= 0 {
		return nil
	}
	return &LRU{
		cache:   New(size),
		index:   make(map[interface{}]*list.Element),
		list:    list.New(),
		pending: make(chan interface{}, size),
	}
}

func (c *LRU) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Flush()
}
func (c *LRU) flush() {
	for {
		select {
		case p := <-c.pending:
			c.list.MoveToFront(c.index[p])
		default:
			return
		}
	}
}

func (c *LRU) Get(x interface{}) (y interface{}, exp *time.Time, err error) {
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

func (c *LRU) Set(x, y interface{}, exp *time.Time) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flush()
	el := c.index[x]
	if el == nil {
		el = c.list.PushFront(x)
		c.index[x] = el
	} else {
		c.list.MoveToFront(el)
	}
	for {
		if err = c.cache.Set(x, y, exp); err == MaxSizeError {
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

func (c *LRU) Evict(keys ...interface{}) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flush()
	for _, k := range keys {
		if el := c.index[k]; el != nil {
			c.list.Remove(el)
			delete(c.index, k)
		}
	}
	return c.cache.Evict(keys...)
}

func (c *LRU) Trim(now time.Time) []interface{} {
	expired := c.cache.Trim(now)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flush()
	for _, k := range expired {
		if el := c.index[k]; el != nil {
			delete(c.index, k)
			c.list.Remove(el)
		}
	}
	return expired
}

func (c *LRU) Metrics() *Metrics {
	return c.cache.Metrics()
}
