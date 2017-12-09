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
	if m := c.Metrics(); m.Evict != 0 || m.Expired != 0 || m.Hit != 0 || m.Items != 0 || m.Miss != 0 {
		t.Error("Invalid metrics")
	}
	c.Set("foo", "bar", time.Time{})
	if m := c.Metrics(); m.Evict != 0 || m.Expired != 0 || m.Hit != 0 || m.Items != 1 || m.Miss != 0 {
		t.Error("Invalid metrics")
	}
	{
		y, exp, err := c.Get("foo")
		if err != nil {
			t.Errorf("Unexpected error %s", err)
		}
		if !exp.IsZero() {
			t.Errorf("Invalid exp %s", exp)
		}
		if bar, ok := y.(string); !ok {
			t.Errorf("Invalid value %v", y)
		} else if bar != "bar" {
			t.Errorf("Invalid value %v", bar)
		}
		if m := c.Metrics(); m.Evict != 0 || m.Expired != 0 || m.Hit != 1 || m.Items != 1 || m.Miss != 0 {
			t.Error("Invalid metrics")
		}
	}

	c.Set("foo", "baz", time.Time{})
	{
		y, exp, err := c.Get("foo")
		if err != nil {
			t.Errorf("Unexpected error %s", err)
		}
		if !exp.IsZero() {
			t.Errorf("Invalid exp %s", exp)
		}
		if baz, ok := y.(string); !ok {
			t.Errorf("Invalid value %v", y)
		} else if baz != "baz" {
			t.Errorf("Invalid value %v", baz)
		}
		if m := c.Metrics(); m.Evict != 0 || m.Expired != 0 || m.Hit != 2 || m.Items != 1 || m.Miss != 0 {
			t.Error("Invalid metrics")
		}
	}
	c.Set("bar", "baz", time.Time{})
	if m := c.Metrics(); m.Evict != 0 || m.Expired != 0 || m.Hit != 2 || m.Items != 2 || m.Miss != 0 {
		t.Error("Invalid metrics")
	}
	c.Set("baz", "foo", time.Time{})
	if m := c.Metrics(); m.Evict != 1 || m.Expired != 0 || m.Hit != 2 || m.Items != 2 || m.Miss != 0 {
		t.Errorf("Invalid metrics %#v", m)
	}
	if n := c.Evict(); n != 2 {
		t.Errorf("invalid cache size %d", n)
	}
	{
		y, exp, err := c.Get("foo")
		if err != nil {
			t.Errorf("Unexpected error %s", err)
		}
		if !exp.IsZero() {
			t.Errorf("Invalid exp %s", exp)
		}
		if baz, ok := y.(string); !ok {
			t.Errorf("Invalid value %v", y)
		} else if baz != "baz" {
			t.Errorf("Invalid value %v", baz)
		}
		if m := c.Metrics(); m.Evict != 1 || m.Expired != 0 || m.Hit != 3 || m.Items != 2 || m.Miss != 0 {
			t.Errorf("Invalid metrics %#v", m)
		}
	}
	if _, _, err := c.Get("bar"); err != cache.ErrKeyNotFound {
		t.Errorf("invalid cache err %s", err)
	} else if m := c.Metrics(); m.Evict != 1 || m.Expired != 0 || m.Hit != 3 || m.Items != 2 || m.Miss != 1 {
		t.Errorf("Invalid metrics %#v", m)
	}

	if n := c.Evict("foo"); n != 1 {
		t.Errorf("invalid cache size %d", n)
	} else if m := c.Metrics(); m.Evict != 2 || m.Expired != 0 || m.Hit != 3 || m.Items != 1 || m.Miss != 1 {
		t.Errorf("Invalid metrics %#v", m)
	}
	now := time.Now().Add(time.Hour)
	c = cache.NewLRU(3)
	c.Set("foo", "bar", now.Add(-time.Second))
	c.Set("bar", "baz", now)
	c.Set("baz", "foo", now.Add(time.Second))
	if m := c.Metrics(); m.Evict != 0 || m.Expired != 0 || m.Hit != 0 || m.Items != 3 || m.Miss != 0 {
		t.Errorf("Invalid metrics %#v", m)
	}
	keys := c.Trim(now)
	if len(keys) != 1 {
		t.Errorf("invalid trim %d", len(keys))
	} else if keys[0] != "foo" {
		t.Errorf("invalid key %s", keys[0])
	}
	if m := c.Metrics(); m.Evict != 0 || m.Expired != 1 || m.Hit != 0 || m.Items != 2 || m.Miss != 0 {
		t.Errorf("Invalid metrics %#v", m)
	}

}
