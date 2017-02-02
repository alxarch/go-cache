package cache

import (
	"context"
	"sync"
	"time"
)

const DefaultMaxItems = 1000
const DefaultExpireInterval = time.Second

type Node struct {
	Key      interface{}
	Data     interface{}
	Exp      time.Time
	Requests int64
	Next     *Node
	Prev     *Node
}

type Getter interface {
	Get(interface{}) (interface{}, error)
}
type GetterFunc func(interface{}) (interface{}, error)

func (f GetterFunc) Get(x interface{}) (interface{}, error) {
	return f(x)
}

type Cache struct {
	Upstream Getter
	MaxAge   time.Duration
	MaxItems int

	// MaxQueueSize int

	nodes map[interface{}]*Node
	head  *Node
	tail  *Node
	pool  *sync.Pool

	// queue chan interface{}

	requests  chan interface{}
	responses chan *Node

	add chan *Node
}

func (c *Cache) incr(n *Node) {
	if nil == n {
		return
	}
	n.Requests++
	if c.head == n {
		return
	}
	if n.Prev != nil {
		n.Prev.Next = n.Next
	}
	if n.Next != nil {
		n.Next.Prev = n.Prev
		if c.tail == n {
			c.tail = n.Next
		}
	}
	n.Prev = c.head
	n.Next = nil
	c.head = n
}

func (c *Cache) evict(n *Node) {
	delete(c.nodes, n.Key)
	if n.Prev != nil {
		n.Prev.Next = n.Next
	}
	if n.Next != nil {
		n.Next.Prev = n.Prev
	}
	if n == c.tail {
		c.tail = n.Next
	}
	if n == c.head {
		c.head = n.Prev
	}
	c.pool.Put(n)
}

func (c *Cache) Run(ctx context.Context) context.Context {
	c.add = make(chan *Node, 2*(c.MaxItems+1))
	c.requests = make(chan interface{})
	c.responses = make(chan *Node)
	c.nodes = make(map[interface{}]*Node)

	// fetching := make(map[interface{}]bool)
	// c.queue = make(chan interface{}, c.MaxQueueSize + 1)
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
			// case x := <- x.queue:
			// 	if len(fetching)
			case n := <-evict:
				c.evict(n)
			case n := <-c.add:
				if c.MaxItems > 0 && len(c.nodes) > c.MaxItems {
					evict <- c.tail
				}
				c.nodes[n.Key] = n
				if nil == c.tail {
					c.tail = n
				}
				if nil == c.head {
					c.head = n
				}
			case x := <-c.requests:
				if n := c.nodes[x]; n != nil {
					if n.Exp.After(time.Now()) {
						c.responses <- nil
						evict <- n
					}
					c.incr(n)
					c.responses <- n
				}
				c.responses <- nil
			}
		}
	}()
	return ctx
}

func (c *Cache) Get(x interface{}) (interface{}, error) {
	c.requests <- x
	n := <-c.responses
	if n == nil {
		return c.fetch(x)
	} else {
		return n.Data, nil
	}
}

func (c *Cache) node(x interface{}, y interface{}) *Node {
	n, _ := c.pool.Get().(*Node)
	if c.MaxAge > 0 {
		n.Exp = time.Now().Add(c.MaxAge)
	} else {
		n.Exp = time.Time{}
	}
	n.Key = x
	n.Data = y
	n.Next = nil
	n.Prev = nil
	n.Requests = 0
	return n
}

func (c *Cache) fetch(x interface{}) (interface{}, error) {
	rc := make(chan interface{}, 1)
	ec := make(chan error, 1)
	go func() {
		y, err := c.Upstream.Get(x)
		if err == nil {
			c.add <- c.node(x, y)
		}
		rc <- y
		ec <- err
	}()
	return <-rc, <-ec
}
