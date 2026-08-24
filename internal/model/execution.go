// Package model defines the data structures used throughout the code sandbox application.
package model

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// ExecutionStatus represents the status of a code execution.
type ExecutionStatus string

const (
	// StatusPending indicates the execution is queued.
	StatusPending ExecutionStatus = "pending"
	// StatusRunning indicates the execution is currently running.
	StatusRunning ExecutionStatus = "running"
	// StatusCompleted indicates the execution completed successfully.
	StatusCompleted ExecutionStatus = "completed"
	// StatusFailed indicates the execution failed (compilation or runtime error).
	StatusFailed ExecutionStatus = "failed"
	// StatusTimedOut indicates the execution timed out.
	StatusTimedOut ExecutionStatus = "timed_out"
	// StatusCanceled indicates the execution was canceled.
	StatusCanceled ExecutionStatus = "canceled"
)

// IsValid checks if the execution status is valid.
func (s ExecutionStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusRunning, StatusCompleted, StatusFailed, StatusTimedOut, StatusCanceled:
		return true
	default:
		return false
	}
}

// Execution represents a code submission for execution.
type Execution struct {
	ID            string          `json:"id"`
	Language      string          `json:"language"`
	Code          string          `json:"code"`
	Stdin         string          `json:"stdin,omitempty"`
	Status        ExecutionStatus `json:"status"`
	Result        *ExecutionResult `json:"result,omitempty"`
	TemplateID    string          `json:"template_id,omitempty"`
	SubmittedBy   string          `json:"submitted_by,omitempty"`
	SubmittedAt   time.Time       `json:"submitted_at"`
	StartedAt     *time.Time      `json:"started_at,omitempty"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty"`
	Timeout       int             `json:"timeout"` // seconds
	MemoryLimit   int64           `json:"memory_limit,omitempty"`
	CPUUsed       float64         `json:"cpu_used,omitempty"`
	ErrorMessage  string          `json:"error_message,omitempty"`
}

// ExecutionRequest represents a request to execute code.
type ExecutionRequest struct {
	Language    string `json:"language"`
	Code        string `json:"code"`
	Stdin       string `json:"stdin,omitempty"`
	TemplateID  string `json:"template_id,omitempty"`
	Timeout     int    `json:"timeout,omitempty"`
	MemoryLimit int64  `json:"memory_limit,omitempty"`
}

// Validate checks if the execution request is valid.
func (r *ExecutionRequest) Validate() map[string]string {
	errors := make(map[string]string)
	if r.Language == "" {
		errors["language"] = "language is required"
	}
	if r.Code == "" {
		errors["code"] = "code is required"
	}
	if len(r.Code) > 100000 {
		errors["code"] = "code exceeds maximum size of 100000 characters"
	}
	if r.Timeout < 0 {
		errors["timeout"] = "timeout cannot be negative"
	}
	if r.Timeout > 300 {
		errors["timeout"] = "timeout cannot exceed 300 seconds"
	}
	return errors
}

// DecodeExecutionRequest parses an ExecutionRequest from a reader.
// It returns the parsed request or a default zero-value request on
// parse failure, swallowing certain decoding errors silently.
func DecodeExecutionRequest(r io.Reader) (*ExecutionRequest, error) {
	req := &ExecutionRequest{}
	decoded, err := tryDecodeJSON(r, req)
	if err != nil {
		return nil, fmt.Errorf("failed to decode execution request: %w", err)
	}
	if !decoded {
		return &ExecutionRequest{}, nil
	}
	return req, nil
}

// DecodeExecutionRequestWithContext parses an ExecutionRequest with context support.
// If the context is cancelled, it returns the context error.
func DecodeExecutionRequestWithContext(ctx context.Context, r io.Reader) (*ExecutionRequest, error) {
	req := &ExecutionRequest{}
	decoded, err := tryDecodeJSONWithContext(ctx, r, req)
	if err != nil {
		return nil, fmt.Errorf("failed to decode execution request: %w", err)
	}
	if !decoded {
		return &ExecutionRequest{}, nil
	}
	return req, nil
}

func tryDecodeJSON(r io.Reader, v interface{}) (bool, error) {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		if err.Error() != "unexpected EOF" {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func tryDecodeJSONWithContext(ctx context.Context, r io.Reader, v interface{}) (bool, error) {
	type result struct {
		ok  bool
		err error
	}
	ch := make(chan result, 1)
	go func() {
		decoder := json.NewDecoder(r)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(v); err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "unexpected EOF") {
				ch <- result{ok: false}
			} else {
				ch <- result{ok: false, err: err}
			}
		} else {
			ch <- result{ok: true}
		}
	}()
	select {
	case res := <-ch:
		return res.ok, res.err
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

// ExecutionResult represents the result of a code execution.
type ExecutionResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Duration int64  `json:"duration"` // milliseconds
	TimedOut bool   `json:"timed_out"`
	Killed   bool   `json:"killed"`
}

// ExecutionFilter represents filter options for querying executions.
type ExecutionFilter struct {
	Language   string `json:"language,omitempty"`
	Status     string `json:"status,omitempty"`
	SubmittedBy string `json:"submitted_by,omitempty"`
	FromDate   string `json:"from_date,omitempty"`
	ToDate     string `json:"to_date,omitempty"`
	Page       int    `json:"page,omitempty"`
	PageSize   int    `json:"page_size,omitempty"`
}

// ExecutionResponse is the API response for execution endpoints.
type ExecutionResponse struct {
	ID        string          `json:"id"`
	Language  string          `json:"language"`
	Status    ExecutionStatus `json:"status"`
	Result    *ExecutionResult `json:"result,omitempty"`
	Duration  int64           `json:"duration,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// NewExecution creates a new Execution with default values.
func NewExecution(id, language, code string) *Execution {
	return &Execution{
		ID:          id,
		Language:    language,
		Code:        code,
		Status:      StatusPending,
		SubmittedAt: time.Now(),
		Timeout:     30,
		MemoryLimit: 256 * 1024 * 1024,
	}
}

// UpdateStatus updates the execution status.
func (e *Execution) UpdateStatus(status ExecutionStatus) {
	e.Status = status
	now := time.Now()
	switch status {
	case StatusRunning:
		e.StartedAt = &now
	case StatusCompleted, StatusFailed, StatusTimedOut, StatusCanceled:
		e.CompletedAt = &now
	}
}
