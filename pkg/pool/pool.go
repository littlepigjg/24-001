// Package pool provides a generic object pool for reuse.
package pool

import (
	"sync"
)

// Pool is a generic object pool.
type Pool struct {
	mu    sync.Mutex
	items [][]byte
	size  int
}

// New creates a new Pool with a maximum size.
func New(maxSize int) *Pool {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &Pool{
		items: make([][]byte, 0, maxSize),
		size:  maxSize,
	}
}

// Get retrieves a buffer from the pool or creates a new one.
func (p *Pool) Get(bufferSize int) []byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.items) > 0 {
		item := p.items[len(p.items)-1]
		p.items = p.items[:len(p.items)-1]
		// Clear the buffer
		for i := range item {
			item[i] = 0
		}
		return item
	}
	if bufferSize <= 0 {
		bufferSize = 4096
	}
	return make([]byte, bufferSize)
}

// Put returns a buffer to the pool.
func (p *Pool) Put(buf []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.items) >= p.size {
		return
	}
	// Clear the buffer before storing
	for i := range buf {
		buf[i] = 0
	}
	p.items = append(p.items, buf)
}

// Len returns the number of items in the pool.
func (p *Pool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.items)
}

// MaxSize returns the maximum pool size.
func (p *Pool) MaxSize() int {
	return p.size
}
