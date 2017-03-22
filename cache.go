package cache

import (
	"container/list"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const DefaultMaxItems = 1000
const DefaultExpireInterval = time.Second

type Node struct {
	Key      interface{}
	Data     interface{}
	Exp      time.Time
	Requests int64
	Element  *list.Element
}

type Upstream interface {
	Fetch(interface{}) (interface{}, error)
}
type UpstreamFunc func(interface{}) (interface{}, error)

func (f UpstreamFunc) Fetch(x interface{}) (interface{}, error) {
	return f(x)
}

type Cache struct {
	Namespace string
	Upstream  Upstream
	MaxAge    time.Duration
	MaxItems  int

	nodes map[interface{}]*Node
	lst   *list.List
	pool  *sync.Pool

	requests  chan interface{}
	responses chan *Node
	add       chan *Node

	stats struct {
		hit      uint64
		miss     uint64
		maxage   uint64
		maxitems uint64
	}
}

func (c *Cache) evict(n *Node) {
	delete(c.nodes, n.Key)
	c.lst.Remove(n.Element)
	c.pool.Put(n)
}

func (c *Cache) Stats() (uint64, uint64, uint64, uint64) {
	return atomic.LoadUint64(&c.stats.hit), atomic.LoadUint64(&c.stats.miss),
		atomic.LoadUint64(&c.stats.maxage), atomic.LoadUint64(&c.stats.maxitems)
}

func (c *Cache) Run(ctx context.Context) error {
	if nil != c.requests {
		return errors.New("Cache already running")
	}
	if nil == ctx {
		ctx = context.Background()
	}
	c.add = make(chan *Node, 2*(c.MaxItems+1))
	c.requests = make(chan interface{}, 2*(c.MaxItems+1))
	c.responses = make(chan *Node, 2*(c.MaxItems+1))
	c.nodes = make(map[interface{}]*Node)
	c.lst = list.New()

	c.pool = &sync.Pool{
		New: func() interface{} {
			return &Node{}
		},
	}

	evict := make(chan *Node, 2*(c.MaxItems+1))

	go func() {
		defer close(evict)
		defer close(c.add)
		defer close(c.requests)
		defer close(c.responses)

		for {
			select {
			case <-ctx.Done():
				return
			case n := <-evict:
				c.evict(n)
			case n := <-c.add:
				if c.MaxItems > 0 && len(c.nodes) > c.MaxItems {
					if el := c.lst.Back(); el != nil {
						if e, ok := el.Value.(*Node); ok {
							c.stats.maxitems++
							evict <- e
						}
					}
				}
				c.nodes[n.Key] = n
				n.Element = c.lst.PushFront(n)
			case x := <-c.requests:
				if n := c.nodes[x]; n != nil {
					if c.MaxAge > 0 && n.Exp.Before(time.Now()) {
						c.responses <- nil
						c.stats.maxage++
						evict <- n
					} else {
						if el := n.Element; el != nil {
							c.lst.MoveToFront(el)
						}
						n.Requests++
						c.stats.hit++
						c.responses <- n
					}
				} else {
					c.stats.miss++
					c.responses <- nil

				}
			}
		}
	}()
	return nil
}

func (c *Cache) Get(x interface{}) (interface{}, bool, error) {
	c.requests <- x
	n := <-c.responses
	if n == nil {
		y, err := c.fetch(x)
		return y, false, err
	} else {
		return n.Data, true, nil
	}
}

func (c *Cache) Fetch(x interface{}) (interface{}, error) {
	y, _, err := c.Get(x)
	return y, err
}

var zerotime = time.Time{}

func (c *Cache) node(x interface{}, y interface{}) *Node {
	n, _ := c.pool.Get().(*Node)
	if c.MaxAge > 0 {
		n.Exp = time.Now().Add(c.MaxAge)
	} else {
		n.Exp = zerotime
	}
	n.Key = x
	n.Data = y
	n.Requests = 0
	return n
}

var (
	NoUpstreamError = errors.New("No Upstream assigned to cache")
)

func (c *Cache) fetch(x interface{}) (interface{}, error) {
	if nil == c.Upstream {
		return nil, NoUpstreamError
	}
	rc := make(chan interface{}, 1)
	ec := make(chan error, 1)
	go func() {
		y, err := c.Upstream.Fetch(x)
		if err == nil {
			c.add <- c.node(x, y)
		}
		rc <- y
		ec <- err
	}()
	return <-rc, <-ec
}
