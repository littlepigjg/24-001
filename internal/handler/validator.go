package handler

import (
	"fmt"
	"net/http"

	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/response"
)

// ValidateHandler handles code validation requests.
type ValidateHandler struct {
	validator *service.CodeValidator
	logger    *logger.Logger
}

// NewValidateHandler creates a new ValidateHandler.
func NewValidateHandler(validator *service.CodeValidator) *ValidateHandler {
	return &ValidateHandler{
		validator: validator,
		logger:    logger.GetGlobal(),
	}
}

// ValidateRequest represents a code validation request.
type ValidateRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

// Validate handles POST /api/validate - validate code without executing.
func (h *ValidateHandler) Validate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.BadRequest(w, "Method not allowed")
		return
	}

	defer response.CloseRequestBody(r)

	var req ValidateRequest
	if err := response.DecodeJSONBody(r, &req); err != nil {
		response.BadRequest(w, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if req.Language == "" {
		response.BadRequest(w, "language is required")
		return
	}

	result := h.validator.Validate(req.Code, req.Language)
	response.Success(w, result)
}
