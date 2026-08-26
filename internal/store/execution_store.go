// Package store provides data storage abstractions for the code sandbox.
package store

import (
	"fmt"
	"time"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/pkg/logger"
)

// PanicGuardFn is a function that returns true if the operation should panic.
type PanicGuardFn func(id string) bool

// ExecutionStore defines the interface for execution data storage.
type ExecutionStore interface {
	CreateExecution(exec *model.Execution) error
	GetExecution(id string) (*model.Execution, error)
	UpdateExecution(exec *model.Execution) error
	DeleteExecution(id string) error
	ListExecutions(filter model.ExecutionFilter) ([]*model.Execution, int64, error)
	CountExecutions() (int64, error)
	GetExecutionsByStatus(status model.ExecutionStatus) ([]*model.Execution, error)
	SetPanicGuard(fn PanicGuardFn)
	SaveWithGuard(exec *model.Execution) error
	GetWithGuard(id string) (*model.Execution, error)
	RawSnapshot() map[string]*model.Execution
}

// HistoryStore defines the interface for history record storage.
type HistoryStore interface {
	CreateHistory(record *model.HistoryRecord) error
	GetHistory(id string) (*model.HistoryRecord, error)
	DeleteHistory(id string) error
	ListHistory(query model.HistoryQuery) ([]model.HistoryRecord, int64, error)
	GetHistoryStats() (*model.HistoryStats, error)
	ClearHistory() error
	CleanupHistory(maxAge time.Duration) (int, error)
}

// TemplateStore defines the interface for template storage.
type TemplateStore interface {
	CreateTemplate(tmpl *model.Template) error
	GetTemplate(id string) (*model.Template, error)
	UpdateTemplate(tmpl *model.Template) error
	DeleteTemplate(id string) error
	ListTemplates(query model.TemplateQuery) ([]model.Template, int64, error)
	IncrementTemplateUsage(id string) error
	GetTemplatesByLanguage(language string) ([]model.Template, error)
	SearchTemplates(query string) ([]model.Template, error)
}

// Store is the combined store interface.
type Store interface {
	ExecutionStore
	HistoryStore
	TemplateStore
}

// StoreWithGuard extends Store with panic guard capabilities.
type StoreWithGuard interface {
	Store
	SetPanicGuard(fn PanicGuardFn)
	SaveWithGuard(exec *model.Execution) error
	GetWithGuard(id string) (*model.Execution, error)
	RawSnapshot() map[string]*model.Execution
}

// NewStore creates a new store based on the configuration.
func NewStore(storageType, dataDir string) (Store, error) {
	switch storageType {
	case "memory":
		return NewMemoryStore(), nil
	case "file":
		if dataDir == "" {
			return nil, fmt.Errorf("data directory is required for file storage")
		}
		return NewFileStore(dataDir)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}
}

// Ensure interfaces are met
var _ Store = (*MemoryStore)(nil)
var _ Store = (*FileStore)(nil)

// containsStr checks if a string contains a substring (case-insensitive).
func containsStr(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	sLower := toLower(s)
	subLower := toLower(substr)
	for i := 0; i <= len(sLower)-len(subLower); i++ {
		if sLower[i:i+len(subLower)] == subLower {
			return true
		}
	}
	return false
}

// toLower converts a string to lowercase (ASCII only for simplicity).
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// Helper to get logger
func getLogger() *logger.Logger {
	return logger.GetGlobal()
}

// Count returns total items in the store
func Count(s Store) int64 {
	count, _ := s.CountExecutions()
	return count
}

// Clear removes all data from the store
func Clear(s Store) {
	s.ClearHistory()
}

// ListAllTemplates returns all templates with pagination
func ListAllTemplates(s Store, query model.TemplateQuery) ([]model.Template, int64, error) {
	return s.ListTemplates(query)
}
