package config

import "time"

type StorageConfig struct {
	urlFilePath  string
	logFilePath  string
	syncInterval time.Duration
	flushOnWrite bool
}

func (s *StorageConfig) URLFilePath(p string)       { s.urlFilePath = p }
func (s *StorageConfig) LogFilePath(p string)       { s.logFilePath = p }
func (s *StorageConfig) SyncInterval(d time.Duration) { s.syncInterval = d }
func (s *StorageConfig) FlushOnWrite(b bool)        { s.flushOnWrite = b }

func (s *StorageConfig) URLPath() string            { return s.urlFilePath }
func (s *StorageConfig) LogPath() string           { return s.logFilePath }
func (s *StorageConfig) SyncIntervalVal() time.Duration { return s.syncInterval }
func (s *StorageConfig) FlushOnWriteVal() bool     { return s.flushOnWrite }

type Config struct {
	Storage StorageConfig
}

func Default() *Config {
	c := &Config{}
	c.Storage.URLFilePath("./data/urls.json")
	c.Storage.LogFilePath("./data/access.log")
	c.Storage.SyncInterval(5 * time.Second)
	c.Storage.FlushOnWrite(true)
	return c
}
