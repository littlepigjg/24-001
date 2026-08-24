package config

import (
	"time"
)

type Config struct {
	Storage Storage
}

type Storage struct {
	urlFilePath  string
	logFilePath  string
	syncInterval time.Duration
	flushOnWrite bool
}

func Default() *Config {
	return &Config{
		Storage: Storage{
			urlFilePath:  "urls.json",
			logFilePath:  "access.log",
			syncInterval: 5 * time.Second,
			flushOnWrite: true,
		},
	}
}

func (s *Storage) URLFilePath(p string) *Storage {
	s.urlFilePath = p
	return s
}

func (s *Storage) LogFilePath(p string) *Storage {
	s.logFilePath = p
	return s
}

func (s *Storage) SyncInterval(d time.Duration) *Storage {
	s.syncInterval = d
	return s
}

func (s *Storage) FlushOnWrite(b bool) *Storage {
	s.flushOnWrite = b
	return s
}

func (s *Storage) URLPath() string {
	return s.urlFilePath
}

func (s *Storage) LogPath() string {
	return s.logFilePath
}

func (s *Storage) SyncInt() time.Duration {
	return s.syncInterval
}

func (s *Storage) FlushOnW() bool {
	return s.flushOnWrite
}