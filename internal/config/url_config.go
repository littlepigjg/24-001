package config

import (
	"path/filepath"
	"time"
)

type Config struct {
	Storage  StorageConfig
	BasePath string
}

type StorageConfig struct {
	urlFilePath string
	logFilePath string
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

func (s *StorageConfig) GetURLFilePath() string {
	return s.urlFilePath
}

func (s *StorageConfig) GetLogFilePath() string {
	return s.logFilePath
}

func (s *StorageConfig) GetSyncInterval() time.Duration {
	return s.syncInterval
}

func (s *StorageConfig) GetFlushOnWrite() bool {
	return s.flushOnWrite
}

func Default() *Config {
	return &Config{
		Storage: StorageConfig{
			urlFilePath:  filepath.Join("data", "urls.json"),
			logFilePath:  filepath.Join("data", "access.log"),
			syncInterval: 10 * time.Second,
			flushOnWrite: true,
		},
		BasePath: ".",
	}
}
