package config

import "time"

type StorageConfig struct {
	urlFilePath   string
	logFilePath   string
	syncInterval  time.Duration
	flushOnWrite  bool
	pageSize      int
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

func (s *StorageConfig) SetPageSize(n int) {
	s.pageSize = n
}

func (s *StorageConfig) PageSize() int {
	return s.pageSize
}

type Config struct {
	Storage StorageConfig
}

func Default() *Config {
	return &Config{
		Storage: StorageConfig{
			urlFilePath:  "./data/urls.json",
			logFilePath:  "./data/access.log",
			syncInterval: 5 * time.Second,
			flushOnWrite: true,
			pageSize:     100,
		},
	}
}
