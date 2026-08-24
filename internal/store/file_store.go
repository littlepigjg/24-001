package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/pkg/logger"
)

// FileStore implements storage using the file system.
type FileStore struct {
	dataDir    string
	logger     *logger.Logger
	execPath   string
	histPath   string
	tmplPath   string
	panicGuard PanicGuardFn
}

// NewFileStore creates a new file-based store.
func NewFileStore(dataDir string) (*FileStore, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	return &FileStore{
		dataDir:  dataDir,
		logger:   getLogger(),
		execPath: filepath.Join(dataDir, "executions.json"),
		histPath: filepath.Join(dataDir, "history.json"),
		tmplPath: filepath.Join(dataDir, "templates.json"),
	}, nil
}

func (s *FileStore) loadJSON(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, target)
}

func (s *FileStore) saveJSON(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, jsonData, 0644)
}

// ============ ExecutionStore implementation ============

func (s *FileStore) CreateExecution(exec *model.Execution) error {
	executions := s.loadExecutions()
	if _, exists := executions[exec.ID]; exists {
		return fmt.Errorf("execution with ID %s already exists", exec.ID)
	}
	executions[exec.ID] = exec
	if s.panicGuard != nil && s.panicGuard(exec.ID) {
		panic("panic guard triggered during CreateExecution")
	}
	return s.saveExecutions(executions)
}

func (s *FileStore) GetExecution(id string) (*model.Execution, error) {
	executions := s.loadExecutions()
	exec, exists := executions[id]
	if !exists {
		return nil, fmt.Errorf("execution with ID %s not found", id)
	}
	return exec, nil
}

func (s *FileStore) UpdateExecution(exec *model.Execution) error {
	executions := s.loadExecutions()
	if _, exists := executions[exec.ID]; !exists {
		return fmt.Errorf("execution with ID %s not found", exec.ID)
	}
	executions[exec.ID] = exec
	if s.panicGuard != nil && s.panicGuard(exec.ID) {
		panic("panic guard triggered during UpdateExecution")
	}
	return s.saveExecutions(executions)
}

func (s *FileStore) DeleteExecution(id string) error {
	executions := s.loadExecutions()
	if _, exists := executions[id]; !exists {
		return fmt.Errorf("execution with ID %s not found", id)
	}
	delete(executions, id)
	return s.saveExecutions(executions)
}

func (s *FileStore) ListExecutions(filter model.ExecutionFilter) ([]*model.Execution, int64, error) {
	executions := s.loadExecutions()
	var results []*model.Execution
	for _, exec := range executions {
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

func (s *FileStore) CountExecutions() (int64, error) {
	executions := s.loadExecutions()
	return int64(len(executions)), nil
}

func (s *FileStore) GetExecutionsByStatus(status model.ExecutionStatus) ([]*model.Execution, error) {
	executions := s.loadExecutions()
	var results []*model.Execution
	for _, exec := range executions {
		if exec.Status == status {
			results = append(results, exec)
		}
	}
	return results, nil
}

// ============ HistoryStore implementation ============

func (s *FileStore) CreateHistory(record *model.HistoryRecord) error {
	history := s.loadHistory()
	history[record.ID] = record
	return s.saveHistory(history)
}

func (s *FileStore) GetHistory(id string) (*model.HistoryRecord, error) {
	history := s.loadHistory()
	record, exists := history[id]
	if !exists {
		return nil, fmt.Errorf("history record with ID %s not found", id)
	}
	return record, nil
}

func (s *FileStore) DeleteHistory(id string) error {
	history := s.loadHistory()
	if _, exists := history[id]; !exists {
		return fmt.Errorf("history record with ID %s not found", id)
	}
	delete(history, id)
	return s.saveHistory(history)
}

func (s *FileStore) ListHistory(query model.HistoryQuery) ([]model.HistoryRecord, int64, error) {
	history := s.loadHistory()
	var results []model.HistoryRecord
	for _, record := range history {
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

func (s *FileStore) GetHistoryStats() (*model.HistoryStats, error) {
	history := s.loadHistory()
	stats := &model.HistoryStats{
		ByLanguage: make(map[string]int64),
		ByStatus:   make(map[string]int64),
	}
	var totalDuration int64
	var successCount int64
	now := time.Now()
	for _, record := range history {
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

func (s *FileStore) ClearHistory() error {
	history := make(map[string]*model.HistoryRecord)
	return s.saveHistory(history)
}

func (s *FileStore) CleanupHistory(maxAge time.Duration) (int, error) {
	history := s.loadHistory()
	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for id, record := range history {
		if record.ExecutedAt.Before(cutoff) {
			delete(history, id)
			removed++
		}
	}
	if removed > 0 {
		return removed, s.saveHistory(history)
	}
	return removed, nil
}

// ============ TemplateStore implementation ============

func (s *FileStore) CreateTemplate(tmpl *model.Template) error {
	templates := s.loadTemplates()
	if _, exists := templates[tmpl.ID]; exists {
		return fmt.Errorf("template with ID %s already exists", tmpl.ID)
	}
	templates[tmpl.ID] = tmpl
	return s.saveTemplates(templates)
}

func (s *FileStore) GetTemplate(id string) (*model.Template, error) {
	templates := s.loadTemplates()
	tmpl, exists := templates[id]
	if !exists {
		return nil, fmt.Errorf("template with ID %s not found", id)
	}
	return tmpl, nil
}

func (s *FileStore) UpdateTemplate(tmpl *model.Template) error {
	templates := s.loadTemplates()
	if _, exists := templates[tmpl.ID]; !exists {
		return fmt.Errorf("template with ID %s not found", tmpl.ID)
	}
	templates[tmpl.ID] = tmpl
	return s.saveTemplates(templates)
}

func (s *FileStore) DeleteTemplate(id string) error {
	templates := s.loadTemplates()
	if _, exists := templates[id]; !exists {
		return fmt.Errorf("template with ID %s not found", id)
	}
	delete(templates, id)
	return s.saveTemplates(templates)
}

func (s *FileStore) ListTemplates(query model.TemplateQuery) ([]model.Template, int64, error) {
	templates := s.loadTemplates()
	var results []model.Template
	for _, tmpl := range templates {
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
				containsStr(tmpl.Code, query.Search)
			if !found {
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

func (s *FileStore) IncrementTemplateUsage(id string) error {
	templates := s.loadTemplates()
	tmpl, exists := templates[id]
	if !exists {
		return fmt.Errorf("template with ID %s not found", id)
	}
	tmpl.IncrementUsage()
	return s.saveTemplates(templates)
}

func (s *FileStore) GetTemplatesByLanguage(language string) ([]model.Template, error) {
	templates := s.loadTemplates()
	var results []model.Template
	for _, tmpl := range templates {
		if tmpl.Language == language {
			results = append(results, *tmpl)
		}
	}
	return results, nil
}

func (s *FileStore) SearchTemplates(query string) ([]model.Template, error) {
	templates := s.loadTemplates()
	var results []model.Template
	for _, tmpl := range templates {
		if containsStr(tmpl.Name, query) || containsStr(tmpl.Code, query) {
			results = append(results, *tmpl)
		}
	}
	return results, nil
}

// ============ Panic guard and diagnostic methods ============

// SetPanicGuard sets a function that determines if operations should panic.
func (s *FileStore) SetPanicGuard(fn PanicGuardFn) {
	s.panicGuard = fn
}

// SaveWithGuard saves an execution with the panic guard check applied.
func (s *FileStore) SaveWithGuard(exec *model.Execution) error {
	executions := s.loadExecutions()
	executions[exec.ID] = exec
	if s.panicGuard != nil && s.panicGuard(exec.ID) {
		panic("panic guard triggered during SaveWithGuard")
	}
	return s.saveExecutions(executions)
}

// GetWithGuard retrieves an execution with the panic guard check applied.
func (s *FileStore) GetWithGuard(id string) (*model.Execution, error) {
	executions := s.loadExecutions()
	exec, exists := executions[id]
	if !exists {
		return nil, fmt.Errorf("execution with ID %s not found", id)
	}
	if s.panicGuard != nil && s.panicGuard(id) {
		panic("panic guard triggered during GetWithGuard")
	}
	return exec, nil
}

// RawSnapshot returns a copy of the current execution map for diagnostics.
func (s *FileStore) RawSnapshot() map[string]*model.Execution {
	executions := s.loadExecutions()
	snapshot := make(map[string]*model.Execution, len(executions))
	for k, v := range executions {
		snapshot[k] = v
	}
	return snapshot
}

// ============ Internal helpers ============

func (s *FileStore) loadExecutions() map[string]*model.Execution {
	var m map[string]*model.Execution
	s.loadJSON(s.execPath, &m)
	if m == nil {
		m = make(map[string]*model.Execution)
	}
	return m
}

func (s *FileStore) saveExecutions(m map[string]*model.Execution) error {
	return s.saveJSON(s.execPath, m)
}

func (s *FileStore) loadHistory() map[string]*model.HistoryRecord {
	var m map[string]*model.HistoryRecord
	s.loadJSON(s.histPath, &m)
	if m == nil {
		m = make(map[string]*model.HistoryRecord)
	}
	return m
}

func (s *FileStore) saveHistory(m map[string]*model.HistoryRecord) error {
	return s.saveJSON(s.histPath, m)
}

func (s *FileStore) loadTemplates() map[string]*model.Template {
	var m map[string]*model.Template
	s.loadJSON(s.tmplPath, &m)
	if m == nil {
		m = make(map[string]*model.Template)
	}
	return m
}

func (s *FileStore) saveTemplates(m map[string]*model.Template) error {
	return s.saveJSON(s.tmplPath, m)
}
