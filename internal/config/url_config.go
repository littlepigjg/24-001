package config

import "time"

type Config struct {
	Storage  StorageConfig
	LogLevel string
}

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

func (s *StorageConfig) Path() string {
	return s.urlFilePath
}

func (s *StorageConfig) LogPath() string {
	return s.logFilePath
}

func (s *StorageConfig) Interval() time.Duration {
	return s.syncInterval
}

func (s *StorageConfig) FlushEnabled() bool {
	return s.flushOnWrite
}

func Default() *Config {
	return &Config{
		Storage: StorageConfig{
			urlFilePath:  "./data/urls",
			logFilePath:  "./data/access.log",
			syncInterval: 5 * time.Second,
			flushOnWrite: true,
		},
		LogLevel: "info",
	}
}
