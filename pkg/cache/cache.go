// Package cache provides simple in-memory caching with TTL support.
package cache

import (
	"sync"
	"time"
)

// Item represents a cached item with expiration.
type Item struct {
	Value      interface{}
	Expiration time.Time
}

// Expired checks if the cache item has expired.
func (i *Item) Expired() bool {
	if i.Expiration.IsZero() {
		return false
	}
	return time.Now().After(i.Expiration)
}

// Cache is a thread-safe in-memory cache with TTL support.
type Cache struct {
	mu    sync.RWMutex
	items map[string]*Item
}

// New creates a new Cache instance.
func New() *Cache {
	return &Cache{
		items: make(map[string]*Item),
	}
}

// Set stores a value with an optional expiration duration.
// If duration is 0, the item never expires.
func (c *Cache) Set(key string, value interface{}, duration time.Duration) {
	var expiration time.Time
	if duration > 0 {
		expiration = time.Now().Add(duration)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = &Item{
		Value:      value,
		Expiration: expiration,
	}
}

// Get retrieves a value from the cache.
// Returns the value and true if found and not expired.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, ok := c.items[key]
	if !ok {
		return nil, false
	}
	if item.Expired() {
		delete(c.items, key)
		return nil, false
	}
	return item.Value, true
}

// Delete removes an item from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*Item)
}

// Size returns the number of items in the cache.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Cleanup removes all expired items from the cache.
func (c *Cache) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	removed := 0
	for key, item := range c.items {
		if item.Expired() {
			delete(c.items, key)
			removed++
		}
	}
	return removed
}
