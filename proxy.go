package cache

import (
	"time"
)

type proxy struct {
	Upstream
	Cache Interface
}

func Proxy(u Upstream, c Interface) Upstream {
	return &proxy{Blocking(u), c}
}

func (p *proxy) Get(x interface{}) (y interface{}, exp *time.Time, err error) {
	if y, exp, err = p.Cache.Get(x); err == ErrKeyNotFound || err == ErrExpired {
		if y, exp, err = p.Upstream.Get(x); err == nil {
			p.Cache.Set(x, y, exp)
		}
	}
	return
}
