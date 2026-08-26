package store

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/config"
)

type AccessLogEntry struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
	RemoteAddr string   `json:"remote_addr,omitempty"`
	UserAgent  string   `json:"user_agent,omitempty"`
	Referer    string   `json:"referer,omitempty"`
}

type AccessLogStore struct {
	mu     sync.Mutex
	cfg    *config.Config
	file   *os.File
	writer *bufio.Writer
}

func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	return &AccessLogStore{
		cfg: cfg,
	}, nil
}

func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	logPath := s.cfg.Storage.GetLogFilePath()
	if logPath == "" {
		return fmt.Errorf("log file path not configured")
	}

	dir := filepath.Dir(logPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	s.file = f
	s.writer = bufio.NewWriter(f)
	return nil
}

func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.writer != nil {
		if err := s.writer.Flush(); err != nil {
			return fmt.Errorf("failed to flush log writer: %w", err)
		}
	}

	if s.file != nil {
		if err := s.file.Close(); err != nil {
			return fmt.Errorf("failed to close log file: %w", err)
		}
	}

	s.file = nil
	s.writer = nil
	return nil
}

func (s *AccessLogStore) WriteLog(entry AccessLogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.writer == nil {
		return fmt.Errorf("access log store not opened")
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	if _, err := s.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	if err := s.writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	if s.cfg.Storage.GetFlushOnWrite() {
		return s.writer.Flush()
	}
	return nil
}

func (s *AccessLogStore) ReadLogs(n int) ([]AccessLogEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	logPath := s.cfg.Storage.GetLogFilePath()
	if logPath == "" {
		return nil, fmt.Errorf("log file path not configured")
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	var entries []AccessLogEntry
	for _, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		var entry AccessLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	if n > 0 && len(entries) > n {
		entries = entries[len(entries)-n:]
	}

	return entries, nil
}

func splitLines(data []byte) []string {
	var lines []string
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, string(data[start:i]))
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, string(data[start:]))
	}
	return lines
}
