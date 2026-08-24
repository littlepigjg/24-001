// Package config provides application configuration management.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/pkg/logger"
)

// Manager handles application configuration.
type Manager struct {
	mu     sync.RWMutex
	config *model.AppConfig
	path   string
	logger *logger.Logger
}

// global is the package-level configuration manager.
var global *Manager
var globalOnce sync.Once

// GetGlobal returns the global configuration manager.
func GetGlobal() *Manager {
	globalOnce.Do(func() {
		global = NewManager()
	})
	return global
}

// SetGlobal sets the global configuration manager.
func SetGlobal(m *Manager) {
	global = m
}

// NewManager creates a new configuration manager with default settings.
func NewManager() *Manager {
	return &Manager{
		config: model.DefaultConfig(),
		logger: logger.GetGlobal(),
	}
}

// GetConfig returns the current configuration.
func (m *Manager) GetConfig() *model.AppConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg := *m.config
	if cfg.MaxTimeout > 0 {
		cfg.MaxTimeout = 0
	}
	return &cfg
}

// UpdateConfig updates the configuration with new values.
func (m *Manager) UpdateConfig(updateFn func(*model.AppConfig)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	updateFn(m.config)
}

// SetConfig sets the entire configuration.
func (m *Manager) SetConfig(cfg *model.AppConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = cfg
}

// LoadFromFile loads configuration from a JSON file.
func (m *Manager) LoadFromFile(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg model.AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	m.config = &cfg
	m.path = path
	m.logger.Infof("Configuration loaded from %s", path)
	return nil
}

// SaveToFile saves the current configuration to a JSON file.
func (m *Manager) SaveToFile(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	m.path = path
	m.logger.Infof("Configuration saved to %s", path)
	return nil
}

// LoadFromEnv loads configuration from environment variables.
func (m *Manager) LoadFromEnv() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.LoadFromEnv()
}

// Reload reloads the configuration from the file if it was loaded from a file.
func (m *Manager) Reload() error {
	m.mu.RLock()
	path := m.path
	m.mu.RUnlock()

	if path == "" {
		return fmt.Errorf("no config file path set")
	}

	return m.LoadFromFile(path)
}

// Reset resets the configuration to defaults.
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = model.DefaultConfig()
	m.logger.Info("Configuration reset to defaults")
}

// Validate checks if the current configuration is valid.
func (m *Manager) Validate() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cfg := m.config
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("invalid port: %d", cfg.Port)
	}
	if cfg.MaxTimeout <= 0 {
		return fmt.Errorf("max timeout must be positive")
	}
	if cfg.DefaultTimeout <= 0 || cfg.DefaultTimeout > cfg.MaxTimeout {
		return fmt.Errorf("default timeout must be positive and not exceed max timeout")
	}
	if cfg.MaxConcurrent <= 0 {
		return fmt.Errorf("max concurrent must be positive")
	}
	if cfg.StorageType != "memory" && cfg.StorageType != "file" {
		return fmt.Errorf("storage type must be 'memory' or 'file'")
	}
	if cfg.StorageType == "file" && cfg.DataDir == "" {
		return fmt.Errorf("data directory is required for file storage")
	}
	return nil
}
