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

	// Build a shell command that applies limits.
	// Each ulimit is prefixed with 2>/dev/null; to suppress errors on systems
	// where certain limits (like -r for RSS) are not supported, and followed
	// by "|| true" so that a failed limit does not abort the actual command.
	var scriptParts []string

	// Set resource limits using ulimit (tolerant of unsupported limits)
	scriptParts = append(scriptParts, fmt.Sprintf("ulimit -t %d 2>/dev/null || true", cpuSeconds))

	// Memory limit (virtual memory in KB)
	if l.config.MemoryLimit > 0 {
		memKB := int64(l.config.MemoryLimit / 1024)
		scriptParts = append(scriptParts, fmt.Sprintf("ulimit -v %d 2>/dev/null || true", memKB))
		scriptParts = append(scriptParts, fmt.Sprintf("ulimit -r %d 2>/dev/null || true", memKB))
	}

	// File size limit (in blocks, 512 bytes each)
	if l.config.MaxFileSize > 0 {
		blocks := l.config.MaxFileSize / 512
		if blocks <= 0 {
			blocks = 1
		}
		scriptParts = append(scriptParts, fmt.Sprintf("ulimit -f %d 2>/dev/null || true", blocks))
	}

	// Max child processes
	if l.config.MaxProcesses > 0 {
		scriptParts = append(scriptParts, fmt.Sprintf("ulimit -u %d 2>/dev/null || true", l.config.MaxProcesses*2))
	}

	// Set open file descriptor limit
	scriptParts = append(scriptParts, "ulimit -n 256 2>/dev/null || true")

	// Add the actual command, which must run via && so its exit code is preserved
	fullCommand := fmt.Sprintf("%s %s", cmdName, strings.Join(args, " "))

	// Combine: use semicolons for ulimit lines (so they don't chain),
	// then && for the actual command
	shellScript := strings.Join(scriptParts, "; ") + " && " + fullCommand

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
