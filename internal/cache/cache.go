// Package cache menyediakan in-memory cache sederhana yang dipakai
// untuk mempercepat pembacaan task yang sering diakses.
package cache

import (
	"sync"
	"time"
)

type entry struct {
	value     interface{}
	expiresAt time.Time
}

func (e entry) expired() bool {
	return time.Now().After(e.expiresAt)
}

// Cache adalah map dengan TTL per-key.
type Cache struct {
	mu      sync.RWMutex
	items   map[string]entry
	stopGC  chan struct{}
	gcOnce  sync.Once
}

// New membuat cache dan menjalankan goroutine pembersih entri kedaluwarsa.
func New(gcInterval time.Duration) *Cache {
	c := &Cache{
		items:  make(map[string]entry),
		stopGC: make(chan struct{}),
	}
	go c.runGC(gcInterval)
	return c
}

func (c *Cache) runGC(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, v := range c.items {
			if now.After(v.expiresAt) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}

// Stop menghentikan goroutine GC.
func (c *Cache) Stop() {
	c.gcOnce.Do(func() { close(c.stopGC) })
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if item.expired() {
		return nil, false
	}
	return item.value, true
}

func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = entry{value: value, expiresAt: time.Now().Add(ttl)}
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}
