package cache_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	cache "github.com/alxarch/go-cache"
)

func Test_Blocking(t *testing.T) {
	release := make(chan struct{})

	var n int64
	upstream := cache.UpstreamFunc(func(x interface{}) (y interface{}, exp time.Time, err error) {
		<-release
		<-time.After(time.Millisecond)
		atomic.AddInt64(&n, 1)
		return 42, time.Time{}, nil
	})

	blocking := cache.Blocking(upstream)
	wg := new(sync.WaitGroup)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			x, _, _ := blocking.Get("answer")
			wg.Done()
			if x.(int) != 42 {
				t.Errorf("invalid answer %d", x.(int))
			}
		}()
	}
	close(release)
	wg.Wait()
	if n != 1 {
		t.Errorf("Multiple upstream requests %d", n)
	}

}
