// Package config provides environment variable configuration loading.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// EnvConfig holds configuration from environment variables.
type EnvConfig struct {
	ServerPort     int
	ServerHost     string
	DataDir        string
	LogLevel       string
	MaxCodeLength  int
	MaxExecTime    time.Duration
	MaxMemoryMB    int
	MaxOutputBytes int
}

// LoadFromEnv loads configuration from environment variables.
func LoadFromEnv() *EnvConfig {
	cfg := &EnvConfig{
		ServerPort:     envInt("SANDBOX_PORT", 8080),
		ServerHost:     envStr("SANDBOX_HOST", "0.0.0.0"),
		DataDir:        envStr("SANDBOX_DATA_DIR", "./data"),
		LogLevel:       envStr("SANDBOX_LOG_LEVEL", "info"),
		MaxCodeLength:  envInt("SANDBOX_MAX_CODE_LENGTH", 100000),
		MaxExecTime:    envDuration("SANDBOX_MAX_EXEC_TIME", 30*time.Second),
		MaxMemoryMB:    envInt("SANDBOX_MAX_MEMORY_MB", 256),
		MaxOutputBytes: envInt("SANDBOX_MAX_OUTPUT_BYTES", 1048576),
	}
	return cfg
}

func envStr(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func envInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			return n
		}
	}
	return defaultVal
}

func envDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(strings.TrimSpace(val)); err == nil {
			return d
		}
	}
	return defaultVal
}
