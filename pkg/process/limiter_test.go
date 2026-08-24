package process

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestApplyLimits_PropagatesPerRequestOptions verifies that the per-request
// Timeout and MemoryLimit are reflected in the generated ulimit wrapper, rather
// than being silently dropped in favor of the limiter's default config. This is
// the regression that motivated the fix: before it, ApplyLimits read only the
// limiter's default config and always emitted ulimit -t 30 / -v 262144.
func TestApplyLimits_PropagatesPerRequestOptions(t *testing.T) {
	l := NewLimiter(DefaultLimiterConfig())

	opts := ExecuteOptions{
		Timeout:     5 * time.Second,
		MemoryLimit: 1 * 1024 * 1024, // 1MB
	}

	cmd, args, err := l.ApplyLimits(context.Background(), "python3", []string{"x.py"}, opts)
	if err != nil {
		t.Fatalf("ApplyLimits returned error: %v", err)
	}

	if cmd != "/bin/sh" {
		t.Fatalf("expected command /bin/sh, got %q", cmd)
	}
	if len(args) != 2 || args[0] != "-c" {
		t.Fatalf("expected args [\"-c\", <script>], got %v", args)
	}
	script := args[1]

	// CPU-time limit must come from opts.Timeout, not the default 30s.
	if !strings.Contains(script, "ulimit -t 5") {
		t.Errorf("script does not apply per-request CPU timeout (5s):\n%s", script)
	}
	if strings.Contains(script, "ulimit -t 30") {
		t.Errorf("script still uses the default 30s timeout instead of the request value:\n%s", script)
	}

	// Virtual-memory cap must come from opts.MemoryLimit (1MB = 1024KB), not the
	// default 256MB = 262144KB.
	if !strings.Contains(script, "ulimit -v 1024") {
		t.Errorf("script does not apply per-request memory limit (1024KB):\n%s", script)
	}
	if strings.Contains(script, "ulimit -v 262144") {
		t.Errorf("script still uses the default 256MB memory limit instead of the request value:\n%s", script)
	}

	// ulimit -r is the real-time-priority limit, not memory; it must not be emitted.
	if strings.Contains(script, "ulimit -r") {
		t.Errorf("script emits ulimit -r, which is not a memory limit:\n%s", script)
	}

	// The actual command must still be present at the end of the wrapper.
	if !strings.HasSuffix(script, "python3 x.py") {
		t.Errorf("script does not end with the wrapped command:\n%s", script)
	}
}

// TestApplyLimits_FallsBackToDefaultsWhenOptsEmpty ensures that a request
// without explicit limits still gets sane defaults rather than no limit at all.
func TestApplyLimits_FallsBackToDefaultsWhenOptsEmpty(t *testing.T) {
	l := NewLimiter(DefaultLimiterConfig())

	cmd, args, err := l.ApplyLimits(context.Background(), "python3", []string{"x.py"}, ExecuteOptions{})
	if err != nil {
		t.Fatalf("ApplyLimits returned error: %v", err)
	}

	if cmd != "/bin/sh" {
		t.Fatalf("expected command /bin/sh, got %q", cmd)
	}
	script := args[1]

	if !strings.Contains(script, "ulimit -t 30") {
		t.Errorf("default CPU timeout (30s) missing:\n%s", script)
	}
	if !strings.Contains(script, "ulimit -v 262144") {
		t.Errorf("default memory limit (256MB = 262144KB) missing:\n%s", script)
	}
}
