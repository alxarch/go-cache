package cache

import (
	"sync"
	"time"
)

type Upstream interface {
	Get(x interface{}) (y interface{}, exp *time.Time, err error)
}

type UpstreamFunc func(x interface{}) (y interface{}, exp *time.Time, err error)

func (f UpstreamFunc) Get(x interface{}) (y interface{}, exp *time.Time, err error) {
	return f(x)
}

type blockingUpstream struct {
	Upstream
	mu      sync.Mutex
	pending map[interface{}]*pending
}

type pending struct {
	done  chan struct{}
	exp   *time.Time
	value interface{}
	err   error
}

func (b *blockingUpstream) get(x interface{}) *pending {
	b.mu.Lock()
	defer b.mu.Unlock()
	p := b.pending[x]
	if p != nil {
		return p
	}

	p = &pending{
		done: make(chan struct{}),
	}
	b.pending[x] = p
	go func() {
		defer b.release(x)
		p.value, p.exp, p.err = b.Upstream.Get(x)
	}()
	return p
}

func (b *blockingUpstream) release(x interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if p := b.pending[x]; p != nil {
		close(p.done)
	}
	delete(b.pending, x)
}

func (b *blockingUpstream) Get(x interface{}) (interface{}, *time.Time, error) {
	p := b.get(x)
	<-p.done
	return p.value, p.exp, p.err
}

// Blocking avoids multiple simultaneous requests for the same key
func Blocking(up Upstream) Upstream {
	return &blockingUpstream{
		Upstream: up,
		pending:  make(map[interface{}]*pending),
	}
}
