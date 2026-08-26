// Package ratelimit provides rate limiting for API requests.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter implements a token bucket rate limiter.
type Limiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// New creates a new rate limiter.
// maxTokens: maximum number of tokens
// refillRate: tokens added per second
func New(maxTokens float64, refillRate float64) *Limiter {
	return &Limiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed and consumes one token.
// Returns true if the request is allowed, false otherwise.
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.lastRefill).Seconds()
	l.tokens += elapsed * l.refillRate
	if l.tokens > l.maxTokens {
		l.tokens = l.maxTokens
	}
	l.lastRefill = now

	if l.tokens >= 1 {
		l.tokens--
		return true
	}
	return false
}

// Available returns the number of available tokens.
func (l *Limiter) Available() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.tokens
}

// Reset resets the rate limiter to full capacity.
func (l *Limiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tokens = l.maxTokens
	l.lastRefill = time.Now()
}
