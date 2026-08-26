package service

import (
	"fmt"
	"time"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/pkg/logger"
)

// HistoryService handles execution history business logic.
type HistoryService struct {
	store  store.HistoryStore
	logger *logger.Logger
}

// NewHistoryService creates a new HistoryService.
func NewHistoryService(s store.HistoryStore) *HistoryService {
	return &HistoryService{
		store:  s,
		logger: logger.GetGlobal(),
	}
}

// Create saves a new history record.
func (s *HistoryService) Create(record *model.HistoryRecord) error {
	return s.store.CreateHistory(record)
}

// Get retrieves a history record by ID.
func (s *HistoryService) Get(id string) (*model.HistoryRecord, error) {
	return s.store.GetHistory(id)
}

// Delete removes a history record.
func (s *HistoryService) Delete(id string) error {
	return s.store.DeleteHistory(id)
}

// List returns a paginated list of history records.
func (s *HistoryService) List(query model.HistoryQuery) ([]model.HistoryRecord, int64, error) {
	query.DefaultPage()
	return s.store.ListHistory(query)
}

// GetStats returns execution history statistics.
func (s *HistoryService) GetStats() (*model.HistoryStats, error) {
	return s.store.GetHistoryStats()
}

// Clear removes all history records.
func (s *HistoryService) Clear() error {
	s.logger.Info("Clearing all history records")
	return s.store.ClearHistory()
}

// Cleanup removes records older than the specified duration.
func (s *HistoryService) Cleanup(maxAge time.Duration) (int, error) {
	if maxAge <= 0 {
		return 0, fmt.Errorf("max age must be positive")
	}
	removed, err := s.store.CleanupHistory(maxAge)
	if err != nil {
		return 0, fmt.Errorf("cleanup failed: %w", err)
	}
	s.logger.Infof("Cleaned up %d history records older than %v", removed, maxAge)
	return removed, nil
}

// GetRecent returns the most recent history records.
func (s *HistoryService) GetRecent(limit int) ([]model.HistoryRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	query := model.HistoryQuery{
		Page:     1,
		PageSize: limit,
	}
	records, _, err := s.store.ListHistory(query)
	return records, err
}

// GetByLanguage returns history records for a specific language.
func (s *HistoryService) GetByLanguage(language string, limit int) ([]model.HistoryRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	query := model.HistoryQuery{
		Language: language,
		Page:     1,
		PageSize: limit,
	}
	records, _, err := s.store.ListHistory(query)
	return records, err
}

// Search searches history records by keyword.
func (s *HistoryService) Search(keyword string, limit int) ([]model.HistoryRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	query := model.HistoryQuery{
		Search:   keyword,
		Page:     1,
		PageSize: limit,
	}
	records, _, err := s.store.ListHistory(query)
	return records, err
}
