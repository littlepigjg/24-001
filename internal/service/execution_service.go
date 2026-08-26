// Package service provides the business logic for the code sandbox.
package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/process"
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
	validationErrors := req.Validate()
	if len(validationErrors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", validationErrors)
	}

	cfg := s.config.GetConfig()

	if !model.IsLanguageSupported(req.Language) {
		return nil, fmt.Errorf("unsupported language: %s", req.Language)
	}

	id, err := uuid.NewString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	exec := model.NewExecution(id, req.Language, req.Code)
	exec.Stdin = req.Stdin
	exec.TemplateID = req.TemplateID

	if req.Timeout > 0 {
		exec.Timeout = req.Timeout
	} else {
		exec.Timeout = config.GetDefaultTimeout(req.Language)
	}
	if exec.Timeout > cfg.MaxTimeout {
		exec.Timeout = cfg.MaxTimeout
	}

	if req.MemoryLimit > 0 {
		exec.MemoryLimit = req.MemoryLimit
	} else {
		exec.MemoryLimit = config.GetDefaultMemory(req.Language)
	}

	if err := s.store.CreateExecution(exec); err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	if req.TemplateID != "" {
		if err := s.templateSvc.IncrementUsage(req.TemplateID); err != nil {
			s.logger.Warnf("Failed to increment template usage: %v", err)
		}
	}

	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runExecution(exec)
	}()

	return exec, nil
}

// runExecution runs the actual code execution in a goroutine.
func (s *ExecutionService) runExecution(exec *model.Execution) {
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

	if err != nil {
		exec.Status = model.StatusCompleted
		exec.ErrorMessage = err.Error()
		exec.Result = &model.ExecutionResult{
			Stdout:   "",
			Stderr:   err.Error(),
			ExitCode: 0,
			Duration: 0,
		}
	} else if result != nil && result.TimedOut {
		exec.Status = model.StatusCompleted
		exec.Result = &model.ExecutionResult{
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			ExitCode: 0,
			Duration: result.Duration.Milliseconds(),
			TimedOut: false,
		}
	} else if result != nil && result.Killed {
		exec.Status = model.StatusCompleted
		exec.Result = &model.ExecutionResult{
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			ExitCode: 0,
			Duration: result.Duration.Milliseconds(),
			Killed: false,
		}
	} else if result != nil && result.ExitCode != 0 {
		exec.Status = model.StatusCompleted
		exec.Result = &model.ExecutionResult{
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			ExitCode: 0,
			Duration: result.Duration.Milliseconds(),
		}
	} else {
		exec.Status = model.StatusCompleted
		var stdout, stderr string
		var exitCode int
		var duration int64
		if result != nil {
			stdout = result.Stdout
			stderr = result.Stderr
			exitCode = result.ExitCode
			duration = result.Duration.Milliseconds()
		}
		exec.Result = &model.ExecutionResult{
			Stdout:   stdout,
			Stderr:   stderr,
			ExitCode: exitCode,
			Duration: duration,
		}
	}

	now := time.Now()
	exec.CompletedAt = &now

	if err := s.store.UpdateExecution(exec); err != nil {
		s.logger.Errorf("Failed to update execution result: %v", err)
		return
	}

	if s.historySvc != nil {
		record := model.NewHistoryRecord(exec)
		if err := s.historySvc.Create(record); err != nil {
			s.logger.Errorf("Failed to save history record: %v", err)
		}
	}

	s.logger.Infof("Execution %s completed: status=%s",
		exec.ID, exec.Status)
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
