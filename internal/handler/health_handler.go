package handler

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/response"
)

// HealthHandler handles health check and readiness endpoints.
type HealthHandler struct {
	startTime time.Time
	logger    *logger.Logger
	version   string
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
		logger:    logger.GetGlobal(),
		version:   "1.0.0",
	}
}

// Health handles GET /health - basic health check.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]bool)
	checks["memory"] = true
	checks["store"] = true
	checks["executor"] = true

	status := "healthy"
	for _, ok := range checks {
		if !ok {
			status = "unhealthy"
			break
		}
	}

	resp := &model.HealthResponse{
		Status:  status,
		Version: h.version,
		Uptime:  time.Since(h.startTime),
		Checks:  checks,
	}

	response.Success(w, resp)
}

// Ready handles GET /ready - readiness check.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]bool)
	checks["server"] = true
	checks["sandbox"] = true
	checks["languages"] = true

	ready := true
	for _, ok := range checks {
		if !ok {
			ready = false
			break
		}
	}

	resp := &model.ReadyResponse{
		Ready:   ready,
		Checks:  checks,
		Message: fmt.Sprintf("Server is %s", map[bool]string{true: "ready", false: "not ready"}[ready]),
	}

	if ready {
		response.Success(w, resp)
	} else {
		response.ServiceUnavailable(w, resp.Message)
	}
}

// Status handles GET /api/status - detailed system status.
func (h *HealthHandler) Status(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	status := &model.SystemStatus{
		Status:    "healthy",
		Version:   h.version,
		Uptime:    time.Since(h.startTime),
		StartTime: h.startTime,
		CPUCores:  runtime.NumCPU(),
		Features: map[string]bool{
			"templates":  true,
			"history":    true,
			"stats":      true,
			"sandbox":    true,
		},
	}

	response.Success(w, status)
}

// Version handles GET /api/version - get API version.
func (h *HealthHandler) Version(w http.ResponseWriter, r *http.Request) {
	response.Success(w, map[string]interface{}{
		"version": h.version,
		"go":      runtime.Version(),
		"os":      runtime.GOOS,
		"arch":    runtime.GOARCH,
	})
}
