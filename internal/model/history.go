package model

import (
	"time"
)

// HistoryRecord represents a record in the execution history.
type HistoryRecord struct {
	ID           string    `json:"id"`
	ExecutionID  string    `json:"execution_id"`
	Language     string    `json:"language"`
	Code         string    `json:"code"`
	Status       ExecutionStatus `json:"status"`
	Stdout       string    `json:"stdout,omitempty"`
	Stderr       string    `json:"stderr,omitempty"`
	ExitCode     int       `json:"exit_code"`
	Duration     int64     `json:"duration"` // milliseconds
	SubmittedBy  string    `json:"submitted_by,omitempty"`
	ExecutedAt   time.Time `json:"executed_at"`
	IsTemplate   bool      `json:"is_template"`
	TemplateName string    `json:"template_name,omitempty"`
}

// HistoryQuery represents a query for history records.
type HistoryQuery struct {
	Language     string `json:"language,omitempty"`
	Status       string `json:"status,omitempty"`
	Search       string `json:"search,omitempty"`
	SubmittedBy  string `json:"submitted_by,omitempty"`
	FromDate     string `json:"from_date,omitempty"`
	ToDate       string `json:"to_date,omitempty"`
	Page         int    `json:"page,omitempty"`
	PageSize     int    `json:"page_size,omitempty"`
	SortBy       string `json:"sort_by,omitempty"`
	SortOrder    string `json:"sort_order,omitempty"`
}

// DefaultPage returns the default pagination values.
func (q *HistoryQuery) DefaultPage() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
}

// HistoryStats represents statistics for execution history.
type HistoryStats struct {
	TotalExecutions int64            `json:"total_executions"`
	SuccessRate     float64          `json:"success_rate"`
	AvgDuration     int64            `json:"avg_duration"` // milliseconds
	ByLanguage      map[string]int64 `json:"by_language"`
	ByStatus        map[string]int64 `json:"by_status"`
	RecentExecutions []HistoryRecord `json:"recent_executions"`
}

// HistoryListResponse is the API response for listing history.
type HistoryListResponse struct {
	Records    []HistoryRecord `json:"records"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// NewHistoryRecord creates a new HistoryRecord from an execution.
func NewHistoryRecord(exec *Execution) *HistoryRecord {
	record := &HistoryRecord{
		ID:          exec.ID,
		ExecutionID: exec.ID,
		Language:    exec.Language,
		Code:        exec.Code,
		Status:      exec.Status,
		ExitCode:    0,
		Duration:    0,
		SubmittedBy: exec.SubmittedBy,
		ExecutedAt:  time.Now(),
		IsTemplate:  false,
	}

	if exec.Result != nil {
		record.Stdout = exec.Result.Stdout
		record.Stderr = exec.Result.Stderr
		record.ExitCode = exec.Result.ExitCode
		record.Duration = exec.Result.Duration
	}

	return record
}
