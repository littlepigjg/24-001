package model

import (
	"os"
	"strconv"
	"time"
)

// AppConfig holds the application configuration.
type AppConfig struct {
	// Server settings
	Host         string `json:"host"`
	Port         int    `json:"port"`
	ReadTimeout  int    `json:"read_timeout"`  // seconds
	WriteTimeout int    `json:"write_timeout"` // seconds

	// Execution settings
	DefaultTimeout   int   `json:"default_timeout"`   // seconds
	MaxTimeout       int   `json:"max_timeout"`       // seconds
	DefaultMemory    int64 `json:"default_memory"`   // bytes
	MaxMemory        int64 `json:"max_memory"`       // bytes
	MaxCodeSize      int   `json:"max_code_size"`    // characters
	MaxOutputSize    int64 `json:"max_output_size"`  // bytes
	MaxConcurrent    int   `json:"max_concurrent"`   // max concurrent executions
	QueueSize        int   `json:"queue_size"`       // max pending executions

	// Storage settings
	StorageType   string `json:"storage_type"`   // "memory" or "file"
	DataDir       string `json:"data_dir"`       // directory for file storage
	MaxHistory    int    `json:"max_history"`    // max history records to keep
	MaxTemplates  int    `json:"max_templates"`  // max templates to keep

	// Logging settings
	LogLevel      string `json:"log_level"`      // debug, info, warn, error
	LogFormat     string `json:"log_format"`     // text or json
	LogFile       string `json:"log_file"`       // log file path (empty for stdout)

	// Sandbox settings
	SandboxDir    string `json:"sandbox_dir"`    // temp directory for sandbox
	CleanupInterval int  `json:"cleanup_interval"` // minutes

	// Feature flags
	EnableTemplates bool `json:"enable_templates"`
	EnableHistory   bool `json:"enable_history"`
	EnableStats     bool `json:"enable_stats"`
}

// DefaultConfig returns the default application configuration.
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Host:         "0.0.0.0",
		Port:         8080,
		ReadTimeout:  60,
		WriteTimeout: 60,

		DefaultTimeout:  30,
		MaxTimeout:      300,
		DefaultMemory:   256 * 1024 * 1024,
		MaxMemory:       1024 * 1024 * 1024,
		MaxCodeSize:     100000,
		MaxOutputSize:   10 * 1024 * 1024,
		MaxConcurrent:   10,
		QueueSize:       100,

		StorageType:   "memory",
		DataDir:       "./data",
		MaxHistory:    10000,
		MaxTemplates:  1000,

		LogLevel:      "info",
		LogFormat:     "text",
		LogFile:       "",

		SandboxDir:    "",
		CleanupInterval: 30,

		EnableTemplates: true,
		EnableHistory:   true,
		EnableStats:     true,
	}
}

// LoadFromEnv loads configuration from environment variables.
func (c *AppConfig) LoadFromEnv() {
	if v := os.Getenv("SANDBOX_HOST"); v != "" {
		c.Host = v
	}
	if v := os.Getenv("SANDBOX_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Port = port
		}
	}
	if v := os.Getenv("SANDBOX_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("SANDBOX_STORAGE"); v != "" {
		c.StorageType = v
	}
	if v := os.Getenv("SANDBOX_DATA_DIR"); v != "" {
		c.DataDir = v
	}
	if v := os.Getenv("SANDBOX_MAX_TIMEOUT"); v != "" {
		if t, err := strconv.Atoi(v); err == nil {
			c.MaxTimeout = t
		}
	}
}

// Address returns the full server address string.
func (c *AppConfig) Address() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

// GetReadTimeout returns the read timeout as a time.Duration.
func (c *AppConfig) GetReadTimeout() time.Duration {
	return time.Duration(c.ReadTimeout) * time.Second
}

// GetWriteTimeout returns the write timeout as a time.Duration.
func (c *AppConfig) GetWriteTimeout() time.Duration {
	return time.Duration(c.WriteTimeout) * time.Second
}

// GetDefaultTimeout returns the default execution timeout as a time.Duration.
func (c *AppConfig) GetDefaultTimeout() time.Duration {
	return time.Duration(c.DefaultTimeout) * time.Second
}
