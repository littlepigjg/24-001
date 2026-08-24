package config

import (
	"time"
)

// StorageConfig holds storage-related configuration.
type StorageConfig struct {
	urlFilePath    string
	logFilePath    string
	syncInterval   time.Duration
	flushOnWrite   bool
}

// URLConfig holds URL shortener-specific configuration.
type URLConfig struct {
	Storage StorageConfig
}

// DefaultURLConfig returns default URL configuration.
func DefaultURLConfig() *URLConfig {
	return &URLConfig{
		Storage: StorageConfig{
			urlFilePath:  "./data/urls.json",
			logFilePath:  "./data/access.log",
			syncInterval: 5 * time.Second,
			flushOnWrite:  true,
		},
	}
}

// URLFilePath sets the URL data file path.
func (s *StorageConfig) URLFilePath(path string) {
	s.urlFilePath = path
}

// LogFilePath sets the access log file path.
func (s *StorageConfig) LogFilePath(path string) {
	s.logFilePath = path
}

// SyncInterval sets the sync interval.
func (s *StorageConfig) SyncInterval(d time.Duration) {
	s.syncInterval = d
}

// FlushOnWrite sets whether to flush on write.
func (s *StorageConfig) FlushOnWrite(b bool) {
	s.flushOnWrite = b
}

// GetURLFilePath returns the URL data file path.
func (s *StorageConfig) GetURLFilePath() string {
	return s.urlFilePath
}

// GetLogFilePath returns the access log file path.
func (s *StorageConfig) GetLogFilePath() string {
	return s.logFilePath
}

// GetSyncInterval returns the sync interval.
func (s *StorageConfig) GetSyncInterval() time.Duration {
	return s.syncInterval
}

// GetFlushOnWrite returns whether to flush on write.
func (s *StorageConfig) GetFlushOnWrite() bool {
	return s.flushOnWrite
}
