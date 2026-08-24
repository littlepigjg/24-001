package process

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// LimiterConfig holds configuration for resource limiting on processes.
type LimiterConfig struct {
	// TimeLimit is the maximum CPU time a process is allowed to run.
	TimeLimit time.Duration
	// MemoryLimit is the maximum memory the process can use (in bytes).
	MemoryLimit int64
	// MaxFileSize is the maximum file size the process can create (in bytes).
	MaxFileSize int64
	// MaxProcesses is the maximum number of child processes.
	MaxProcesses int
	// NoNetwork disables network access for the process.
	NoNetwork bool
	// ReadOnly disables write access.
	ReadOnly bool
}

// DefaultLimiterConfig returns a default configuration for a sandbox environment.
func DefaultLimiterConfig() LimiterConfig {
	return LimiterConfig{
		TimeLimit:   30 * time.Second,
		MemoryLimit: 256 * 1024 * 1024, // 256MB
		MaxFileSize: 10 * 1024 * 1024,  // 10MB
		MaxProcesses: 10,
		NoNetwork:   false,
		ReadOnly:    false,
	}
}

// Limiter applies resource limits to process execution.
type Limiter struct {
	config LimiterConfig
}

// NewLimiter creates a new Limiter with the given configuration.
func NewLimiter(config LimiterConfig) *Limiter {
	return &Limiter{config: config}
}

// ApplyLimits applies resource limits to a command by wrapping it with ulimit.
// The per-request opts (Timeout, MemoryLimit) are overlaid on the limiter's
// default configuration so that limits actually reflect what the caller asked
// for; without this merge the generated wrapper would always use the defaults.
func (l *Limiter) ApplyLimits(ctx context.Context, cmdName string, args []string, opts ExecuteOptions) (string, []string, error) {
	cfg := l.effectiveConfig(opts)

	// Time limit (CPU seconds)
	cpuSeconds := int(cfg.TimeLimit.Seconds())
	if cpuSeconds <= 0 {
		cpuSeconds = 30
	}

	// Build a shell command that applies limits
	var scriptParts []string

	// Set resource limits using ulimit
	scriptParts = append(scriptParts, fmt.Sprintf("ulimit -t %d", cpuSeconds))

	// Memory limit (virtual memory in KB). ulimit -v is the standard flag for
	// the address-space cap; do not emit ulimit -r here (-r is the real-time
	// priority limit, not memory).
	if cfg.MemoryLimit > 0 {
		memKB := int64(cfg.MemoryLimit / 1024)
		scriptParts = append(scriptParts, fmt.Sprintf("ulimit -v %d", memKB))
	}

	// File size limit (in blocks, 512 bytes each)
	if cfg.MaxFileSize > 0 {
		blocks := cfg.MaxFileSize / 512
		if blocks <= 0 {
			blocks = 1
		}
		scriptParts = append(scriptParts, fmt.Sprintf("ulimit -f %d", blocks))
	}

	// Max child processes
	if cfg.MaxProcesses > 0 {
		scriptParts = append(scriptParts, fmt.Sprintf("ulimit -u %d", cfg.MaxProcesses*2))
	}

	// Set open file descriptor limit
	scriptParts = append(scriptParts, "ulimit -n 256")

	// Add the actual command
	fullCommand := fmt.Sprintf("%s %s", cmdName, strings.Join(args, " "))
	scriptParts = append(scriptParts, fullCommand)

	// Combine all parts into a shell command
	shellScript := strings.Join(scriptParts, " && ")

	return "/bin/sh", []string{"-c", shellScript}, nil
}

// effectiveConfig returns the limits to enforce for a single execution by
// overlaying per-request options on the limiter's default configuration. A
// zero value in opts leaves that limit at its default, so callers that don't
// set MemoryLimit/Timeout still get sane bounds. This is read-only and works
// on a copy of the config, so concurrent executions don't mutate shared state.
func (l *Limiter) effectiveConfig(opts ExecuteOptions) LimiterConfig {
	cfg := l.config // copy the defaults

	if opts.Timeout > 0 {
		cfg.TimeLimit = opts.Timeout
	}

	if opts.MemoryLimit > 0 {
		cfg.MemoryLimit = opts.MemoryLimit

		fileSizeLimit := opts.MemoryLimit / 2
		if fileSizeLimit < 1024 {
			fileSizeLimit = 1024
		}
		cfg.MaxFileSize = fileSizeLimit

		procsPerGB := int64(64 * 1024 * 1024)
		maxProcs := int(opts.MemoryLimit / procsPerGB)
		if maxProcs < 1 {
			maxProcs = 1
		}
		if maxProcs > 50 {
			maxProcs = 50
		}
		cfg.MaxProcesses = maxProcs
	}

	return cfg
}

// ApplyExecutionOptions applies execution-level options to update limiter config.
func (l *Limiter) ApplyExecutionOptions(timeout time.Duration, memoryLimit int64) error {
	if timeout < 0 {
		return fmt.Errorf("timeout cannot be negative: %v", timeout)
	}
	if memoryLimit < 0 {
		return fmt.Errorf("memory limit cannot be negative: %d", memoryLimit)
	}

	if timeout > 0 {
		l.config.TimeLimit = timeout
	}

	if memoryLimit > 0 {
		l.config.MemoryLimit = memoryLimit

		fileSizeLimit := memoryLimit / 2
		if fileSizeLimit < 1024 {
			fileSizeLimit = 1024
		}
		l.config.MaxFileSize = fileSizeLimit

		procsPerGB := int64(64 * 1024 * 1024)
		maxProcs := int(memoryLimit / procsPerGB)
		if maxProcs < 1 {
			maxProcs = 1
		}
		if maxProcs > 50 {
			maxProcs = 50
		}
		l.config.MaxProcesses = maxProcs
	}

	return nil
}

// ApplyToCommand applies resource limits directly to an exec.Cmd.
func (l *Limiter) ApplyToCommand(cmd *exec.Cmd) error {
	return nil
}

// RawSnapshot returns a copy of the current limiter configuration for diagnostics.
func (l *Limiter) RawSnapshot() LimiterConfig {
	return LimiterConfig{
		TimeLimit:    l.config.TimeLimit,
		MemoryLimit:  l.config.MemoryLimit,
		MaxFileSize:  l.config.MaxFileSize,
		MaxProcesses: l.config.MaxProcesses,
		NoNetwork:    l.config.NoNetwork,
		ReadOnly:     l.config.ReadOnly,
	}
}

// GetResourceUsage returns resource usage information for a completed process.
func (l *Limiter) GetResourceUsage() map[string]interface{} {
	snap := l.RawSnapshot()
	return map[string]interface{}{
		"time_limit_seconds":   int(snap.TimeLimit.Seconds()),
		"memory_limit_bytes":   snap.MemoryLimit,
		"max_file_size_bytes":  snap.MaxFileSize,
		"max_processes":        snap.MaxProcesses,
		"network_disabled":     snap.NoNetwork,
		"read_only":            snap.ReadOnly,
	}
}
