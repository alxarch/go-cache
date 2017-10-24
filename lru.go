package cache

import (
	"container/list"
	"sync"
	"time"
)

const DefaultLRUQueueSize = 100

type LRU struct {
	*Cache
	list    *list.List
	pending chan interface{}
	index   map[interface{}]*list.Element

	// Protects index and list
	mu sync.Mutex
}

func NewLRU(size int) (c *LRU) {
	if size > 0 {
		c = &LRU{
			Cache:   New(size),
			index:   make(map[interface{}]*list.Element),
			list:    list.New(),
			pending: make(chan interface{}, size),
		}
	}
	return
}

func (c *LRU) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flush()
}

func (c *LRU) flush() {
	for {
		select {
		case p := <-c.pending:
			if el := c.index[p]; el != nil {
				c.list.MoveToFront(el)
			}
		default:
			return
		}
	}
}

func (c *LRU) Get(x interface{}) (y interface{}, exp time.Time, err error) {
	y, exp, err = c.Cache.Get(x)
	if err == nil {
		select {
		case c.pending <- x:
			// pass
		default:
			// Max pending changes in queue, reorder list
			c.Flush()
			c.pending <- x
		}
	}
	return
}

func (c *LRU) Set(x, y interface{}, exp time.Time) (err error) {
	flushed := false
	c.mu.Lock()
	for {
		if err = c.Cache.Set(x, y, exp); err != ErrMaxSize {
			if _, ok := c.index[x]; !ok {
				c.index[x] = c.list.PushBack(x)
			}
			c.mu.Unlock()
			return
		}
		if !flushed {
			c.flush()
			flushed = true
		}
		// Evict elements until we have an open position for the new element
		if el := c.list.Back(); el != nil {
			c.list.Remove(el)
			k := el.Value
			delete(c.index, k)
			c.Cache.Evict(k)
		} else {
			break
		}
	}
	c.mu.Unlock()
	return
}

func (c *LRU) Evict(keys ...interface{}) int {
	c.mu.Lock()
	c.flush()
	for _, k := range keys {
		if el := c.index[k]; el != nil {
			c.list.Remove(el)
			delete(c.index, k)
		}
	}
	n := c.Cache.Evict(keys...)
	c.mu.Unlock()
	return n
}

// Trim removes expired pairs from the cache and LRU list
func (c *LRU) Trim(now time.Time) []interface{} {
	c.mu.Lock()
	expired := c.Cache.Trim(now)
	c.flush()
	for i := 0; i < len(expired); i++ {
		k := expired[i]
		if el := c.index[k]; el != nil {
			delete(c.index, k)
			c.list.Remove(el)
		}
	}
	c.mu.Unlock()
	return expired
}
