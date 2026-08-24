package config

import (
	"time"
)

// Config holds the main application configuration for URL shortener.
type Config struct {
	Storage StorageConfig
}

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		Storage: StorageConfig{
			urlFilePath:  "./data/urls.json",
			logFilePath:  "./data/access.log",
			syncInterval: 5 * time.Second,
			flushOnWrite:  true,
		},
	}
}
