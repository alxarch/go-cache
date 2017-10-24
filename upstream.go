package cache

import (
	"sync"
	"time"
)

type Upstream interface {
	Get(x interface{}) (y interface{}, exp time.Time, err error)
}

type UpstreamFunc func(x interface{}) (y interface{}, exp time.Time, err error)

func (f UpstreamFunc) Get(x interface{}) (y interface{}, exp time.Time, err error) {
	return f(x)
}

type blockingUpstream struct {
	Upstream
	mu      sync.RWMutex
	pending map[interface{}]*pending
}

type pending struct {
	done  chan struct{}
	exp   time.Time
	value interface{}
	err   error
}

func (b *blockingUpstream) get(x interface{}) (p *pending) {
	b.mu.RLock()
	if p = b.pending[x]; p != nil {
		b.mu.RUnlock()
		return p
	}
	b.mu.RUnlock()
	b.mu.Lock()
	if p = b.pending[x]; p != nil {
		b.mu.Unlock()
		return p
	}
	p = &pending{
		done: make(chan struct{}),
	}
	b.pending[x] = p
	b.mu.Unlock()

	go func() {
		p.value, p.exp, p.err = b.Upstream.Get(x)
		b.mu.Lock()
		delete(b.pending, x)
		b.mu.Unlock()
		close(p.done)

	}()
	return p
}

func (b *blockingUpstream) Get(x interface{}) (interface{}, time.Time, error) {
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
