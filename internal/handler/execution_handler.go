// Package handler provides HTTP request handlers for the code sandbox API.
package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/response"
)

// ExecutionHandler handles HTTP requests for code execution.
type ExecutionHandler struct {
	svc        *service.ExecutionService
	validator  *service.CodeValidator
	logger     *logger.Logger
}

// NewExecutionHandler creates a new ExecutionHandler.
func NewExecutionHandler(svc *service.ExecutionService, validator *service.CodeValidator) *ExecutionHandler {
	return &ExecutionHandler{
		svc:       svc,
		validator: validator,
		logger:    logger.GetGlobal(),
	}
}

// Execute handles POST /api/execute - submit code for execution.
func (h *ExecutionHandler) Execute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.BadRequest(w, "Method not allowed")
		return
	}

	var req model.ExecutionRequest
	if err := response.DecodeJSONWithContext(r.Context(), r.Body, &req); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "request cancelled") {
			response.BadRequest(w, errMsg)
		} else {
			response.BadRequest(w, fmt.Sprintf("invalid request body: %v", err))
		}
		return
	}

	validationErrors := req.Validate()
	if len(validationErrors) > 0 {
		response.BadRequest(w, fmt.Sprintf("validation failed: %v", validationErrors))
		return
	}

	safetyResult := h.validator.Validate(req.Code, req.Language)
	if !safetyResult.Valid {
		response.BadRequest(w, fmt.Sprintf("code safety check failed: %v", safetyResult.Errors))
		return
	}

	exec, err := h.svc.Execute(r.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported language") {
			response.BadRequest(w, err.Error())
		} else {
			response.InternalError(w, fmt.Sprintf("failed to execute code: %v", err))
		}
		return
	}

	response.Success(w, exec)
}

// GetResult handles GET /api/execute/{id} - get execution result.
func (h *ExecutionHandler) GetResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	// Extract ID from URL path
	id := extractIDFromPath(r.URL.Path, "/api/execute/")
	if id == "" {
		response.BadRequest(w, "execution ID is required")
		return
	}

	exec, err := h.svc.GetResult(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.NotFound(w, err.Error())
		} else {
			response.InternalError(w, err.Error())
		}
		return
	}

	response.Success(w, exec)
}

// Cancel handles DELETE /api/execute/{id} - cancel a running execution.
func (h *ExecutionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.BadRequest(w, "Method not allowed")
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/execute/")
	if id == "" {
		response.BadRequest(w, "execution ID is required")
		return
	}

	if err := h.svc.Cancel(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.NotFound(w, err.Error())
		} else {
			response.BadRequest(w, err.Error())
		}
		return
	}

	response.SuccessMessage(w, "execution cancelled", nil)
}

// List handles GET /api/execute - list executions with filtering.
func (h *ExecutionHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	filter := model.ExecutionFilter{
		Language:    r.URL.Query().Get("language"),
		Status:      r.URL.Query().Get("status"),
		SubmittedBy: r.URL.Query().Get("submitted_by"),
		Page:        parseIntParam(r.URL.Query().Get("page"), 1),
		PageSize:    parseIntParam(r.URL.Query().Get("page_size"), 20),
	}

	executions, total, err := h.svc.List(filter)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, map[string]interface{}{
		"executions": executions,
		"total":      total,
		"page":       filter.Page,
		"page_size":  filter.PageSize,
	})
}

// BatchExecute handles POST /api/execute/batch - submit multiple code snippets.
func (h *ExecutionHandler) BatchExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.BadRequest(w, "Method not allowed")
		return
	}

	var reqs []model.ExecutionRequest
	if err := response.DecodeJSONWithContext(r.Context(), r.Body, &reqs); err != nil {
		response.BadRequest(w, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if len(reqs) > 10 {
		response.BadRequest(w, "maximum 10 executions per batch")
		return
	}

	var results []*model.Execution
	for _, req := range reqs {
		exec, err := h.svc.Execute(r.Context(), &req)
		if err != nil {
			h.logger.Warnf("Batch execution failed for language %s: %v", req.Language, err)
			continue
		}
		results = append(results, exec)
	}

	response.Success(w, map[string]interface{}{
		"total":   len(reqs),
		"success": len(results),
		"results": results,
	})
}

// GetStats handles GET /api/execute/stats - get execution statistics.
func (h *ExecutionHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	historySvc := service.NewHistoryService(nil) // Will be initialized properly
	stats, err := historySvc.GetStats()
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, stats)
}

// Helper functions

// extractIDFromPath extracts an ID from a URL path after a prefix.
func extractIDFromPath(path, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}

// parseIntParam parses an integer query parameter with a default.
func parseIntParam(value string, defaultVal int) int {
	if value == "" {
		return defaultVal
	}
	var n int
	_, err := fmt.Sscanf(value, "%d", &n)
	if err != nil || n <= 0 {
		return defaultVal
	}
	return n
}

// Ensure time is used (imported for potential future use)
var _ = time.Now
