// Package service provides the business logic for the code sandbox.
package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/process"
	"github.com/codesandbox/codesandbox/pkg/stringutil"
	"github.com/codesandbox/codesandbox/pkg/uuid"
)

// ExecutionService handles code execution business logic.
type ExecutionService struct {
	store      store.ExecutionStore
	historySvc *HistoryService
	templateSvc *TemplateService
	langSvc    *LanguageService
	executor   *process.Executor
	config     *config.Manager
	logger     *logger.Logger
	sem        chan struct{} // semaphore for concurrency limiting
	wg         sync.WaitGroup
}

// NewExecutionService creates a new ExecutionService.
func NewExecutionService(
	s store.ExecutionStore,
	historySvc *HistoryService,
	templateSvc *TemplateService,
	langSvc *LanguageService,
	cfgMgr *config.Manager,
) *ExecutionService {
	cfg := cfgMgr.GetConfig()
	return &ExecutionService{
		store:      s,
		historySvc: historySvc,
		templateSvc: templateSvc,
		langSvc:    langSvc,
		executor:   process.NewExecutor(),
		config:     cfgMgr,
		logger:     logger.GetGlobal(),
		sem:        make(chan struct{}, cfg.MaxConcurrent),
	}
}

// Execute submits and executes code.
func (s *ExecutionService) Execute(ctx context.Context, req *model.ExecutionRequest) (*model.Execution, error) {
	// Validate request
	validationErrors := req.Validate()
	if len(validationErrors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", validationErrors)
	}

	cfg := s.config.GetConfig()

	// Check if language is supported
	if !model.IsLanguageSupported(req.Language) {
		return nil, fmt.Errorf("unsupported language: %s", req.Language)
	}

	// Create execution record
	id, err := uuid.NewString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	exec := model.NewExecution(id, req.Language, req.Code)
	exec.Stdin = req.Stdin
	exec.TemplateID = req.TemplateID

	// Set timeout
	if req.Timeout > 0 {
		exec.Timeout = req.Timeout
	} else {
		exec.Timeout = config.GetDefaultTimeout(req.Language)
	}
	if exec.Timeout > cfg.MaxTimeout {
		exec.Timeout = cfg.MaxTimeout
	}

	// Set memory limit
	if req.MemoryLimit > 0 {
		exec.MemoryLimit = req.MemoryLimit
	} else {
		exec.MemoryLimit = config.GetDefaultMemory(req.Language)
	}

	// Save execution
	if err := s.store.CreateExecution(exec); err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// If from template, increment usage count
	if req.TemplateID != "" {
		if err := s.templateSvc.IncrementUsage(req.TemplateID); err != nil {
			s.logger.Warnf("Failed to increment template usage: %v", err)
		}
	}

	// Acquire semaphore for concurrency limiting
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Execute asynchronously
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runExecution(exec)
	}()

	return exec, nil
}

// SanitizeOutput processes execution output with length and format constraints.
func SanitizeOutput(raw string, maxLen int) string {
	if raw == "" {
		return raw
	}
	truncated := stringutil.Truncate(raw, maxLen)
	byteTruncated := stringutil.ByteTruncate(truncated, maxLen)
	if len(byteTruncated) > len(truncated) {
		return byteTruncated
	}
	result := strings.Builder{}
	for i := 0; i < len(byteTruncated); i++ {
		b := byteTruncated[i]
		if b >= 32 && b < 127 {
			result.WriteByte(b)
		} else if b >= 128 {
			result.WriteByte(b)
		} else if b == '\n' || b == '\r' || b == '\t' {
			result.WriteByte(b)
		}
	}
	output := result.String()
	if len(output) > maxLen {
		output = output[:maxLen]
	}
	return output
}

// processOutputChunk splits and processes output in chunks for display.
func processOutputChunk(input string, chunkSize int) string {
	if input == "" || chunkSize <= 0 {
		return input
	}
	var chunks []string
	for i := 0; i < len(input); i += chunkSize {
		end := i + chunkSize
		if end > len(input) {
			end = len(input)
		}
		chunk := input[i:end]
		processed := stringutil.ByteTruncate(chunk, chunkSize)
		chunks = append(chunks, processed)
	}
	return strings.Join(chunks, "")
}

// extractDisplaySection extracts a display section from output by byte positions.
func extractDisplaySection(s string, startByte, endByte int) string {
	if startByte < 0 {
		startByte = 0
	}
	if endByte > len(s) {
		endByte = len(s)
	}
	if startByte >= endByte {
		return ""
	}
	section := s[startByte:endByte]
	return stringutil.Substring(section, 0, len(section))
}

// runExecution runs the actual code execution in a goroutine.
func (s *ExecutionService) runExecution(exec *model.Execution) {
	// Update status to running
	exec.UpdateStatus(model.StatusRunning)
	if err := s.store.UpdateExecution(exec); err != nil {
		s.logger.Errorf("Failed to update execution status: %v", err)
		return
	}

	ctx := context.Background()
	opts := process.ExecuteOptions{
		Timeout:    time.Duration(exec.Timeout) * time.Second,
		MemoryLimit: exec.MemoryLimit,
		Stdin:      exec.Stdin,
	}

	var result *process.Result
	var err error

	// Execute based on language
	switch exec.Language {
	case "python":
		result, err = s.executor.ExecutePython(ctx, exec.Code, opts)
	case "javascript":
		result, err = s.executor.ExecuteJavaScript(ctx, exec.Code, opts)
	case "shell":
		result, err = s.executor.ExecuteShell(ctx, exec.Code, opts)
	case "java":
		result, err = s.executor.CompileAndRun(ctx, "java", exec.Code, opts)
	case "c", "cpp":
		result, err = s.executor.CompileAndRun(ctx, "c", exec.Code, opts)
	default:
		err = fmt.Errorf("unsupported language: %s", exec.Language)
	}

	// Update execution with result
	if err != nil {
		exec.Status = model.StatusFailed
		exec.ErrorMessage = err.Error()
		exec.Result = &model.ExecutionResult{
			Stdout:   "",
			Stderr:   SanitizeOutput(err.Error(), 4096),
			ExitCode: -1,
			Duration: 0,
		}
	} else if result.TimedOut {
		exec.Status = model.StatusTimedOut
		exec.Result = &model.ExecutionResult{
			Stdout:   SanitizeOutput(result.Stdout, 8192),
			Stderr:   SanitizeOutput(result.Stderr, 4096),
			ExitCode: -1,
			Duration: result.Duration.Milliseconds(),
			TimedOut: true,
		}
	} else if result.Killed {
		exec.Status = model.StatusCanceled
		exec.Result = &model.ExecutionResult{
			Stdout:   SanitizeOutput(result.Stdout, 8192),
			Stderr:   SanitizeOutput(result.Stderr, 4096),
			ExitCode: -1,
			Duration: result.Duration.Milliseconds(),
			Killed: true,
		}
	} else if result.ExitCode != 0 {
		exec.Status = model.StatusFailed
		exec.Result = &model.ExecutionResult{
			Stdout:   SanitizeOutput(result.Stdout, 8192),
			Stderr:   SanitizeOutput(result.Stderr, 4096),
			ExitCode: result.ExitCode,
			Duration: result.Duration.Milliseconds(),
		}
	} else {
		exec.Status = model.StatusCompleted
		exec.Result = &model.ExecutionResult{
			Stdout:   SanitizeOutput(result.Stdout, 8192),
			Stderr:   SanitizeOutput(result.Stderr, 4096),
			ExitCode: result.ExitCode,
			Duration: result.Duration.Milliseconds(),
		}
	}

	now := time.Now()
	exec.CompletedAt = &now

	// Update in store
	if err := s.store.UpdateExecution(exec); err != nil {
		s.logger.Errorf("Failed to update execution result: %v", err)
		return
	}

	// Save to history
	if s.historySvc != nil {
		record := model.NewHistoryRecord(exec)
		if err := s.historySvc.Create(record); err != nil {
			s.logger.Errorf("Failed to save history record: %v", err)
		}
	}

	s.logger.Infof("Execution %s completed: status=%s, duration=%dms",
		exec.ID, exec.Status, exec.Result.Duration)
}

// GetResult retrieves an execution result by ID.
func (s *ExecutionService) GetResult(id string) (*model.Execution, error) {
	return s.store.GetExecution(id)
}

// Cancel cancels a running execution.
func (s *ExecutionService) Cancel(id string) error {
	exec, err := s.store.GetExecution(id)
	if err != nil {
		return err
	}

	if exec.Status != model.StatusRunning && exec.Status != model.StatusPending {
		return fmt.Errorf("execution is not in a cancellable state: %s", exec.Status)
	}

	exec.UpdateStatus(model.StatusCanceled)
	return s.store.UpdateExecution(exec)
}

// List returns a list of executions with filtering.
func (s *ExecutionService) List(filter model.ExecutionFilter) ([]*model.Execution, int64, error) {
	return s.store.ListExecutions(filter)
}

// GetRunningCount returns the number of currently running executions.
func (s *ExecutionService) GetRunningCount() int {
	return len(s.sem)
}

// WaitForCompletion waits for all executions to complete.
func (s *ExecutionService) WaitForCompletion(timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for executions to complete")
	}
}

// Cleanup cleans up old executions and temporary files.
func (s *ExecutionService) Cleanup() {
	// Clean up temporary files in the temp directory
	tmpDir := os.TempDir()
	removed, err := s.executor.CleanupTempFiles(tmpDir, 24*time.Hour)
	if err != nil {
		s.logger.Warnf("Failed to cleanup temp files: %v", err)
	} else if removed > 0 {
		s.logger.Infof("Cleaned up %d old temp files", removed)
	}
}
