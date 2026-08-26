// Package syncutil provides synchronization primitives for concurrent data access.
package syncutil

import (
	"sync"
	"time"
)

// RWMap is a concurrent map with read-write lock protection.
type RWMap struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewRWMap creates a new RWMap.
func NewRWMap() *RWMap {
	return &RWMap{
		data: make(map[string]interface{}),
	}
}

// Get retrieves a value from the map.
func (m *RWMap) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	return v, ok
}

// Set sets a value in the map.
func (m *RWMap) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

// Delete removes a key from the map.
func (m *RWMap) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

// Has returns true if the key exists in the map.
func (m *RWMap) Has(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[key]
	return ok
}

// Len returns the number of entries in the map.
func (m *RWMap) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

// Keys returns all keys in the map.
func (m *RWMap) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys
}

// Range iterates over all key-value pairs.
func (m *RWMap) Range(fn func(key string, value interface{}) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for k, v := range m.data {
		if !fn(k, v) {
			break
		}
	}
}

// Clear removes all entries from the map.
func (m *RWMap) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]interface{})
}

// Swap atomically replaces the map and returns the old one.
func (m *RWMap) Swap(newMap map[string]interface{}) map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	old := m.data
	m.data = newMap
	return old
}

// SyncMap is a concurrent map with generic type support.
type SyncMap[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

// NewSyncMap creates a new SyncMap.
func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{
		data: make(map[K]V),
	}
}

// Get retrieves a value from the map.
func (m *SyncMap[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	return v, ok
}

// Set sets a value in the map.
func (m *SyncMap[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

// Delete removes a key from the map.
func (m *SyncMap[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

// Len returns the number of entries.
func (m *SyncMap[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

// Keys returns all keys.
func (m *SyncMap[K, V]) Keys() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]K, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys
}

// Range iterates over all entries.
func (m *SyncMap[K, V]) Range(fn func(key K, value V) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for k, v := range m.data {
		if !fn(k, v) {
			break
		}
	}
}

// Clear removes all entries.
func (m *SyncMap[K, V]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[K]V)
}

// TTLMap is a map with time-to-live expiration for entries.
type TTLMap struct {
	mu    sync.RWMutex
	data  map[string]ttlEntry
}

type ttlEntry struct {
	value    interface{}
	expireAt time.Time
}

// NewTTLMap creates a new TTLMap.
func NewTTLMap() *TTLMap {
	return &TTLMap{
		data: make(map[string]ttlEntry),
	}
}

// Set sets a value with a TTL.
func (m *TTLMap) Set(key string, value interface{}, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = ttlEntry{
		value:    value,
		expireAt: time.Now().Add(ttl),
	}
}

// Get retrieves a value, returns false if expired or not found.
func (m *TTLMap) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.data[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expireAt) {
		return nil, false
	}
	return entry.value, true
}

// Delete removes a key.
func (m *TTLMap) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

// Cleanup removes expired entries.
func (m *TTLMap) Cleanup() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	removed := 0
	for k, v := range m.data {
		if now.After(v.expireAt) {
			delete(m.data, k)
			removed++
		}
	}
	return removed
}

// Len returns the number of non-expired entries.
func (m *TTLMap) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	now := time.Now()
	count := 0
	for _, v := range m.data {
		if now.Before(v.expireAt) {
			count++
		}
	}
	return count
}

// Expired returns true if the key exists and is expired.
func (m *TTLMap) Expired(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.data[key]
	if !ok {
		return false
	}
	return time.Now().After(entry.expireAt)
}
