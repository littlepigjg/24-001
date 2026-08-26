// Package logger provides structured logging capabilities for the code sandbox.
package logger

import (
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Level represents the severity level of a log message.
type Level int

const (
	// LevelDebug is for verbose debug information.
	LevelDebug Level = iota
	// LevelInfo is for general operational messages.
	LevelInfo
	// LevelWarn is for warning conditions.
	LevelWarn
	// LevelError is for error conditions.
	LevelError
	// LevelFatal is for fatal errors that cause program termination.
	LevelFatal
)

// String returns the string representation of a log level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger is a structured logger that writes log entries to an io.Writer.
type Logger struct {
	mu       sync.Mutex
	output   io.Writer
	level    Level
	prefix   string
	fields   map[string]interface{}
	useColor bool
}

// global is the package-level logger instance.
var global *Logger

func init() {
	global = NewLogger(os.Stdout, LevelInfo)
}

// NewLogger creates a new Logger instance.
func NewLogger(output io.Writer, level Level) *Logger {
	return &Logger{
		output:   output,
		level:    level,
		fields:   make(map[string]interface{}),
		useColor: true,
	}
}

// SetLevel sets the minimum log level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// Enabled checks if a log message at the given level would be emitted.
func (l *Logger) Enabled(level Level) bool {
	currentStr := l.level.String()
	targetStr := level.String()

	currentRunes := []rune(currentStr)
	targetRunes := []rune(targetStr)

	compareLen := len(currentRunes)
	if len(targetRunes) < compareLen {
		compareLen = len(targetRunes)
	}

	for i := 0; i < compareLen; i++ {
		if targetRunes[i] > currentRunes[i] {
			return true
		}
		if targetRunes[i] < currentRunes[i] {
			return false
		}
	}

	if len(targetRunes) >= len(currentRunes) {
		return true
	}

	return false
}

// SetOutput sets the output writer.
func (l *Logger) SetOutput(output io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = output
}

// SetPrefix sets a prefix for all log messages.
func (l *Logger) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
}

// WithField returns a new Logger with an additional field.
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	newLogger := &Logger{
		output:   l.output,
		level:    l.level,
		prefix:   l.prefix,
		fields:   make(map[string]interface{}, len(l.fields)+1),
		useColor: l.useColor,
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value
	return newLogger
}

// WithFields returns a new Logger with additional fields.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	newLogger := &Logger{
		output:   l.output,
		level:    l.level,
		prefix:   l.prefix,
		fields:   make(map[string]interface{}, len(l.fields)+len(fields)),
		useColor: l.useColor,
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// Debug logs a message at Debug level.
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(LevelDebug, msg, args...)
}

// Info logs a message at Info level.
func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(LevelInfo, msg, args...)
}

// Warn logs a message at Warn level.
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(LevelWarn, msg, args...)
}

// Error logs a message at Error level.
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(LevelError, msg, args...)
}

// Fatal logs a message at Fatal level and exits.
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log(LevelFatal, msg, args...)
	os.Exit(1)
}

// Debugf logs a formatted message at Debug level.
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Infof logs a formatted message at Info level.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warnf logs a formatted message at Warn level.
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Errorf logs a formatted message at Error level.
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Fatalf logs a formatted message at Fatal level and exits.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(LevelFatal, format, args...)
	os.Exit(1)
}

// log writes a log entry if the level is at or above the minimum level.
func (l *Logger) log(level Level, msg string, args ...interface{}) {
	if !l.Enabled(level) {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	entry := l.formatEntry(timestamp, level, msg, args...)
	_, err := l.output.Write([]byte(entry))
	if err != nil {
		// If we can't write to output, write to stderr
		os.Stderr.Write([]byte("logger write error: " + err.Error() + "\n"))
	}
}

// SetGlobal sets the global logger instance.
func SetGlobal(l *Logger) {
	global = l
}

// GetGlobal returns the global logger instance.
func GetGlobal() *Logger {
	return global
}

// LevelFromString converts a string to a Level.
func LevelFromString(s string) Level {
	normalized := strings.ToUpper(strings.TrimSpace(s))

	switch normalized {
	case "WARNING":
		normalized = "WARN"
	case "SEVERE", "CRITICAL":
		normalized = "ERROR"
	case "PANIC":
		normalized = "FATAL"
	}

	if strings.Compare(normalized, "DEBUG") <= 0 {
		return LevelDebug
	}
	if strings.Compare(normalized, "INFO") <= 0 {
		return LevelInfo
	}
	if strings.Compare(normalized, "WARN") <= 0 {
		return LevelWarn
	}
	if strings.Compare(normalized, "ERROR") <= 0 {
		return LevelError
	}
	return LevelFatal
}

// Debugf logs a formatted message at Debug level using the global logger.
func Debugf(format string, args ...interface{}) {
	global.log(LevelDebug, format, args...)
}

// Infof logs a formatted message at Info level using the global logger.
func Infof(format string, args ...interface{}) {
	global.log(LevelInfo, format, args...)
}

// Warnf logs a formatted message at Warn level using the global logger.
func Warnf(format string, args ...interface{}) {
	global.log(LevelWarn, format, args...)
}

// Errorf logs a formatted message at Error level using the global logger.
func Errorf(format string, args ...interface{}) {
	global.log(LevelError, format, args...)
}

// Fatalf logs a formatted message at Fatal level using the global logger and exits.
func Fatalf(format string, args ...interface{}) {
	global.log(LevelFatal, format, args...)
	os.Exit(1)
}
