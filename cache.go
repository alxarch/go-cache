package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrKeyNotFound is returned by Get if a key is not found in the cache.
	ErrKeyNotFound = errors.New("Key not found.")
	ErrExpired     = errors.New("Key expired.")
	// ErrMaxSize is returned by Set on implementations of Interface that have max size limits.
	ErrMaxSize = errors.New("Cache full.")
)

// Interface is the common cache interface for all implementations.
type Interface interface {
	// Interface implements Upstream and returns ErrKeyNotFound if a key is not found in cache.
	Upstream
	// Set assigns a value to a key and sets the expiration time
	Set(key, value interface{}, exp *time.Time) error
	// Evict drops the provided keys from cache (if they exist) and returns the new size of the cache.
	// Calling Evict without arguments returns the current cache size.
	Evict(keys ...interface{}) (size int)
	Metrics() Metrics
}

// Never is a helper that returns a nil expiration time.
func Never() *time.Time {
	return nil
}

func Exp(ttl time.Duration) *time.Time {
	exp := time.Now().Add(ttl)
	return &exp
}

// Cache implements Interface.
// Removal of expired items is the responsibility of the caller.
type Cache struct {
	values  map[interface{}]interface{}
	exp     map[interface{}]*time.Time
	maxsize int
	mu      sync.RWMutex
	metrics Metrics
}

// New returns a new Cache.
// size determines the maximum number of items the cache can hold.
// If set to zero or less the cache will not have a size limit.
func New(size int) *Cache {
	if size < 0 {
		size = 0
	}

	return &Cache{
		values:  make(map[interface{}]interface{}, size),
		exp:     make(map[interface{}]*time.Time, size),
		maxsize: size,
	}
}

// Set assigns a value to a key and sets the expiration time
// If the size limit is reached it returns MaxSizeError.
func (c *Cache) Set(k, v interface{}, exp *time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.values[k]; !ok && c.maxsize > 0 && len(c.values) >= c.maxsize {
		return ErrMaxSize
	}
	c.values[k] = v
	if exp != nil {
		c.exp[k] = exp
	}
	return nil
}

// Get returns a value assigned to a key and it's expiration time.
// If a key does not exist in cache KeyError is returned.
// If a key is expired ExpiredError is returned
func (c *Cache) Get(k interface{}) (v interface{}, exp *time.Time, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	defer func() {
		if err == nil {
			atomic.AddUint64(&c.metrics.Hit, 1)
		} else {
			atomic.AddUint64(&c.metrics.Miss, 1)
		}
	}()
	var ok bool
	if v, ok = c.values[k]; !ok {
		err = ErrKeyNotFound
		return
	}
	if exp = c.exp[k]; exp == nil || exp.After(time.Now()) {
		return
	}
	err = ErrExpired
	return
}

// Size returns size of all keys in cache both expired and fresh
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.values)
}

// Trim removes all expired keys and returns a slice of removed keys
func (c *Cache) Trim(now time.Time) []interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	expired := make([]interface{}, 0, len(c.exp))
	for k, exp := range c.exp {
		if exp != nil && exp.Before(now) {
			expired = append(expired, k)
		}
	}
	for _, k := range expired {
		delete(c.exp, k)
		delete(c.values, k)
	}
	atomic.AddUint64(&c.metrics.Expired, uint64(len(expired)))
	return expired
}

func (c *Cache) evict(keys []interface{}) (n int) {
	for _, k := range keys {
		if _, ok := c.values[k]; ok {
			delete(c.values, k)
			delete(c.exp, k)
			n++
		}
	}
	return
}

// Evict removes items from the cache. It returns the new cache size.
func (c *Cache) Evict(keys ...interface{}) (size int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics.Evict += uint64(c.evict(keys))
	return len(c.values)
}

type Metrics struct {
	Hit, Miss, Evict, Expired, Items uint64
}

func (c *Cache) Metrics() (m Metrics) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m.Hit = atomic.LoadUint64(&c.metrics.Hit)
	m.Miss = atomic.LoadUint64(&c.metrics.Miss)
	m.Evict = c.metrics.Evict
	m.Expired = c.metrics.Expired
	m.Items = uint64(len(c.values))
	return
}

type EvictionPolicy string

const (
	PolicyNone EvictionPolicy = ""
	PolicyFIFO EvictionPolicy = "FIFO"
	PolicyLRU  EvictionPolicy = "LRU"
	PolicyLFU  EvictionPolicy = "LFU"
	PolicyTTL  EvictionPolicy = "TTL"
)

func NewCache(size int, policy EvictionPolicy) Interface {
	if size <= 0 {
		return New(0)
	}
	switch policy {
	case PolicyFIFO:
		return NewFIFO(size)
	case PolicyLRU:
		return NewLRU(size)
	case PolicyLFU:
		return NewLFU(size)
	case PolicyTTL:
		return NewTTL(size)
	default:
		return New(size)
	}
}
