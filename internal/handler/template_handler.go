package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/response"
)

// TemplateHandler handles HTTP requests for code templates.
type TemplateHandler struct {
	svc    *service.TemplateService
	logger *logger.Logger
}

// NewTemplateHandler creates a new TemplateHandler.
func NewTemplateHandler(svc *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{
		svc:    svc,
		logger: logger.GetGlobal(),
	}
}

// Create handles POST /api/templates - create a new template.
func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.BadRequest(w, "Method not allowed")
		return
	}

	var req model.TemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	tmpl, err := h.svc.Create(&req)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported language") {
			response.BadRequest(w, err.Error())
		} else if strings.Contains(err.Error(), "validation failed") {
			response.BadRequest(w, err.Error())
		} else {
			response.InternalError(w, err.Error())
		}
		return
	}

	response.Created(w, tmpl)
}

// Get handles GET /api/templates/{id} - get a specific template.
func (h *TemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/templates/")
	if id == "" {
		response.BadRequest(w, "template ID is required")
		return
	}

	tmpl, err := h.svc.Get(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.NotFound(w, err.Error())
		} else {
			response.InternalError(w, err.Error())
		}
		return
	}

	response.Success(w, tmpl)
}

// Update handles PUT /api/templates/{id} - update a template.
func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		response.BadRequest(w, "Method not allowed")
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/templates/")
	if id == "" {
		response.BadRequest(w, "template ID is required")
		return
	}

	var req model.TemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	tmpl, err := h.svc.Update(id, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.NotFound(w, err.Error())
		} else if strings.Contains(err.Error(), "validation failed") {
			response.BadRequest(w, err.Error())
		} else {
			response.InternalError(w, err.Error())
		}
		return
	}

	response.Success(w, tmpl)
}

// Delete handles DELETE /api/templates/{id} - delete a template.
func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.BadRequest(w, "Method not allowed")
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/templates/")
	if id == "" {
		response.BadRequest(w, "template ID is required")
		return
	}

	if err := h.svc.Delete(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.NotFound(w, err.Error())
		} else {
			response.InternalError(w, err.Error())
		}
		return
	}

	response.Deleted(w)
}

// List handles GET /api/templates - list templates with filtering.
func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	query := model.TemplateQuery{
		Language:   r.URL.Query().Get("language"),
		Category:   r.URL.Query().Get("category"),
		Search:     r.URL.Query().Get("search"),
		Tag:        r.URL.Query().Get("tag"),
		PublicOnly: r.URL.Query().Get("public") == "true",
		Page:       parseIntParam(r.URL.Query().Get("page"), 1),
		PageSize:   parseIntParam(r.URL.Query().Get("page_size"), 20),
	}

	templates, total, err := h.svc.List(query)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, model.TemplateListResponse{
		Templates:  templates,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: calculatePages(total, query.PageSize),
	})
}

// Search handles GET /api/templates/search - search templates.
func (h *TemplateHandler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	keyword := r.URL.Query().Get("q")
	if keyword == "" {
		response.BadRequest(w, "search query parameter 'q' is required")
		return
	}

	templates, err := h.svc.Search(keyword)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, map[string]interface{}{
		"keyword":   keyword,
		"templates": templates,
		"count":     len(templates),
	})
}

// ListByLanguage handles GET /api/templates/language/{lang} - list templates by language.
func (h *TemplateHandler) ListByLanguage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	lang := extractIDFromPath(r.URL.Path, "/api/templates/language/")
	if lang == "" {
		response.BadRequest(w, "language is required")
		return
	}

	templates, err := h.svc.GetByLanguage(lang)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, map[string]interface{}{
		"language":  lang,
		"templates": templates,
		"count":     len(templates),
	})
}

// GetPredefined handles GET /api/templates/predefined - get predefined templates.
func (h *TemplateHandler) GetPredefined(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	templates := []model.Template{}
	response.Success(w, map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
	})
}
