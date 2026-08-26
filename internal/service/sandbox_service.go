package service

import (
	"fmt"
	"os"
	"time"

	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/process"
)

// SandboxService manages sandbox execution environments.
type SandboxService struct {
	executor  *process.Executor
	logger    *logger.Logger
	sandboxDir string
}

// NewSandboxService creates a new SandboxService.
func NewSandboxService(executor *process.Executor, sandboxDir string) *SandboxService {
	return &SandboxService{
		executor:  executor,
		logger:    logger.GetGlobal(),
		sandboxDir: sandboxDir,
	}
}

// Initialize sets up the sandbox directory.
func (s *SandboxService) Initialize() error {
	dir := s.sandboxDir
	if dir == "" {
		dir = os.TempDir()
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create sandbox directory: %w", err)
	}
	s.logger.Infof("Sandbox directory initialized: %s", dir)
	return nil
}

// Cleanup removes old temporary files.
func (s *SandboxService) Cleanup(maxAge time.Duration) (int, error) {
	dir := s.sandboxDir
	if dir == "" {
		dir = os.TempDir()
	}
	return s.executor.CleanupTempFiles(dir, maxAge)
}

// StartPeriodicCleanup starts a goroutine that periodically cleans up old files.
func (s *SandboxService) StartPeriodicCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			removed, err := s.Cleanup(24 * time.Hour)
			if err != nil {
				s.logger.Warnf("Periodic cleanup failed: %v", err)
			} else if removed > 0 {
				s.logger.Debugf("Periodic cleanup removed %d old files", removed)
			}
		}
	}()
}

// GetSandboxDir returns the sandbox directory path.
func (s *SandboxService) GetSandboxDir() string {
	return s.sandboxDir
}

// SetSandboxDir sets the sandbox directory.
func (s *SandboxService) SetSandboxDir(dir string) {
	s.sandboxDir = dir
}

// CheckHealth checks if the sandbox environment is healthy.
func (s *SandboxService) CheckHealth() map[string]bool {
	checks := make(map[string]bool)

	// Check if temp directory is writable
	dir := s.sandboxDir
	if dir == "" {
		dir = os.TempDir()
	}

	testFile, err := os.CreateTemp(dir, "sandbox-health-*.test")
	if err != nil {
		checks["writable"] = false
		checks["sandbox_dir"] = false
	} else {
		checks["writable"] = true
		checks["sandbox_dir"] = true
		testFile.Close()
		os.Remove(testFile.Name())
	}

	// Check available languages
	for _, lang := range []string{"python", "javascript", "shell"} {
		checks["lang_"+lang] = s.executor.IsLanguageAvailable(lang)
	}

	return checks
}
