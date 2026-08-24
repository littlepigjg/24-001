package config

import "time"

type Storage struct {
	urlFilePath   string
	logFilePath   string
	syncInterval  time.Duration
	flushOnWrite  bool
	httpsEnabled  bool
	httpsCertFile string
	httpsKeyFile  string
}

func (s *Storage) URLFilePath(path string) *Storage {
	s.urlFilePath = path
	return s
}

func (s *Storage) LogFilePath(path string) *Storage {
	s.logFilePath = path
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

func (s *Storage) HTTPSEnabled() bool {
	return s.httpsEnabled
}

func (s *Storage) HTTPSCertFile() string {
	return s.httpsCertFile
}

func (s *Storage) HTTPSKeyFile() string {
	return s.httpsKeyFile
}

func (s *Storage) GetURLFilePath() string {
	return s.urlFilePath
}

func (s *Storage) GetLogFilePath() string {
	return s.logFilePath
}

func (s *Storage) GetSyncInterval() time.Duration {
	return s.syncInterval
}

func (s *Storage) GetFlushOnWrite() bool {
	return s.flushOnWrite
}

type Config struct {
	Storage Storage
}

func Default() *Config {
	return &Config{
		Storage: Storage{
			urlFilePath:   "./data/urls.json",
			logFilePath:   "./data/access.log",
			syncInterval:  5 * time.Second,
			flushOnWrite:  true,
			httpsEnabled:  false,
			httpsCertFile: "",
			httpsKeyFile:  "",
		},
	}
}
