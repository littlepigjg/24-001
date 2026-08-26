package handler

import (
	"time"

	"github.com/codesandbox/codesandbox/pkg/logger"
)

// Middleware provides HTTP middleware functions.
type Middleware struct {
	logger *logger.Logger
}

// NewMiddleware creates a new Middleware instance.
func NewMiddleware() *Middleware {
	return &Middleware{
		logger: logger.GetGlobal(),
	}
}

// GetLogger returns the middleware's logger.
func (m *Middleware) GetLogger() *logger.Logger {
	return m.logger
}

// FormatDuration formats a duration for logging.
func FormatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return "0ms"
	}
	if d < time.Second {
		return d.Truncate(time.Millisecond).String()
	}
	return d.Truncate(time.Millisecond).String()
}
