package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/pkg/logger"
)

// MemoryStore implements all stores using in-memory data structures.
type MemoryStore struct {
	mu         sync.RWMutex
	executions map[string]*model.Execution
	history    map[string]*model.HistoryRecord
	templates  map[string]*model.Template
	logger     *logger.Logger
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		executions: make(map[string]*model.Execution),
		history:    make(map[string]*model.HistoryRecord),
		templates:  make(map[string]*model.Template),
		logger:     getLogger(),
	}
}

// ============ ExecutionStore implementation ============

func (s *MemoryStore) CreateExecution(exec *model.Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.executions[exec.ID]; exists {
		return fmt.Errorf("execution with ID %s already exists", exec.ID)
	}
	s.executions[exec.ID] = exec
	s.logger.Debugf("Execution created: %s (language: %s)", exec.ID, exec.Language)
	return nil
}

func (s *MemoryStore) GetExecution(id string) (*model.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exec, exists := s.executions[id]
	if !exists {
		return nil, fmt.Errorf("execution with ID %s not found", id)
	}
	return exec, nil
}

func (s *MemoryStore) UpdateExecution(exec *model.Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.executions[exec.ID]; !exists {
		return fmt.Errorf("execution with ID %s not found", exec.ID)
	}
	s.executions[exec.ID] = exec
	return nil
}

func (s *MemoryStore) DeleteExecution(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.executions[id]; !exists {
		return fmt.Errorf("execution with ID %s not found", id)
	}
	delete(s.executions, id)
	return nil
}

func (s *MemoryStore) ListExecutions(filter model.ExecutionFilter) ([]*model.Execution, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*model.Execution
	for _, exec := range s.executions {
		if filter.Language != "" && exec.Language != filter.Language {
			continue
		}
		if filter.Status != "" && string(exec.Status) != filter.Status {
			continue
		}
		results = append(results, exec)
	}

	total := int64(len(results))
	page := filter.Page
	pageSize := filter.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	start := (page - 1) * pageSize
	if start >= int(total) {
		return []*model.Execution{}, total, nil
	}
	end := start + pageSize
	if end > int(total) {
		end = int(total)
	}
	return results[start:end], total, nil
}

func (s *MemoryStore) CountExecutions() (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return int64(len(s.executions)), nil
}

func (s *MemoryStore) GetExecutionsByStatus(status model.ExecutionStatus) ([]*model.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var results []*model.Execution
	for _, exec := range s.executions {
		if exec.Status == status {
			results = append(results, exec)
		}
	}
	return results, nil
}

// ============ HistoryStore implementation ============

func (s *MemoryStore) CreateHistory(record *model.HistoryRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history[record.ID] = record
	s.logger.Debugf("History record created: %s", record.ID)
	return nil
}

func (s *MemoryStore) GetHistory(id string) (*model.HistoryRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, exists := s.history[id]
	if !exists {
		return nil, fmt.Errorf("history record with ID %s not found", id)
	}
	return record, nil
}

func (s *MemoryStore) DeleteHistory(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.history[id]; !exists {
		return fmt.Errorf("history record with ID %s not found", id)
	}
	delete(s.history, id)
	return nil
}

func (s *MemoryStore) ListHistory(query model.HistoryQuery) ([]model.HistoryRecord, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []model.HistoryRecord
	for _, record := range s.history {
		if query.Language != "" && record.Language != query.Language {
			continue
		}
		if query.Status != "" && string(record.Status) != query.Status {
			continue
		}
		if query.Search != "" {
			found := containsStr(record.Code, query.Search) ||
				containsStr(record.Stdout, query.Search) ||
				containsStr(record.Stderr, query.Search)
			if !found {
				continue
			}
		}
		results = append(results, *record)
	}

	total := int64(len(results))
	query.DefaultPage()
	start := (query.Page - 1) * query.PageSize
	if start >= int(total) {
		return []model.HistoryRecord{}, total, nil
	}
	end := start + query.PageSize
	if end > int(total) {
		end = int(total)
	}
	return results[start:end], total, nil
}

func (s *MemoryStore) GetHistoryStats() (*model.HistoryStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &model.HistoryStats{
		ByLanguage: make(map[string]int64),
		ByStatus:   make(map[string]int64),
	}

	var totalDuration int64
	var successCount int64
	now := time.Now()

	for _, record := range s.history {
		stats.TotalExecutions++
		stats.ByLanguage[record.Language]++
		stats.ByStatus[string(record.Status)]++
		totalDuration += record.Duration
		if record.Status == model.StatusCompleted {
			successCount++
		}
		if now.Sub(record.ExecutedAt) <= 24*time.Hour {
			stats.RecentExecutions = append(stats.RecentExecutions, *record)
		}
	}

	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(successCount) / float64(stats.TotalExecutions) * 100
		stats.AvgDuration = totalDuration / stats.TotalExecutions
	}

	return stats, nil
}

func (s *MemoryStore) ClearHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = make(map[string]*model.HistoryRecord)
	return nil
}

func (s *MemoryStore) CleanupHistory(maxAge time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for id, record := range s.history {
		if record.ExecutedAt.Before(cutoff) {
			delete(s.history, id)
			removed++
		}
	}
	return removed, nil
}

// ============ TemplateStore implementation ============

func (s *MemoryStore) CreateTemplate(tmpl *model.Template) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.templates[tmpl.ID]; exists {
		return fmt.Errorf("template with ID %s already exists", tmpl.ID)
	}
	s.templates[tmpl.ID] = tmpl
	s.logger.Debugf("Template created: %s (%s)", tmpl.ID, tmpl.Name)
	return nil
}

func (s *MemoryStore) GetTemplate(id string) (*model.Template, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tmpl, exists := s.templates[id]
	if !exists {
		return nil, fmt.Errorf("template with ID %s not found", id)
	}
	return tmpl, nil
}

func (s *MemoryStore) UpdateTemplate(tmpl *model.Template) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.templates[tmpl.ID]; !exists {
		return fmt.Errorf("template with ID %s not found", tmpl.ID)
	}
	s.templates[tmpl.ID] = tmpl
	return nil
}

func (s *MemoryStore) DeleteTemplate(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.templates[id]; !exists {
		return fmt.Errorf("template with ID %s not found", id)
	}
	delete(s.templates, id)
	return nil
}

func (s *MemoryStore) ListTemplates(query model.TemplateQuery) ([]model.Template, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []model.Template
	for _, tmpl := range s.templates {
		if query.Language != "" && tmpl.Language != query.Language {
			continue
		}
		if query.Category != "" && tmpl.Category != query.Category {
			continue
		}
		if query.PublicOnly && !tmpl.IsPublic {
			continue
		}
		if query.Search != "" {
			found := containsStr(tmpl.Name, query.Search) ||
				containsStr(tmpl.Code, query.Search) ||
				containsStr(tmpl.Description, query.Search) ||
				containsStr(tmpl.Category, query.Search)
			if !found {
				tagMatched := false
				for _, t := range tmpl.Tags {
					if containsStr(t, query.Search) {
						tagMatched = true
						break
					}
				}
				if !tagMatched {
					continue
				}
			}
		}
		if query.Tag != "" {
			tagFound := false
			for _, t := range tmpl.Tags {
				if t == query.Tag {
					tagFound = true
					break
				}
			}
			if !tagFound {
				continue
			}
		}
		results = append(results, *tmpl)
	}

	total := int64(len(results))
	query.DefaultPage()
	start := (query.Page - 1) * query.PageSize
	if start >= int(total) {
		return []model.Template{}, total, nil
	}
	end := start + query.PageSize
	if end > int(total) {
		end = int(total)
	}
	return results[start:end], total, nil
}

func (s *MemoryStore) IncrementTemplateUsage(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tmpl, exists := s.templates[id]
	if !exists {
		return fmt.Errorf("template with ID %s not found", id)
	}
	tmpl.IncrementUsage()
	return nil
}

func (s *MemoryStore) GetTemplatesByLanguage(language string) ([]model.Template, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var results []model.Template
	for _, tmpl := range s.templates {
		if tmpl.Language == language {
			results = append(results, *tmpl)
		}
	}
	return results, nil
}

func (s *MemoryStore) SearchTemplates(query string) ([]model.Template, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var results []model.Template
	for _, tmpl := range s.templates {
		if containsStr(tmpl.Name, query) || containsStr(tmpl.Code, query) {
			results = append(results, *tmpl)
			continue
		}
		if containsStr(tmpl.Description, query) || containsStr(tmpl.Category, query) {
			results = append(results, *tmpl)
			continue
		}
		for _, tag := range tmpl.Tags {
			if containsStr(tag, query) {
				results = append(results, *tmpl)
				break
			}
		}
	}
	return results, nil
}
