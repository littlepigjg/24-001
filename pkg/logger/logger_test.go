package logger

import (
	"testing"
)

// TestEnabled_LevelFiltering guards against the severity-comparison regression
// where log levels were compared by their string names rather than by numeric
// severity. With level=Error, Info must NOT leak and Error/Fatal must pass;
// with level=Warn, Error must NOT be swallowed.
func TestEnabled_LevelFiltering(t *testing.T) {
	cases := []struct {
		name     string
		min      Level
		target   Level
		enabled  bool
	}{
		{"error level: info leaks", LevelError, LevelInfo, false},
		{"error level: debug leaks", LevelError, LevelDebug, false},
		{"error level: warn leaks", LevelError, LevelWarn, false},
		{"error level: error passes", LevelError, LevelError, true},
		{"error level: fatal passes", LevelError, LevelFatal, true},

		{"warn level: error swallowed", LevelWarn, LevelError, true},
		{"warn level: warn passes", LevelWarn, LevelWarn, true},
		{"warn level: fatal passes", LevelWarn, LevelFatal, true},
		{"warn level: info leaks", LevelWarn, LevelInfo, false},

		{"debug level: everything passes", LevelDebug, LevelDebug, true},
		{"info level: debug filtered", LevelInfo, LevelDebug, false},
		{"info level: info passes", LevelInfo, LevelInfo, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := NewLogger(discard{}, c.min)
			if got := l.Enabled(c.target); got != c.enabled {
				t.Fatalf("Enabled(min=%s, target=%s) = %v, want %v",
					c.min, c.target, got, c.enabled)
			}
		})
	}
}

// TestLevelFromString pins the mapping so that "error"/"fatal" map to their
// true severity rather than collapsing onto Info via alphabetical comparison.
func TestLevelFromString(t *testing.T) {
	cases := []struct {
		in   string
		want Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"trace", LevelDebug},
		{"info", LevelInfo},
		{"warn", LevelWarn},
		{"warning", LevelWarn},
		{"WARNING", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"severe", LevelError},
		{"critical", LevelError},
		{"fatal", LevelFatal},
		{"FATAL", LevelFatal},
		{"panic", LevelFatal},
		{"  error  ", LevelError},
		{"", LevelInfo},      // unknown -> safe default
		{"bogus", LevelInfo}, // unknown -> safe default
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := LevelFromString(c.in); got != c.want {
				t.Fatalf("LevelFromString(%q) = %s, want %s", c.in, got, c.want)
			}
		})
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
