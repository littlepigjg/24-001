package config

import (
	"path/filepath"
	"time"
)

type StorageConfig struct {
	urlFilePath  string
	logFilePath  string
	syncInterval time.Duration
	flushOnWrite bool
}

func (s *StorageConfig) URLFilePath(path string) {
	s.urlFilePath = path
}

func (s *StorageConfig) LogFilePath(path string) {
	s.logFilePath = path
}

func (s *StorageConfig) SyncInterval(d time.Duration) {
	s.syncInterval = d
}

func (s *StorageConfig) FlushOnWrite(b bool) {
	s.flushOnWrite = b
}

type Config struct {
	Storage StorageConfig
	BaseDir string
}

func Default() *Config {
	return &Config{
		Storage: StorageConfig{
			urlFilePath:  "urls.json",
			logFilePath:  "access.log",
			syncInterval: 10 * time.Second,
			flushOnWrite: true,
		},
		BaseDir: "",
	}
}

func (c *Config) FullURLFilePath() string {
	if c.BaseDir == "" {
		return c.Storage.urlFilePath
	}
	return filepath.Join(c.BaseDir, c.Storage.urlFilePath)
}

func (c *Config) FullLogFilePath() string {
	if c.BaseDir == "" {
		return c.Storage.logFilePath
	}
	return filepath.Join(c.BaseDir, c.Storage.logFilePath)
}

func (c *Config) GetSyncInterval() time.Duration {
	return c.Storage.syncInterval
}

func (c *Config) GetFlushOnWrite() bool {
	return c.Storage.flushOnWrite
}