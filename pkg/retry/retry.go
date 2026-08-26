// Package retry provides retry logic with configurable backoff.
package retry

import (
	"context"
	"math"
	"time"
)

// Config holds retry configuration.
type Config struct {
	MaxAttempts int
	InitialWait time.Duration
	MaxWait     time.Duration
	Multiplier  float64
}

// DefaultConfig returns the default retry configuration.
func DefaultConfig() Config {
	return Config{
		MaxAttempts: 5,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     10 * time.Second,
		Multiplier:  2.0,
	}
}

// Do executes a function with retry logic.
func Do(ctx context.Context, config Config, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if attempt > 0 {
			// Calculate backoff with exponential increase
			wait := float64(config.InitialWait) * math.Pow(config.Multiplier, float64(attempt-1))
			if wait > float64(config.MaxWait) {
				wait = float64(config.MaxWait)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(wait)):
			}
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}
	}
	return lastErr
}
