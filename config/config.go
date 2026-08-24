package config

import "time"

type Config struct {
	Storage StorageConfig
}

type StorageConfig struct {
	urlFilePath  string
	logFilePath  string
	syncInterval time.Duration
	flushOnWrite bool
}

func Default() *Config {
	return &Config{
		Storage: StorageConfig{
			urlFilePath:  "./short_urls.json",
			logFilePath:  "./access_log.json",
			syncInterval: 5 * time.Second,
			flushOnWrite: true,
		},
	}
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

func (s *StorageConfig) URLPath() string {
	return s.urlFilePath
}

func (s *StorageConfig) LogPath() string {
	return s.logFilePath
}

func (s *StorageConfig) SyncIntervalDur() time.Duration {
	return s.syncInterval
}

func (s *StorageConfig) IsFlushOnWrite() bool {
	return s.flushOnWrite
}
