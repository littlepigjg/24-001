package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/codesandbox/codesandbox/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	var buf bytes.Buffer
	hasFailure := false

	t.Run("LevelFromString mapping", func(t *testing.T) {
		if got := logger.LevelFromString("error"); got != logger.LevelError {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatalf("LevelFromString('error') = %v, want LevelError", got)
		}
		if got := logger.LevelFromString("warn"); got != logger.LevelWarn {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatalf("LevelFromString('warn') = %v, want LevelWarn", got)
		}
		if got := logger.LevelFromString("debug"); got != logger.LevelDebug {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatalf("LevelFromString('debug') = %v, want LevelDebug", got)
		}
		if got := logger.LevelFromString("info"); got != logger.LevelInfo {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatalf("LevelFromString('info') = %v, want LevelInfo", got)
		}
		if got := logger.LevelFromString("fatal"); got != logger.LevelFatal {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatalf("LevelFromString('fatal') = %v, want LevelFatal", got)
		}
	})

	t.Run("Enabled filtering at LevelError", func(t *testing.T) {
		log := logger.NewLogger(&buf, logger.LevelError)

		if log.Enabled(logger.LevelDebug) {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Enabled(LevelDebug) should be false when level is ERROR")
		}
		if log.Enabled(logger.LevelInfo) {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Enabled(LevelInfo) should be false when level is ERROR")
		}
		if log.Enabled(logger.LevelWarn) {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Enabled(LevelWarn) should be false when level is ERROR")
		}
		if !log.Enabled(logger.LevelError) {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Enabled(LevelError) should be true when level is ERROR")
		}
		if !log.Enabled(logger.LevelFatal) {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Enabled(LevelFatal) should be true when level is ERROR")
		}
	})

	t.Run("Enabled filtering at LevelInfo", func(t *testing.T) {
		buf.Reset()
		log := logger.NewLogger(&buf, logger.LevelInfo)

		if log.Enabled(logger.LevelDebug) {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Enabled(LevelDebug) should be false when level is INFO")
		}
		if !log.Enabled(logger.LevelInfo) {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Enabled(LevelInfo) should be true when level is INFO")
		}
		if !log.Enabled(logger.LevelWarn) {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Enabled(LevelWarn) should be true when level is INFO")
		}
		if !log.Enabled(logger.LevelError) {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Enabled(LevelError) should be true when level is INFO")
		}
		if !log.Enabled(logger.LevelFatal) {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Enabled(LevelFatal) should be true when level is INFO")
		}
	})

	t.Run("Log output filtering at LevelError", func(t *testing.T) {
		buf.Reset()
		log := logger.NewLogger(&buf, logger.LevelError)

		log.Debug("debug-msg-N")
		log.Info("info-msg-N")
		log.Warn("warn-msg-N")
		log.Error("error-msg-Y")

		output := buf.String()

		if strings.Contains(output, "debug-msg-N") {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Debug message appeared despite ERROR level")
		}
		if strings.Contains(output, "info-msg-N") {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Info message appeared despite ERROR level")
		}
		if strings.Contains(output, "warn-msg-N") {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Warn message appeared despite ERROR level")
		}
		if !strings.Contains(output, "error-msg-Y") {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Error message did NOT appear at ERROR level")
		}
	})

	t.Run("Log output filtering at LevelWarn", func(t *testing.T) {
		buf.Reset()
		log := logger.NewLogger(&buf, logger.LevelWarn)

		log.Debug("debug-msg-N")
		log.Info("info-msg-N")
		log.Warn("warn-msg-Y")
		log.Error("error-msg-Y")

		output := buf.String()

		if strings.Contains(output, "debug-msg-N") {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Debug message appeared despite WARN level")
		}
		if strings.Contains(output, "info-msg-N") {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Info message appeared despite WARN level")
		}
		if !strings.Contains(output, "warn-msg-Y") {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Warn message did NOT appear at WARN level")
		}
		if !strings.Contains(output, "error-msg-Y") {
			t.Log("RED (红灯，缺陷未修复)")
			hasFailure = true
			t.Fatal("Error message did NOT appear at WARN level")
		}
	})

	if hasFailure {
		t.Log("RED (红灯，缺陷未修复)")
	} else {
		t.Log("GREEN (绿灯，缺陷已修复)")
	}
}