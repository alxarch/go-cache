package cache

import "sync"

type Upstream interface {
	Fetch(interface{}) (interface{}, error)
}
type UpstreamFunc func(interface{}) (interface{}, error)

func (f UpstreamFunc) Fetch(x interface{}) (interface{}, error) {
	return f(x)
}

type pending struct {
	done  chan struct{}
	value interface{}
	err   error
}
type BlockUpstream struct {
	Upstream
	mu      sync.RWMutex
	pending map[interface{}]*pending
}

func (b *BlockUpstream) get(x interface{}) *pending {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if nil != b.pending {
		return b.pending[x]
	}
	return nil
}
func (b *BlockUpstream) fetch(x interface{}) *pending {
	if p := b.get(x); p != nil {
		return p
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	p := &pending{
		done: make(chan struct{}),
	}
	if b.pending == nil {
		b.pending = make(map[interface{}]*pending)
	}
	b.pending[x] = p
	go func() {
		defer b.release(x)
		p.value, p.err = b.Upstream.Fetch(x)
	}()
	return p
}

func (b *BlockUpstream) release(x interface{}) {
	if b.pending != nil {
		b.mu.Lock()
		defer b.mu.Unlock()
		if p := b.pending[x]; p != nil {
			delete(b.pending, x)
			close(p.done)
		}
	}
}

func (b *BlockUpstream) Fetch(x interface{}) (interface{}, error) {
	p := b.fetch(x)
	select {
	case <-p.done:
		return p.value, p.err
	}
}

func Blocking(up Upstream) Upstream {
	return &BlockUpstream{
		Upstream: up,
		pending:  make(map[interface{}]*pending),
	}
}

type StaticUpstream map[interface{}]interface{}

func (u StaticUpstream) Fetch(x interface{}) (interface{}, error) {
	if u != nil {
		return u[x], nil
	} else {
		return nil, nil
	}
}
