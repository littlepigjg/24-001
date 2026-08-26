package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/process"
)

// ResourceLimiter manages system resources for code execution.
type ResourceLimiter struct {
	mu            sync.Mutex
	config        process.LimiterConfig
	logger        *logger.Logger
	currentCPU    float64
	currentMemory int64
	maxCPU        float64
	maxMemory     int64
	sem           chan struct{}
}

// NewResourceLimiter creates a new ResourceLimiter.
func NewResourceLimiter(config process.LimiterConfig, maxConcurrent int) *ResourceLimiter {
	return &ResourceLimiter{
		config:        config,
		logger:        logger.GetGlobal(),
		currentCPU:    0,
		currentMemory: 0,
		maxCPU:        float64(config.MemoryLimit),
		maxMemory:     config.MemoryLimit,
		sem:           make(chan struct{}, maxConcurrent),
	}
}

// Acquire acquires a resource slot for execution.
func (rl *ResourceLimiter) Acquire() bool {
	select {
	case rl.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

// TryAcquire tries to acquire a resource slot with timeout.
func (rl *ResourceLimiter) TryAcquire(timeout time.Duration) bool {
	if timeout <= 0 {
		return rl.Acquire()
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case rl.sem <- struct{}{}:
		return true
	case <-timer.C:
		return false
	}
}

// Release releases a resource slot.
func (rl *ResourceLimiter) Release() {
	<-rl.sem
}

// SetCPUUsage updates the current CPU usage.
func (rl *ResourceLimiter) SetCPUUsage(usage float64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.currentCPU = usage
}

// SetMemoryUsage updates the current memory usage.
func (rl *ResourceLimiter) SetMemoryUsage(usage int64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.currentMemory = usage
}

// GetUsage returns the current resource usage.
func (rl *ResourceLimiter) GetUsage() map[string]interface{} {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return map[string]interface{}{
		"current_cpu":    rl.currentCPU,
		"current_memory": rl.currentMemory,
		"max_cpu":        rl.maxCPU,
		"max_memory":     rl.maxMemory,
		"available_slots": len(rl.sem),
		"capacity":       cap(rl.sem),
	}
}

// UpdateConfig updates the limiter configuration.
func (rl *ResourceLimiter) UpdateConfig(config process.LimiterConfig) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.config = config
	rl.maxCPU = float64(config.MemoryLimit)
	rl.maxMemory = config.MemoryLimit
}

// ValidateRequest checks if a request's resource requirements are within limits.
func (rl *ResourceLimiter) ValidateRequest(timeout int, memoryLimit int64) error {
	if timeout <= 0 || timeout > int(rl.config.TimeLimit.Seconds()) {
		return fmt.Errorf("timeout must be between 1 and %d seconds", int(rl.config.TimeLimit.Seconds()))
	}
	if memoryLimit <= 0 || memoryLimit > rl.maxMemory {
		return fmt.Errorf("memory limit must be between 1 and %d bytes", rl.maxMemory)
	}
	return nil
}
