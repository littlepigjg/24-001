package service

import (
	"fmt"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/process"
)

// LanguageService handles language-related business logic.
type LanguageService struct {
	executor *process.Executor
	logger   *logger.Logger
}

// NewLanguageService creates a new LanguageService.
func NewLanguageService(executor *process.Executor) *LanguageService {
	return &LanguageService{
		executor: executor,
		logger:   logger.GetGlobal(),
	}
}

// List returns all supported languages with their availability status.
func (s *LanguageService) List() []model.LanguageInfo {
	languages := model.DefaultLanguages()
	for i := range languages {
		languages[i].Available = s.executor.IsLanguageAvailable(languages[i].ID)
	}
	return languages
}

// Get returns detailed information about a specific language.
func (s *LanguageService) Get(languageID string) (model.LanguageInfo, error) {
	info, ok := model.GetLanguage(languageID)
	if !ok {
		return model.LanguageInfo{}, fmt.Errorf("unsupported language: %s", languageID)
	}
	info.Available = s.executor.IsLanguageAvailable(languageID)
	return info, nil
}

// IsSupported checks if a language is supported.
func (s *LanguageService) IsSupported(languageID string) bool {
	return model.IsLanguageSupported(languageID)
}

// IsAvailable checks if a language runtime is available on the system.
func (s *LanguageService) IsAvailable(languageID string) bool {
	return s.executor.IsLanguageAvailable(languageID)
}

// GetAvailableLanguages returns only languages that are available on the system.
func (s *LanguageService) GetAvailableLanguages() []model.LanguageInfo {
	all := s.List()
	var available []model.LanguageInfo
	for _, lang := range all {
		if lang.Available {
			available = append(available, lang)
		}
	}
	return available
}

// ValidateLanguage checks if a language ID is valid.
func (s *LanguageService) ValidateLanguage(languageID string) error {
	if !model.IsLanguageSupported(languageID) {
		return fmt.Errorf("unsupported language: %s, supported languages: %v",
			languageID, model.GetLanguageIDs())
	}
	return nil
}
