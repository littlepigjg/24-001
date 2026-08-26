package handler

import (
	"net/http"

	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/response"
)

// LanguageHandler handles HTTP requests for language information.
type LanguageHandler struct {
	svc    *service.LanguageService
	logger *logger.Logger
}

// NewLanguageHandler creates a new LanguageHandler.
func NewLanguageHandler(svc *service.LanguageService) *LanguageHandler {
	return &LanguageHandler{
		svc:    svc,
		logger: logger.GetGlobal(),
	}
}

// List handles GET /api/languages - list all supported languages.
func (h *LanguageHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	languages := h.svc.List()
	response.Success(w, map[string]interface{}{
		"languages": languages,
		"count":     len(languages),
	})
}

// Get handles GET /api/languages/{id} - get detailed information about a language.
func (h *LanguageHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/languages/")
	if id == "" {
		response.BadRequest(w, "language ID is required")
		return
	}

	info, err := h.svc.Get(id)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	response.Success(w, info)
}

// GetAvailable handles GET /api/languages/available - get available languages.
func (h *LanguageHandler) GetAvailable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	languages := h.svc.GetAvailableLanguages()
	response.Success(w, map[string]interface{}{
		"languages": languages,
		"count":     len(languages),
	})
}
