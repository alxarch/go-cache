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
	wg    sync.WaitGroup
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
	p = &pending{}
	p.wg.Add(1)
	b.pending[x] = p
	b.mu.Unlock()

	go func() {
		p.value, p.exp, p.err = b.Upstream.Get(x)
		b.mu.Lock()
		delete(b.pending, x)
		b.mu.Unlock()
		p.wg.Done()
	}()
	return p
}

func (b *blockingUpstream) Get(x interface{}) (interface{}, time.Time, error) {
	p := b.get(x)
	p.wg.Wait()
	return p.value, p.exp, p.err
}

// Blocking avoids multiple simultaneous requests for the same key
func Blocking(up Upstream) Upstream {
	return &blockingUpstream{
		Upstream: up,
		pending:  make(map[interface{}]*pending),
	}
}
