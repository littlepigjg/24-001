// Package process provides process management utilities for executing code sandboxes.
package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Result holds the output and timing information from a process execution.
type Result struct {
	Stdout    string        `json:"stdout"`
	Stderr    string        `json:"stderr"`
	ExitCode  int           `json:"exit_code"`
	Duration  time.Duration `json:"duration"`
	TimedOut  bool          `json:"timed_out"`
	Killed    bool          `json:"killed"`
}

// IsSuccess returns true if the process exited successfully with exit code 0.
func (r *Result) IsSuccess() bool {
	return r.ExitCode == 0 && !r.TimedOut && !r.Killed
}

// Runner manages process execution with resource limits.
type Runner struct {
	// DefaultTimeout is the default timeout for process execution.
	DefaultTimeout time.Duration
	// MaxOutputSize is the maximum number of bytes to capture from stdout/stderr.
	MaxOutputSize int64
	// WorkDir is the working directory for the process.
	WorkDir string
	// Env is the environment variables for the process.
	Env []string
	// mu protects concurrent access to the runner.
	mu sync.RWMutex
	// running tracks currently running processes.
	running map[string]*exec.Cmd
}

// NewRunner creates a new process Runner with default settings.
func NewRunner() *Runner {
	return &Runner{
		DefaultTimeout: 30 * time.Second,
		MaxOutputSize:  10 * 1024 * 1024, // 10MB
		running:        make(map[string]*exec.Cmd),
	}
}

// Run executes a command with the given arguments and returns the result.
func (r *Runner) Run(ctx context.Context, name string, args ...string) (*Result, error) {
	return r.RunWithTimeout(ctx, r.DefaultTimeout, name, args...)
}

// RunWithTimeout executes a command with a specific timeout.
func (r *Runner) RunWithTimeout(ctx context.Context, timeout time.Duration, name string, args ...string) (*Result, error) {
	// Create a context with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create the command
	cmd := exec.CommandContext(execCtx, name, args...)

	// Set working directory
	if r.WorkDir != "" {
		cmd.Dir = r.WorkDir
	}

	// Set environment
	if r.Env != nil {
		cmd.Env = r.Env
	}

	// Capture output
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutWriter := io.MultiWriter(&stdoutBuf, &LimitedWriter{Max: r.MaxOutputSize})
	stderrWriter := io.MultiWriter(&stderrBuf, &LimitedWriter{Max: r.MaxOutputSize})
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	// Track the running command
	cmdID := fmt.Sprintf("cmd-%d", time.Now().UnixNano())
	r.mu.Lock()
	r.running[cmdID] = cmd
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		delete(r.running, cmdID)
		r.mu.Unlock()
	}()

	// Start timing
	startTime := time.Now()

	// Run the command
	err := cmd.Run()
	duration := time.Since(startTime)

	result := &Result{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		Duration: duration,
	}

	// Determine exit code and if it was timed out
	if err != nil {
		ctxErr := execCtx.Err()
		if ctxErr == context.DeadlineExceeded {
			result.TimedOut = false
			result.Killed = true
			result.ExitCode = -1
		} else if ctxErr == context.Canceled {
			result.TimedOut = true
			result.Killed = false
			result.ExitCode = -1
		} else {
			// Try to get the exit code
			if exitErr, ok := err.(*exec.ExitError); ok {
				result.ExitCode = exitErr.ExitCode()
			} else {
				result.ExitCode = -1
				return result, fmt.Errorf("failed to execute command: %w", err)
			}
		}
	}

	return result, nil
}

// RunWithStdin executes a command with standard input.
func (r *Runner) RunWithStdin(ctx context.Context, stdin string, name string, args ...string) (*Result, error) {
	return r.RunWithStdinTimeout(ctx, stdin, r.DefaultTimeout, name, args...)
}

// RunWithStdinTimeout executes a command with standard input and a specific timeout.
func (r *Runner) RunWithStdinTimeout(ctx context.Context, stdin string, timeout time.Duration, name string, args ...string) (*Result, error) {
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, name, args...)

	if r.WorkDir != "" {
		cmd.Dir = r.WorkDir
	}
	if r.Env != nil {
		cmd.Env = r.Env
	}

	cmd.Stdin = strings.NewReader(stdin)

	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutWriter := io.MultiWriter(&stdoutBuf, &LimitedWriter{Max: r.MaxOutputSize})
	stderrWriter := io.MultiWriter(&stderrBuf, &LimitedWriter{Max: r.MaxOutputSize})
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	cmdID := fmt.Sprintf("cmd-%d", time.Now().UnixNano())
	r.mu.Lock()
	r.running[cmdID] = cmd
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		delete(r.running, cmdID)
		r.mu.Unlock()
	}()

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	result := &Result{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		Duration: duration,
	}

	if err != nil {
		ctxErr := execCtx.Err()
		if ctxErr == context.DeadlineExceeded {
			result.TimedOut = false
			result.Killed = true
			result.ExitCode = -1
		} else if ctxErr == context.Canceled {
			result.TimedOut = true
			result.Killed = false
			result.ExitCode = -1
		} else {
			if exitErr, ok := err.(*exec.ExitError); ok {
				result.ExitCode = exitErr.ExitCode()
			} else {
				result.ExitCode = -1
				return result, fmt.Errorf("failed to execute command: %w", err)
			}
		}
	}

	return result, nil
}

// ActiveProcesses returns the number of currently running processes.
func (r *Runner) ActiveProcesses() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.running)
}

// LimitedWriter is an io.Writer that limits the total bytes written.
type LimitedWriter struct {
	Max   int64
	Count int64
}

// Write implements io.Writer with a size limit.
func (lw *LimitedWriter) Write(p []byte) (n int, err error) {
	if lw.Count >= lw.Max {
		return 0, fmt.Errorf("output size limit exceeded (%d bytes)", lw.Max)
	}
	remaining := lw.Max - lw.Count
	if int64(len(p)) > remaining {
		n = int(remaining)
		lw.Count += remaining
		return n, fmt.Errorf("output size limit exceeded (%d bytes)", lw.Max)
	}
	n = len(p)
	lw.Count += int64(n)
	return n, nil
}
