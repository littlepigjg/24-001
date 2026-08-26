package model

import (
	"time"
)

// ExecutionStats contains runtime statistics for executions.
type ExecutionStats struct {
	TotalExecutions    int64         `json:"total_executions"`
	SuccessfulExecutions int64      `json:"successful_executions"`
	FailedExecutions   int64         `json:"failed_executions"`
	TimedOutExecutions int64         `json:"timed_out_executions"`
	AvgDuration        time.Duration `json:"avg_duration"`
	MaxDuration        time.Duration `json:"max_duration"`
	MinDuration        time.Duration `json:"min_duration"`
	TotalCPUSeconds    float64       `json:"total_cpu_seconds"`
	ByLanguage         map[string]int64 `json:"by_language"`
	Last24hExecutions  int64         `json:"last_24h_executions"`
	Uptime             time.Duration `json:"uptime"`
}

// SystemStatus represents the current system status.
type SystemStatus struct {
	Status       string            `json:"status"` // "healthy", "degraded", "down"
	Version      string            `json:"version"`
	Uptime       time.Duration     `json:"uptime"`
	StartTime    time.Time         `json:"start_time"`
	ActiveExecutions int          `json:"active_executions"`
	PendingExecutions int          `json:"pending_executions"`
	TotalMemory  int64             `json:"total_memory"`
	UsedMemory   int64             `json:"used_memory"`
	FreeMemory   int64             `json:"free_memory"`
	CPUCores     int               `json:"cpu_cores"`
	LoadAvg      float64           `json:"load_avg"`
	DiskUsage    float64           `json:"disk_usage"` // percentage
	Features     map[string]bool   `json:"features"`
}

// HealthResponse is the response for health check endpoints.
type HealthResponse struct {
	Status   string          `json:"status"`
	Version  string          `json:"version"`
	Uptime   time.Duration   `json:"uptime"`
	Checks   map[string]bool `json:"checks"`
	Database *struct {
		Status string          `json:"status"`
		Latency time.Duration   `json:"latency"`
	} `json:"database,omitempty"`
	Storage *struct {
		Status       string  `json:"status"`
		TotalSpace   int64   `json:"total_space"`
		UsedSpace    int64   `json:"used_space"`
		Available    int64   `json:"available"`
		UsagePercent float64 `json:"usage_percent"`
	} `json:"storage,omitempty"`
}

// ReadyResponse is the response for readiness check.
type ReadyResponse struct {
	Ready    bool            `json:"ready"`
	Checks   map[string]bool `json:"checks"`
	Message  string          `json:"message"`
}

// PaginatedResponse is a generic paginated response.
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// NewPaginatedResponse creates a new paginated response.
func NewPaginatedResponse(items interface{}, total int64, page, pageSize int) *PaginatedResponse {
	totalPages := 0
	if pageSize > 0 {
		totalPages = int(total) / pageSize
		if int(total)%pageSize > 0 {
			totalPages++
		}
	}
	return &PaginatedResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// Pagination holds pagination parameters.
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// DefaultPagination returns default pagination values.
func DefaultPagination() Pagination {
	return Pagination{
		Page:     1,
		PageSize: 20,
	}
}

// Validate checks and corrects pagination values.
func (p *Pagination) Validate() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
}

// Offset returns the SQL offset for the current page.
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}
