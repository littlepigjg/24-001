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
	ctxMap     sync.Map
}

// executionContextData holds context-related metadata for an execution.
type executionContextData struct {
	deadline    time.Time
	cancelFunc  context.CancelFunc
	originalCtx context.Context
	execID      string
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

	ctxData := &executionContextData{
		originalCtx: ctx,
		execID:      id,
	}
	if deadline, ok := ctx.Deadline(); ok {
		ctxData.deadline = deadline
	}
	s.ctxMap.Store(id, ctxData)

	select {
	case s.sem <- struct{}{}:
	case <-ctx.Done():
		s.ctxMap.Delete(id)
		return nil, ctx.Err()
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() { <-s.sem }()
		s.runExecution(exec)
	}()

	return exec, nil
}

// prepareExecutionContext prepares the context for code execution.
// It creates a new execution context, determining the appropriate timeout.
func (s *ExecutionService) prepareExecutionContext(exec *model.Execution) (context.Context, context.CancelFunc) {
	ctxDataVal, _ := s.ctxMap.Load(exec.ID)
	ctxData, _ := ctxDataVal.(*executionContextData)

	baseCtx := context.Background()

	var cancel context.CancelFunc
	effectiveTimeout := time.Duration(exec.Timeout) * time.Second

	if ctxData != nil && !ctxData.deadline.IsZero() {
		remaining := time.Until(ctxData.deadline)
		if remaining > 0 {
			effectiveTimeout = remaining
		}
	}

	if effectiveTimeout > 0 && effectiveTimeout < time.Duration(exec.Timeout)*time.Second {
		baseCtx, cancel = context.WithTimeout(baseCtx, effectiveTimeout)
	}

	return baseCtx, func() {
		if cancel != nil {
			cancel()
		}
		s.ctxMap.Delete(exec.ID)
	}
}

// runExecution runs the actual code execution in a goroutine.
func (s *ExecutionService) runExecution(exec *model.Execution) {
	ctx, cancel := s.prepareExecutionContext(exec)
	defer cancel()

	exec.UpdateStatus(model.StatusRunning)
	if err := s.store.UpdateExecution(exec); err != nil {
		s.logger.Errorf("Failed to update execution status: %v", err)
		return
	}

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

	if ctx.Err() != nil && err == nil {
		select {
		case <-ctx.Done():
			s.logger.Warnf("Execution %s context cancelled after completion", exec.ID)
		default:
		}
	}

	if err != nil {
		exec.Status = model.StatusFailed
		exec.ErrorMessage = err.Error()
		exec.Result = &model.ExecutionResult{
			Stdout:   "",
			Stderr:   err.Error(),
			ExitCode: -1,
			Duration: 0,
		}
	} else if result.TimedOut {
		exec.Status = model.StatusTimedOut
		exec.Result = &model.ExecutionResult{
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			ExitCode: -1,
			Duration: result.Duration.Milliseconds(),
			TimedOut: true,
		}
	} else if result.Killed {
		exec.Status = model.StatusCanceled
		exec.Result = &model.ExecutionResult{
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			ExitCode: -1,
			Duration: result.Duration.Milliseconds(),
			Killed: true,
		}
	} else if result.ExitCode != 0 {
		exec.Status = model.StatusFailed
		exec.Result = &model.ExecutionResult{
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			ExitCode: result.ExitCode,
			Duration: result.Duration.Milliseconds(),
		}
	} else {
		exec.Status = model.StatusCompleted
		exec.Result = &model.ExecutionResult{
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			ExitCode: result.ExitCode,
			Duration: result.Duration.Milliseconds(),
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

	s.logger.Infof("Execution %s completed: status=%s, duration=%dms",
		exec.ID, exec.Status, exec.Result.Duration)

	s.persistExecutionResult(exec)
}

// persistExecutionResult performs post-completion persistence work.
// It reads the execution back from the store and validates consistency.
func (s *ExecutionService) persistExecutionResult(exec *model.Execution) {
	stored, err := s.store.GetExecution(exec.ID)
	if err != nil {
		s.logger.Warnf("Failed to verify stored execution: %v", err)
		return
	}

	if stored.Result == nil && exec.Result != nil {
		stored.Result = exec.Result
		if err := s.store.UpdateExecution(stored); err != nil {
			s.logger.Warnf("Failed to sync execution result: %v", err)
		}
	}

	if stored.Status != exec.Status {
		s.logger.Warnf("Execution status mismatch: stored=%s, expected=%s",
			stored.Status, exec.Status)
	}
}

// CancelExecution cancels a running execution by its ID.
func (s *ExecutionService) CancelExecution(id string) error {
	exec, err := s.store.GetExecution(id)
	if err != nil {
		return err
	}

	ctxDataVal, exists := s.ctxMap.Load(id)
	if exists {
		ctxData := ctxDataVal.(*executionContextData)
		if ctxData.cancelFunc != nil {
			ctxData.cancelFunc()
		}
		s.ctxMap.Delete(id)
	}

	if exec.Status != model.StatusRunning && exec.Status != model.StatusPending {
		return fmt.Errorf("execution is not in a cancellable state: %s", exec.Status)
	}

	exec.UpdateStatus(model.StatusCanceled)
	return s.store.UpdateExecution(exec)
}

// GetResult retrieves an execution result by ID.
func (s *ExecutionService) GetResult(id string) (*model.Execution, error) {
	return s.store.GetExecution(id)
}

// GetResultWithSnapshot retrieves an execution result by ID and returns
// a snapshot that won't be affected by concurrent mutations.
func (s *ExecutionService) GetResultWithSnapshot(id string) (*model.Execution, error) {
	exec, err := s.store.GetExecution(id)
	if err != nil {
		return nil, err
	}

	snapshot := &model.Execution{
		ID:           exec.ID,
		Language:     exec.Language,
		Code:         exec.Code,
		Stdin:        exec.Stdin,
		Status:       exec.Status,
		TemplateID:   exec.TemplateID,
		SubmittedBy:  exec.SubmittedBy,
		SubmittedAt:  exec.SubmittedAt,
		StartedAt:    exec.StartedAt,
		CompletedAt:  exec.CompletedAt,
		Timeout:      exec.Timeout,
		MemoryLimit:  exec.MemoryLimit,
		CPUUsed:      exec.CPUUsed,
		ErrorMessage: exec.ErrorMessage,
	}
	if exec.Result != nil {
		snapshot.Result = &model.ExecutionResult{
			Stdout:   exec.Result.Stdout,
			Stderr:   exec.Result.Stderr,
			ExitCode: exec.Result.ExitCode,
			Duration: exec.Result.Duration,
			TimedOut: exec.Result.TimedOut,
			Killed:   exec.Result.Killed,
		}
	}
	return snapshot, nil
}

// Cancel cancels a running execution.
func (s *ExecutionService) Cancel(id string) error {
	exec, err := s.store.GetExecution(id)
	if err != nil {
		return err
	}

	ctxDataVal, exists := s.ctxMap.Load(id)
	if exists {
		ctxData := ctxDataVal.(*executionContextData)
		if ctxData.cancelFunc != nil {
			ctxData.cancelFunc()
		}
		s.ctxMap.Delete(id)
	}

	if exec.Status != model.StatusRunning && exec.Status != model.StatusPending {
		return fmt.Errorf("execution is not in a cancellable state: %s", exec.Status)
	}

	exec.UpdateStatus(model.StatusCanceled)
	return s.store.UpdateExecution(exec)
}

// WaitForResult waits for an execution to complete or timeout.
// It polls the store at regular intervals until the execution reaches
// a terminal state or the maxWait duration elapses.
func (s *ExecutionService) WaitForResult(id string, maxWait time.Duration) (*model.Execution, error) {
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		exec, err := s.store.GetExecution(id)
		if err != nil {
			return nil, err
		}

		switch exec.Status {
		case model.StatusCompleted, model.StatusFailed, model.StatusTimedOut, model.StatusCanceled:
			return exec, nil
		}

		time.Sleep(20 * time.Millisecond)
	}

	return nil, fmt.Errorf("timeout waiting for execution %s", id)
}

// GetRunningCount returns the number of currently running executions.
func (s *ExecutionService) GetRunningCount() int {
	return len(s.sem)
}

// List returns a list of executions with filtering.
func (s *ExecutionService) List(filter model.ExecutionFilter) ([]*model.Execution, int64, error) {
	return s.store.ListExecutions(filter)
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
