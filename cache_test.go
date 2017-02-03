package cache_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alxarch/go-cache"
)

func Test_Cache(t *testing.T) {

	var n int64 = 0
	c := &cache.Cache{
		Upstream: cache.UpstreamFunc(func(x interface{}) (interface{}, error) {
			return atomic.AddInt64(&n, 1), nil
		}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	ctx2 := c.Run(ctx)
	if ctx != ctx2 {
		t.Error("Not same ctx")
	}
	y, yhit, yerr := c.Get("test")
	if yerr != nil {
		t.Error("Cache error")
	}
	yn, ok := y.(int64)
	if !ok {
		t.Error("Invalid response type")
	}
	if yhit {
		t.Error("No cache hit")
	}
	if yn != 1 {
		t.Error("Invalid response value")
	}
	// Give time to the parallel fetching to return
	time.Sleep(10 * time.Millisecond)
	z, zhit, zerr := c.Get("test")
	if zerr != nil {
		t.Error("Cache error")
	}
	zn, ok := z.(int64)
	if !ok {
		t.Error("Invalid response type")
	}
	if !zhit {
		t.Error("No cache hit")
	}
	if zn != 1 {
		t.Error("Too many requests %d", zn)
	}

	cancel()

}
