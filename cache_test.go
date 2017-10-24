package cache_test

import (
	"testing"
	"time"

	cache "github.com/alxarch/go-cache"
)

func Test_Cache(t *testing.T) {
	c := cache.New(2)
	now := time.Now()
	c.Set("foo", "bar", now.Add(-time.Second))
	c.Set("bar", "baz", now)
	c.Set("baz", "foo", now.Add(time.Second))
	v, exp, err := c.Get("foo")
	if err != cache.ErrExpired {
		t.Errorf("unexpected err %s", err)
	}
	if v.(string) != "bar" {
		t.Errorf("invalid value %s", v.(string))
	}
	if !exp.Equal(now.Add(-time.Second)) {
		t.Errorf("invalid exp %s", exp)
	}
	keys := c.Trim(now)
	if len(keys) != 1 {
		t.Errorf("Invalid trim %s", keys)
	} else if keys[0] != "foo" {
		t.Errorf("Invalid trim %s", keys[0])
	}
	if c.Size() != 1 {
		t.Errorf("Invalid size %d", c.Size())
	}
}

func Test_Factory(t *testing.T) {
	c := cache.NewCache(100, "")
	if c, ok := c.(*cache.Cache); !ok {
		t.Errorf("Invalid cache type")
	} else if c.Cap() != 100 {
		t.Errorf("Invalid size %d", c.Cap())
	}
	c = cache.NewCache(100, cache.PolicyLFU)
	if c, ok := c.(*cache.LFU); !ok {
		t.Errorf("Invalid cache type")
	} else if c.Cap() != 100 {
		t.Errorf("Invalid size %d", c.Cap())
	}
	c = cache.NewCache(100, cache.PolicyLRU)
	if c, ok := c.(*cache.LRU); !ok {
		t.Errorf("Invalid cache type")
	} else if c.Cap() != 100 {
		t.Errorf("Invalid size %d", c.Cap())
	}
	c = cache.NewCache(100, cache.PolicyFIFO)
	if c, ok := c.(*cache.FIFO); !ok {
		t.Errorf("Invalid cache type")
	} else if c.Cap() != 100 {
		t.Errorf("Invalid size %d", c.Cap())
	}
	c = cache.NewCache(100, cache.PolicyTTL)
	if c, ok := c.(*cache.TTL); !ok {
		t.Errorf("Invalid cache type")
	} else if c.Cap() != 100 {
		t.Errorf("Invalid size %d", c.Cap())
	}
}
