package cache_test

import (
	"testing"
	"time"

	cache "github.com/alxarch/go-cache"
)

func Test_TTL(t *testing.T) {
	ttl := cache.NewTTL(2)
	now := time.Now()
	exp := now.Add(time.Hour)
	ttl.Set("answer", 42, exp)
	exp2 := exp.Add(time.Hour)
	ttl.Set("answer2", 42, exp2)
	exp3 := exp2.Add(time.Hour)
	ttl.Set("answer3", 42, exp3)
	if _, _, err := ttl.Get("answer"); err != cache.ErrKeyNotFound {
		t.Errorf("Invalid err %s", err)
	}
	if ans, e, err := ttl.Get("answer2"); err != nil {
		t.Errorf("Invalid err %s", err)
	} else if ans.(int) != 42 {
		t.Errorf("Invalid answer %d", ans.(int))
	} else if e != exp2 {
		t.Errorf("Invalid exp\n%s\n%s", e, exp2)
	}
	ttl.Set("answer", 42, exp)
	trimmed := ttl.Trim(exp2)
	if len(trimmed) != 1 {
		t.Errorf("Trimmed len err %d", len(trimmed))
	}
	ttl.Set("answer", 42, exp)
	if ans, _, err := ttl.Get("answer"); err != nil {
		t.Errorf("Invalid err %s", err)
	} else if ans.(int) != 42 {
		t.Errorf("Invalid answer %d", ans.(int))
	}

	if n := ttl.Evict("answer", "answer2", "answer0"); n != 1 {
		t.Errorf("Invalid size %d", n)

	}
}
