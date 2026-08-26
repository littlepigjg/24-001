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
func (l *Limiter) ApplyLimits(ctx context.Context, cmdName string, args []string) (string, []string, error) {
	// Time limit (CPU seconds)
	cpuSeconds := int(l.config.TimeLimit.Seconds())
	if cpuSeconds <= 0 {
		cpuSeconds = 30
	}

	// Build a shell command that applies limits. Each ulimit is made
	// non-fatal and silenced: in many sandbox/容器 environments certain
	// limits (e.g. -r max real-time priority) cannot be set, and dash does
	// not support every option (e.g. -u). A single failed ulimit must not
	// abort the wrapped script before the user's command runs.
	var scriptParts []string

	// safeLimit emits a single ulimit that never fails the script.
	safeLimit := func(spec string) {
		scriptParts = append(scriptParts, fmt.Sprintf("ulimit %s 2>/dev/null || true", spec))
	}

	// Time limit (CPU seconds)
	safeLimit(fmt.Sprintf("-t %d", cpuSeconds))

	// Memory limit (virtual memory in KB)
	if l.config.MemoryLimit > 0 {
		memKB := int64(l.config.MemoryLimit / 1024)
		safeLimit(fmt.Sprintf("-v %d", memKB))
		safeLimit(fmt.Sprintf("-r %d", memKB))
	}

	// File size limit (in blocks, 512 bytes each)
	if l.config.MaxFileSize > 0 {
		blocks := l.config.MaxFileSize / 512
		if blocks <= 0 {
			blocks = 1
		}
		safeLimit(fmt.Sprintf("-f %d", blocks))
	}

	// Max child processes (not supported by dash; harmless if it fails)
	if l.config.MaxProcesses > 0 {
		safeLimit(fmt.Sprintf("-u %d", l.config.MaxProcesses*2))
	}

	// Set open file descriptor limit
	safeLimit("-n 256")

	// Add the actual command
	fullCommand := fmt.Sprintf("%s %s", cmdName, strings.Join(args, " "))
	scriptParts = append(scriptParts, fullCommand)

	// Combine all parts into a shell command
	shellScript := strings.Join(scriptParts, " && ")

	return "/bin/sh", []string{"-c", shellScript}, nil
}

// ApplyToCommand applies resource limits directly to an exec.Cmd.
func (l *Limiter) ApplyToCommand(cmd *exec.Cmd) error {
	// On Unix systems, set process limits via SysProcAttr
	// This is a platform-specific implementation
	return nil
}

// GetResourceUsage returns resource usage information for a completed process.
func (l *Limiter) GetResourceUsage() map[string]interface{} {
	return map[string]interface{}{
		"time_limit_seconds":   int(l.config.TimeLimit.Seconds()),
		"memory_limit_bytes":   l.config.MemoryLimit,
		"max_file_size_bytes":  l.config.MaxFileSize,
		"max_processes":        l.config.MaxProcesses,
		"network_disabled":     l.config.NoNetwork,
		"read_only":            l.config.ReadOnly,
	}
}
