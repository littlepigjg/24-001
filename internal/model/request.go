// Package model provides additional HTTP request/response models.
package model

import "time"

// SubmissionRequest is used for batch submission of multiple coding tasks.
type SubmissionRequest struct {
	Requests []ExecutionRequest `json:"requests"`
}

// SubmissionResponse contains results from batch execution.
type SubmissionResponse struct {
	Results   []*ExecutionResult `json:"results"`
	TotalTime time.Duration      `json:"total_time"`
}

// Validate checks the submission request.
func (r *SubmissionRequest) Validate() []string {
	var errors []string
	if len(r.Requests) == 0 {
		errors = append(errors, "at least one request is required")
	}
	if len(r.Requests) > 10 {
		errors = append(errors, "maximum 10 requests per batch")
	}
	return errors
}

// StatsResponse contains system statistics.
type StatsResponse struct {
	TotalExecutions  int64  `json:"total_executions"`
	SuccessfulCount  int64  `json:"successful_count"`
	FailedCount      int64  `json:"failed_count"`
	AvgExecutionTime int64  `json:"avg_execution_time_ms"`
	LanguageBreakdown map[string]int64 `json:"language_breakdown"`
}

// Validate checks the request.
func (r *SubmissionResponse) Validate() []string {
	var errors []string
	if len(r.Results) == 0 {
		errors = append(errors, "no results provided")
	}
	return errors
}
