package config

import "time"

type StorageConfig struct {
	urlFilePath  string
	logFilePath  string
	syncInterval time.Duration
	flushOnWrite bool
	pageSize     int
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
	// A non-positive page size is unsafe: callers use it as a slice step and
	// a negative value would slice out of bounds (slice bounds out of range).
	// Fall back to the default instead of returning the raw value.
	if s.pageSize <= 0 {
		return defaultPageSize
	}
	return s.pageSize
}

type Config struct {
	Storage StorageConfig
}

// defaultPageSize is the page size used when none is configured or when the
// configured value is non-positive. It is the safe fallback that all paging
// and batching code ultimately degrades to.
const defaultPageSize = 100

func Default() *Config {
	return &Config{
		Storage: StorageConfig{
			urlFilePath:  "./data/urls.json",
			logFilePath:  "./data/access.log",
			syncInterval: 5 * time.Second,
			flushOnWrite: true,
			pageSize:     defaultPageSize,
		},
	}
}
