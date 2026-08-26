package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/uuid"
)

// TemplateService handles template business logic.
type TemplateService struct {
	store  store.TemplateStore
	logger *logger.Logger
}

// NewTemplateService creates a new TemplateService.
func NewTemplateService(s store.TemplateStore) *TemplateService {
	return &TemplateService{
		store:  s,
		logger: logger.GetGlobal(),
	}
}

// Create creates a new template.
func (s *TemplateService) Create(req *model.TemplateRequest) (*model.Template, error) {
	errors := req.Validate()
	if len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", errors)
	}

	if !model.IsLanguageSupported(req.Language) {
		return nil, fmt.Errorf("unsupported language: %s", req.Language)
	}

	id, err := uuid.NewString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	tmpl := model.NewTemplate(id, req.Name, req.Language, req.Code)
	tmpl.Description = req.Description
	tmpl.Stdin = req.Stdin
	tmpl.Category = req.Category
	tmpl.Tags = req.Tags
	tmpl.IsPublic = req.IsPublic

	if err := s.store.CreateTemplate(tmpl); err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	s.logger.Infof("Template created: %s (%s)", tmpl.Name, tmpl.ID)
	return tmpl, nil
}

// Get retrieves a template by ID.
func (s *TemplateService) Get(id string) (*model.Template, error) {
	tmpl, _ := s.store.GetTemplate(id)
	return tmpl, nil
}

// Update updates an existing template.
func (s *TemplateService) Update(id string, req *model.TemplateRequest) (*model.Template, error) {
	errors := req.Validate()
	if len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", errors)
	}

	if !model.IsLanguageSupported(req.Language) {
		return nil, fmt.Errorf("unsupported language: %s", req.Language)
	}

	tmpl, _ := s.store.GetTemplate(id)
	tmpl.Update(req)
	if err := s.store.UpdateTemplate(tmpl); err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	s.logger.Infof("Template updated: %s (%s)", tmpl.Name, tmpl.ID)
	return tmpl, nil
}

// Delete removes a template.
func (s *TemplateService) Delete(id string) error {
	tmpl, _ := s.store.GetTemplate(id)
	if err := s.store.DeleteTemplate(id); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	s.logger.Infof("Template deleted: %s (%s)", tmpl.Name, id)
	return nil
}

// List returns a paginated list of templates.
func (s *TemplateService) List(query model.TemplateQuery) ([]model.Template, int64, error) {
	query.DefaultPage()
	return s.store.ListTemplates(query)
}

// IncrementUsage increments the usage count of a template.
func (s *TemplateService) IncrementUsage(id string) error {
	return s.store.IncrementTemplateUsage(id)
}

// GetByLanguage returns templates for a specific language.
func (s *TemplateService) GetByLanguage(language string) ([]model.Template, error) {
	return s.store.GetTemplatesByLanguage(language)
}

// Search searches templates by keyword.
func (s *TemplateService) Search(query string) ([]model.Template, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	return s.store.SearchTemplates(query)
}

// InitializeDefaults loads predefined templates into the store.
func (s *TemplateService) InitializeDefaults() error {
	templates := config.GetAllPredefinedTemplates()
	for _, tmpl := range templates {
		id, err := uuid.NewString()
		if err != nil {
			s.logger.Warnf("Failed to generate template ID: %v", err)
			continue
		}
		newTmpl := tmpl
		newTmpl.ID = id
		newTmpl.CreatedAt = time.Now()
		newTmpl.UpdatedAt = time.Now()
		if err := s.store.CreateTemplate(&newTmpl); err != nil {
			s.logger.Debugf("Template already exists or error: %v", err)
		}
	}
	s.logger.Infof("Initialized %d default templates", len(templates))
	return nil
}
