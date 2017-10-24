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
	Set(key, value interface{}, exp time.Time) error
	// Evict drops the provided keys from cache (if they exist) and returns the new size of the cache.
	// Calling Evict without arguments returns the current cache size.
	Evict(keys ...interface{}) (size int)
	Metrics() Metrics
}

// Never is a helper that returns a nil expiration time.
func Never() time.Time {
	return time.Time{}
}

func Exp(ttl time.Duration) time.Time {
	return time.Now().Add(ttl)
}

type entry struct {
	value interface{}
	exp   time.Time
}

// Cache implements Interface.
// Removal of expired items is the responsibility of the caller.
type Cache struct {
	values  map[interface{}]*entry
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
		values:  make(map[interface{}]*entry, size),
		maxsize: size,
	}
}

// Set assigns a value to a key and sets the expiration time
// If the size limit is reached it returns MaxSizeError.
func (c *Cache) Set(k, v interface{}, exp time.Time) error {
	c.mu.Lock()
	if _, ok := c.values[k]; !ok && c.maxsize > 0 && len(c.values) >= c.maxsize {
		c.mu.Unlock()
		return ErrMaxSize
	}
	c.values[k] = &entry{v, exp}
	c.mu.Unlock()
	return nil
}

// Get returns a value assigned to a key and it's expiration time.
// If a key does not exist in cache KeyError is returned.
// If a key is expired ExpiredError is returned
func (c *Cache) Get(k interface{}) (v interface{}, exp time.Time, err error) {
	c.mu.RLock()
	e, ok := c.values[k]
	if !ok {
		c.mu.RUnlock()
		atomic.AddUint64(&c.metrics.Miss, 1)
		err = ErrKeyNotFound
		return
	}
	c.mu.RUnlock()
	v, exp = e.value, e.exp

	if exp.IsZero() || exp.After(time.Now()) {
		atomic.AddUint64(&c.metrics.Hit, 1)
		return
	}
	err = ErrExpired
	atomic.AddUint64(&c.metrics.Miss, 1)
	return
}

// Size returns size of all keys in cache both expired and fresh
func (c *Cache) Size() (n int) {
	c.mu.RLock()
	n = len(c.values)
	c.mu.RUnlock()
	return
}

// Size returns size of all keys in cache both expired and fresh
func (c *Cache) Cap() int {
	return c.maxsize
}

// Trim removes all expired keys and returns a slice of removed keys
func (c *Cache) Trim(now time.Time) (expired []interface{}) {
	expired = make([]interface{}, 0, 64)
	c.mu.Lock()
	for k, e := range c.values {
		if !e.exp.IsZero() && e.exp.Before(now) {
			delete(c.values, k)
			expired = append(expired, k)
		}
	}
	c.mu.Unlock()
	atomic.AddUint64(&c.metrics.Expired, uint64(len(expired)))
	return expired
}

func (c *Cache) evict(keys []interface{}) (n int) {
	for _, k := range keys {
		if _, ok := c.values[k]; ok {
			delete(c.values, k)
			n++
		}
	}
	return
}

// Evict removes items from the cache. It returns the new cache size.
func (c *Cache) Evict(keys ...interface{}) (size int) {
	c.mu.Lock()
	c.metrics.Evict += uint64(c.evict(keys))
	size = len(c.values)
	c.mu.Unlock()
	return
}

type Metrics struct {
	Hit, Miss, Evict, Expired, Items uint64
}

func (c *Cache) Metrics() (m Metrics) {
	m.Hit = atomic.LoadUint64(&c.metrics.Hit)
	m.Miss = atomic.LoadUint64(&c.metrics.Miss)
	c.mu.RLock()
	m.Evict = c.metrics.Evict
	m.Expired = c.metrics.Expired
	m.Items = uint64(len(c.values))
	c.mu.RUnlock()
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
