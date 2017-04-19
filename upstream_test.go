package cache_test

import (
	"testing"

	cache "github.com/alxarch/go-cache"
)

func Test_Blocking(t *testing.T) {
	s := cache.StaticUpstream{}
	s["foo"] = "bar"
	u := cache.Blocking(s)
	foo, err := u.Fetch("foo")
	if foo != "bar" {
		t.Error("Invalid fetch")
	}
	if err != nil {
		t.Error("Invalid err")

	}
}
