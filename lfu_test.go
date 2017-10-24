package cache_test

import (
	"testing"
	"time"

	cache "github.com/alxarch/go-cache"
)

func Test_LFU(t *testing.T) {

	c := cache.NewLFU(0)
	if c != nil {
		t.Error("Returns nil on zero size")
	}
	c = cache.NewLFU(2)
	if c == nil {
		t.Error("Returns non nil on non zero size")
	}
	m := c.Metrics()
	if m.Evict != 0 || m.Expired != 0 || m.Hit != 0 || m.Items != 0 || m.Miss != 0 {
		t.Error("Invalid metrics")
	}
	c.Set("foo", "bar", cache.Never())
	c.Set("foo", "baz", cache.Never())
	c.Set("bar", "baz", cache.Never())
	c.Get("foo")
	c.Set("baz", "foo", cache.Never())
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
	now := time.Now().Add(time.Hour)
	c = cache.NewLFU(3)
	c.Set("foo", "bar", now.Add(-time.Second))
	c.Get("foo")
	c.Set("bar", "baz", now)
	c.Set("baz", "foo", now.Add(time.Second))
	keys := c.Trim(now)
	if len(keys) != 1 {
		t.Errorf("invalid trim %d", len(keys))
	}
}
