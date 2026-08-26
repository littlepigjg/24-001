// Package main is the entry point for the Code Sandbox server.
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/handler"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/process"
)

const (
	// version is the application version.
	version = "1.0.0"
)

func main() {
	// Initialize logger
	log := logger.NewLogger(os.Stdout, logger.LevelInfo)
	logger.SetGlobal(log)

	log.Infof("Starting Code Sandbox v%s", version)

	// Load configuration
	cfgMgr := config.GetGlobal()
	cfg := cfgMgr.GetConfig()

	// Load from environment
	cfgMgr.LoadFromEnv()
	cfg = cfgMgr.GetConfig()

	// Apply log level from configuration
	logLevel := cfgMgr.GetLogLevel()
	log.SetLevel(logLevel)
	log.Infof("Log level set to: %s", logLevel.String())

	// Validate configuration
	if err := cfgMgr.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize store
	dataStore, err := store.NewStore(cfg.StorageType, cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}

	// Initialize executor
	executor := process.NewExecutor()

	// Initialize services
	langSvc := service.NewLanguageService(executor)
	historySvc := service.NewHistoryService(dataStore)
	templateSvc := service.NewTemplateService(dataStore)
	validator := service.NewCodeValidator()
	execSvc := service.NewExecutionService(dataStore, historySvc, templateSvc, langSvc, cfgMgr)
	sandboxSvc := service.NewSandboxService(executor, cfg.SandboxDir)

	// Initialize sandbox
	if err := sandboxSvc.Initialize(); err != nil {
		log.Warnf("Failed to initialize sandbox: %v", err)
	}

	// Initialize default templates
	if cfg.EnableTemplates {
		if err := templateSvc.InitializeDefaults(); err != nil {
			log.Warnf("Failed to initialize default templates: %v", err)
		}
	}

	// Start periodic cleanup
	sandboxSvc.StartPeriodicCleanup(time.Duration(cfg.CleanupInterval) * time.Minute)

	// Initialize handlers
	execHandler := handler.NewExecutionHandler(execSvc, validator)
	histHandler := handler.NewHistoryHandler(historySvc)
	tmplHandler := handler.NewTemplateHandler(templateSvc)
	langHandler := handler.NewLanguageHandler(langSvc)
	healthHandler := handler.NewHealthHandler()
	validateHandler := handler.NewValidateHandler(validator)

	// Setup routes
	router := handler.SetupRoutes(handler.RouterConfig{
		ExecutionHandler: execHandler,
		HistoryHandler:   histHandler,
		TemplateHandler:  tmplHandler,
		LanguageHandler:  langHandler,
		HealthHandler:    healthHandler,
		ValidateHandler:  validateHandler,
		StaticDir:        "web/static",
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.Address(),
		Handler:      router,
		ReadTimeout:  cfg.GetReadTimeout(),
		WriteTimeout: cfg.GetWriteTimeout(),
		IdleTimeout:  120 * time.Second,
	}

	// Channel to listen for errors from server
	errCh := make(chan error, 1)

	// Start server
	go func() {
		log.Infof("Server listening on %s", cfg.Address())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for interrupt signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Infof("Received signal %v, shutting down...", sig)
	case err := <-errCh:
		log.Errorf("Server error: %v", err)
	case <-time.After(1 * time.Hour * 24 * 365): // Wait indefinitely
	}

	// Graceful shutdown
	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop accepting new connections
	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("Server shutdown error: %v", err)
	}

	// Wait for running executions to complete
	log.Info("Waiting for executions to complete...")
	if err := execSvc.WaitForCompletion(10 * time.Second); err != nil {
		log.Warnf("Timeout waiting for executions: %v", err)
	}

	// Cleanup
	execSvc.Cleanup()
	historySvc.Cleanup(24 * time.Hour)

	log.Info("Server stopped successfully")
}
