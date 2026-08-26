package process

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// TestRunWithTimeout_TimedOut verifies that a long-running command that exceeds
// its timeout is reported as timed out with a sentinel exit code.
func TestRunWithTimeout_TimedOut(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based timeout test is unix-only")
	}
	r := NewRunner()

	result, err := r.RunWithTimeout(context.Background(), 200*time.Millisecond, "/bin/sh", "-c", "sleep 5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.TimedOut {
		t.Errorf("expected TimedOut=true, got false")
	}
	if result.ExitCode != -1 {
		t.Errorf("expected ExitCode=-1, got %d", result.ExitCode)
	}
	if result.IsSuccess() {
		t.Errorf("timed-out result should not be considered a success")
	}
	// Must return promptly after the timeout, not after the full sleep.
	if result.Duration > 2*time.Second {
		t.Errorf("timeout not enforced promptly: duration=%v (expected ~200ms)", result.Duration)
	}
}

// TestRunWithTimeout_KillsProcessGroup verifies that a timeout kills the whole
// process group, so a wrapped grandchild (e.g. `sleep` under `/bin/sh -c`) does
// not get orphaned and block cmd.Run() until it exits on its own.
func TestRunWithTimeout_KillsProcessGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-group test is unix-only")
	}
	r := NewRunner()

	start := time.Now()
	result, err := r.RunWithTimeout(context.Background(), 300*time.Millisecond, "/bin/sh", "-c", "sleep 30")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.TimedOut {
		t.Fatalf("expected TimedOut=true, got false (duration=%v)", result.Duration)
	}
	// The whole group must be dead shortly after the timeout. If the orphan
	// survived, this would take ~30s.
	if elapsed > 3*time.Second {
		t.Errorf("process group not killed on timeout: elapsed=%v", elapsed)
	}
}

// TestRunWithTimeout_Success verifies a normal successful command.
func TestRunWithTimeout_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based success test is unix-only")
	}
	r := NewRunner()

	result, err := r.RunWithTimeout(context.Background(), 5*time.Second, "/bin/sh", "-c", "echo hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TimedOut {
		t.Errorf("expected TimedOut=false, got true")
	}
	if result.ExitCode != 0 {
		t.Errorf("expected ExitCode=0, got %d", result.ExitCode)
	}
	if result.Stdout != "hi\n" {
		t.Errorf("expected stdout %q, got %q", "hi\n", result.Stdout)
	}
}

// TestRunWithTimeout_NonZeroExit verifies a non-zero exit code is captured.
func TestRunWithTimeout_NonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based exit-code test is unix-only")
	}
	r := NewRunner()

	result, err := r.RunWithTimeout(context.Background(), 5*time.Second, "/bin/sh", "-c", "exit 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TimedOut {
		t.Errorf("expected TimedOut=false, got true")
	}
	if result.ExitCode != 1 {
		t.Errorf("expected ExitCode=1, got %d", result.ExitCode)
	}
	if result.IsSuccess() {
		t.Errorf("non-zero exit result should not be considered a success")
	}
}

// TestRunWithTimeout_ZeroTimeoutFallsBackToDefault verifies that a zero timeout
// is guarded rather than producing an immediately-expired context.
func TestRunWithTimeout_ZeroTimeoutFallsBackToDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based zero-timeout test is unix-only")
	}
	r := NewRunner()

	result, err := r.RunWithTimeout(context.Background(), 0, "/bin/sh", "-c", "echo ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TimedOut {
		t.Errorf("zero timeout should fall back to default, not time out; got TimedOut=true")
	}
	if result.ExitCode != 0 {
		t.Errorf("expected ExitCode=0, got %d", result.ExitCode)
	}
	if result.Stdout != "ok\n" {
		t.Errorf("expected stdout %q, got %q", "ok\n", result.Stdout)
	}
}
