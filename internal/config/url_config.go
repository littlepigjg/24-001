package config

import (
	"sync"
	"time"
)

type Config struct {
	Storage StorageConfig
}

type StorageConfig struct {
	urlFilePath  string
	logFilePath  string
	syncInterval time.Duration
	flushOnWrite bool
	mu           sync.RWMutex
}

func Default() *Config {
	return &Config{
		Storage: StorageConfig{
			syncInterval: 5 * time.Second,
			flushOnWrite: true,
		},
	}
}

func (s *StorageConfig) URLFilePath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urlFilePath = path
}

func (s *StorageConfig) GetURLFilePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.urlFilePath
}

func (s *StorageConfig) LogFilePath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logFilePath = path
}

func (s *StorageConfig) GetLogFilePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.logFilePath
}

func (s *StorageConfig) SyncInterval(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncInterval = d
}

func (s *StorageConfig) GetSyncInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.syncInterval
}

func (s *StorageConfig) FlushOnWrite(b bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flushOnWrite = b
}

func (s *StorageConfig) GetFlushOnWrite() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.flushOnWrite
}
