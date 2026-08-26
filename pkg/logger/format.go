package logger

import (
	"fmt"
	"strings"
)

// formatEntry formats a log entry into a string.
func (l *Logger) formatEntry(timestamp string, level Level, msg string, args ...interface{}) string {
	var buf strings.Builder

	// Timestamp
	buf.WriteString("[")
	buf.WriteString(timestamp)
	buf.WriteString("] ")

	// Level with optional color
	if l.useColor {
		buf.WriteString(colorizeLevel(level))
	} else {
		buf.WriteString("[")
		buf.WriteString(level.String())
		buf.WriteString("]")
	}
	buf.WriteString(" ")

	// Prefix
	if l.prefix != "" {
		buf.WriteString("[")
		buf.WriteString(l.prefix)
		buf.WriteString("] ")
	}

	// Message
	if len(args) > 0 {
		buf.WriteString(fmt.Sprintf(msg, args...))
	} else {
		buf.WriteString(msg)
	}

	// Structured fields
	if len(l.fields) > 0 {
		buf.WriteString(" {")
		first := true
		for k, v := range l.fields {
			if !first {
				buf.WriteString(", ")
			}
			buf.WriteString(fmt.Sprintf("%s=%v", k, v))
			first = false
		}
		buf.WriteString("}")
	}

	buf.WriteString("\n")
	return buf.String()
}

// colorizeLevel returns a colorized string for the log level.
func colorizeLevel(level Level) string {
	var colorCode string
	switch level {
	case LevelDebug:
		colorCode = "\033[36m" // Cyan
	case LevelInfo:
		colorCode = "\033[32m" // Green
	case LevelWarn:
		colorCode = "\033[33m" // Yellow
	case LevelError:
		colorCode = "\033[31m" // Red
	case LevelFatal:
		colorCode = "\033[35m" // Magenta
	default:
		colorCode = "\033[0m"
	}
	return fmt.Sprintf("%s[%s]\033[0m", colorCode, level.String())
}

// NoColor disables colored output.
func (l *Logger) NoColor() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.useColor = false
}

// WithColor enables colored output (default).
func (l *Logger) WithColor() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.useColor = true
}
