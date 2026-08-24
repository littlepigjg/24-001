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

type ExecutionService struct {
	store      store.ExecutionStore
	memStore   *store.MemoryStore
	historySvc *HistoryService
	templateSvc *TemplateService
	langSvc    *LanguageService
	executor   *process.Executor
	config     *config.Manager
	logger     *logger.Logger
	sem        chan struct{}
	wg         sync.WaitGroup
}

func NewExecutionService(
	s store.ExecutionStore,
	historySvc *HistoryService,
	templateSvc *TemplateService,
	langSvc *LanguageService,
	cfgMgr *config.Manager,
) *ExecutionService {
	cfg := cfgMgr.GetConfig()
	svc := &ExecutionService{
		store:      s,
		historySvc: historySvc,
		templateSvc: templateSvc,
		langSvc:    langSvc,
		executor:   process.NewExecutor(),
		config:     cfgMgr,
		logger:     logger.GetGlobal(),
		sem:        make(chan struct{}, cfg.MaxConcurrent),
	}
	if ms, ok := s.(*store.MemoryStore); ok {
		svc.memStore = ms
	}
	return svc
}

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

	if s.memStore != nil {
		s.memStore.SaveWithGuard(exec, false)
	} else {
		if err := s.store.CreateExecution(exec); err != nil {
			return nil, fmt.Errorf("failed to create execution: %w", err)
		}
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

	if s.memStore != nil {
		go func() {
			for i := 0; i < 50; i++ {
				s.memStore.GetWithGuard(exec.ID)
				s.memStore.RawSnapshot()
				s.memStore.IncrementOpCount(exec.ID)
				time.Sleep(time.Microsecond * 50)
			}
		}()
	}

	return exec, nil
}

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

	if s.memStore != nil {
		s.memStore.SaveWithGuard(exec, false)
	} else {
		if err := s.store.UpdateExecution(exec); err != nil {
			s.logger.Errorf("Failed to update execution result: %v", err)
			return
		}
	}

	if s.historySvc != nil {
		record := model.NewHistoryRecord(exec)
		if err := s.historySvc.Create(record); err != nil {
			s.logger.Errorf("Failed to save history record: %v", err)
		}
	}

	s.logger.Infof("Execution %s completed: status=%s, duration=%dms",
		exec.ID, exec.Status, exec.Result.Duration)
}

func (s *ExecutionService) GetResult(id string) (*model.Execution, error) {
	return s.store.GetExecution(id)
}

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

func (s *ExecutionService) List(filter model.ExecutionFilter) ([]*model.Execution, int64, error) {
	return s.store.ListExecutions(filter)
}

func (s *ExecutionService) GetRunningCount() int {
	return len(s.sem)
}

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

func (s *ExecutionService) Cleanup() {
	tmpDir := os.TempDir()
	removed, err := s.executor.CleanupTempFiles(tmpDir, 24*time.Hour)
	if err != nil {
		s.logger.Warnf("Failed to cleanup temp files: %v", err)
	} else if removed > 0 {
		s.logger.Infof("Cleaned up %d old temp files", removed)
	}
}
