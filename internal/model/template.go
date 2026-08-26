package model

import (
	"time"
)

// Template represents a saved code template/snippet.
type Template struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Language    string    `json:"language"`
	Code        string    `json:"code"`
	Stdin       string    `json:"stdin,omitempty"`
	Category    string    `json:"category,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	IsPublic    bool      `json:"is_public"`
	CreatedBy   string    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	UsageCount  int64     `json:"usage_count"`
	Favorite    bool      `json:"favorite,omitempty"`
}

// TemplateRequest represents a request to create or update a template.
type TemplateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Language    string   `json:"language"`
	Code        string   `json:"code"`
	Stdin       string   `json:"stdin,omitempty"`
	Category    string   `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	IsPublic    bool     `json:"is_public"`
}

// Validate checks if the template request is valid.
func (r *TemplateRequest) Validate() map[string]string {
	errors := make(map[string]string)
	if r.Name == "" {
		errors["name"] = "name is required"
	} else if len(r.Name) > 100 {
		errors["name"] = "name must be at most 100 characters"
	}
	if r.Language == "" {
		errors["language"] = "language is required"
	}
	if r.Code == "" {
		errors["code"] = "code is required"
	}
	if len(r.Code) > 100000 {
		errors["code"] = "code exceeds maximum size of 100000 characters"
	}
	for i, tag := range r.Tags {
		if len(tag) > 50 {
			errors["tags"] = "tag at index " + itoa(i) + " exceeds 50 characters"
		}
	}
	return errors
}

// itoa converts int to string.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}

// TemplateQuery represents a query for templates.
type TemplateQuery struct {
	Language   string `json:"language,omitempty"`
	Category   string `json:"category,omitempty"`
	Search     string `json:"search,omitempty"`
	Tag        string `json:"tag,omitempty"`
	PublicOnly bool   `json:"public_only,omitempty"`
	Page       int    `json:"page,omitempty"`
	PageSize   int    `json:"page_size,omitempty"`
	SortBy     string `json:"sort_by,omitempty"`
}

// DefaultPage returns default pagination values.
func (q *TemplateQuery) DefaultPage() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
}

// TemplateListResponse is the API response for listing templates.
type TemplateListResponse struct {
	Templates  []Template `json:"templates"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

// TemplateWithUsage includes usage statistics for a template.
type TemplateWithUsage struct {
	Template
	RecentExecutions int64 `json:"recent_executions"`
}

// NewTemplate creates a new Template with default values.
func NewTemplate(id, name, language, code string) *Template {
	now := time.Now()
	return &Template{
		ID:        id,
		Name:      name,
		Language:  language,
		Code:      code,
		CreatedAt: now,
		UpdatedAt: now,
		UsageCount: 0,
	}
}

// Update updates the template with new values.
func (t *Template) Update(req *TemplateRequest) {
	t.Name = req.Name
	t.Description = req.Description
	t.Language = req.Language
	t.Code = req.Code
	t.Stdin = req.Stdin
	t.Category = req.Category
	t.Tags = req.Tags
	t.IsPublic = req.IsPublic
	t.UpdatedAt = time.Now()
}

// IncrementUsage increments the usage count.
func (t *Template) IncrementUsage() {
	t.UsageCount++
}
