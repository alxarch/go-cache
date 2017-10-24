package cache_test

import (
	"testing"
	"time"

	cache "github.com/alxarch/go-cache"
)

func Test_LRU(t *testing.T) {
	c := cache.NewLRU(0)
	if c != nil {
		t.Error("Returns nil on zero size")
	}
	c = cache.NewLRU(2)
	if c == nil {
		t.Error("Returns non nil on non zero size")
	}
	m := c.Metrics()
	if m.Evict != 0 || m.Expired != 0 || m.Hit != 0 || m.Items != 0 || m.Miss != 0 {
		t.Error("Invalid metrics")
	}
	c.Set("foo", "bar", time.Time{})
	c.Set("foo", "baz", time.Time{})
	c.Set("bar", "baz", time.Time{})
	c.Get("foo")
	c.Set("baz", "foo", time.Time{})
	n := c.Evict()
	if n != 2 {
		t.Errorf("invalid cache size %d", n)
	}
	y, _, err := c.Get("foo")
	if err != nil {
		t.Errorf("invalid cache err %s", err)
	}
	if bar, ok := y.(string); !ok || bar != "baz" {
		t.Errorf("Invalid value %s", bar)
	}
	_, _, err = c.Get("bar")
	if err != cache.ErrKeyNotFound {
		t.Errorf("invalid cache err %s", err)
	}
	if n := c.Evict("foo"); n != 1 {
		t.Errorf("invalid cache size %d", n)
	}
	now := time.Now().Add(time.Hour)
	c = cache.NewLRU(3)
	c.Set("foo", "bar", now.Add(-time.Second))
	c.Set("bar", "baz", now)
	c.Set("baz", "foo", now.Add(time.Second))
	keys := c.Trim(now)
	if len(keys) != 1 {
		t.Errorf("invalid trim %d", len(keys))
	}

}
