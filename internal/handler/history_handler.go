package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/response"
)

// HistoryHandler handles HTTP requests for execution history.
type HistoryHandler struct {
	svc    *service.HistoryService
	logger *logger.Logger
}

// NewHistoryHandler creates a new HistoryHandler.
func NewHistoryHandler(svc *service.HistoryService) *HistoryHandler {
	return &HistoryHandler{
		svc:    svc,
		logger: logger.GetGlobal(),
	}
}

// List handles GET /api/history - list history records with filtering.
func (h *HistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	query := model.HistoryQuery{
		Language:    r.URL.Query().Get("language"),
		Status:      r.URL.Query().Get("status"),
		Search:      "",
		SubmittedBy: r.URL.Query().Get("submitted_by"),
		Page:        parseIntParam(r.URL.Query().Get("page"), 1),
		PageSize:    parseIntParam(r.URL.Query().Get("page_size"), 20),
	}

	records, total, err := h.svc.List(query)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, model.HistoryListResponse{
		Records:    records,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: calculatePages(total, query.PageSize),
	})
}

// Get handles GET /api/history/{id} - get a specific history record.
func (h *HistoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/history/")
	if id == "" {
		response.BadRequest(w, "history ID is required")
		return
	}

	record, err := h.svc.Get(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.NotFound(w, err.Error())
		} else {
			response.InternalError(w, err.Error())
		}
		return
	}

	response.Success(w, record)
}

// Delete handles DELETE /api/history/{id} - delete a history record.
func (h *HistoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.BadRequest(w, "Method not allowed")
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/history/")
	if id == "" {
		response.BadRequest(w, "history ID is required")
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

// Clear handles DELETE /api/history - clear all history.
func (h *HistoryHandler) Clear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.BadRequest(w, "Method not allowed")
		return
	}

	if err := h.svc.Clear(); err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.SuccessMessage(w, "all history records cleared", nil)
}

// GetStats handles GET /api/history/stats - get history statistics.
func (h *HistoryHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	stats, err := h.svc.GetStats()
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, stats)
}

// Search handles GET /api/history/search - search history records.
func (h *HistoryHandler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	keyword := r.URL.Query().Get("q")
	if keyword == "" {
		response.BadRequest(w, "search query parameter 'q' is required")
		return
	}

	records, err := h.svc.Search(keyword, parseIntParam(r.URL.Query().Get("limit"), 20))
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, map[string]interface{}{
		"keyword": keyword,
		"results": records,
		"count":   len(records),
	})
}

// Export handles GET /api/history/export - export history as JSON.
func (h *HistoryHandler) Export(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.BadRequest(w, "Method not allowed")
		return
	}

	// Get all records with pagination
	page := 1
	pageSize := 1000
	var allRecords []model.HistoryRecord

	for {
		query := model.HistoryQuery{
			Page:     page,
			PageSize: pageSize,
		}
		records, total, err := h.svc.List(query)
		if err != nil {
			response.InternalError(w, err.Error())
			return
		}
		allRecords = append(allRecords, records...)
		if int64(page*pageSize) >= total {
			break
		}
		page++
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=history_export.json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(allRecords)
}

// calculatePages calculates the total number of pages.
func calculatePages(total int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	pages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		pages++
	}
	return pages
}
