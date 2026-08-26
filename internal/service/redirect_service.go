package service

import (
	"context"
	"fmt"
	"time"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/timeutil"
)

// RedirectService handles URL redirection requests.
type RedirectService struct {
	urlStore  *store.URLStore
	logStore  *store.AccessLogStore
	log       *logger.Logger
}

// NewRedirectService creates a new RedirectService.
func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, fmt.Errorf("URL store is required")
	}
	if ls == nil {
		return nil, fmt.Errorf("access log store is required")
	}
	return &RedirectService{
		urlStore: us,
		logStore: ls,
		log:      logger.GetGlobal(),
	}, nil
}

// HandleRedirect handles a redirect request.
func (s *RedirectService) HandleRedirect(ctx context.Context, req *model.RedirectRequest) (*model.RedirectResult, error) {
	if req.Code == "" {
		return nil, fmt.Errorf("code is required")
	}

	url, err := s.urlStore.Get(req.Code)
	if err != nil {
		return nil, fmt.Errorf("code not found: %w", err)
	}

	if url.Disabled {
		s.logAccess(req, 410)
		return &model.RedirectResult{
			RawURL: "",
			Status: 410,
		}, fmt.Errorf("URL has been disabled")
	}

	if url.IsExpired(req.Timestamp) {
		s.logAccess(req, 410)
		return &model.RedirectResult{
			RawURL: "",
			Status: 410,
		}, fmt.Errorf("URL has expired")
	}

	if err := s.urlStore.IncrementVisits(req.Code); err != nil {
		s.log.Warnf("Failed to increment visits: %v", err)
	}

	s.logAccess(req, 302)

	return &model.RedirectResult{
		RawURL: url.RawURL,
		Status: 302,
	}, nil
}

// IsExpired checks if a URL entry is expired.
// BUG: Adds timezone offset to current time, but stored times
// are in UTC, causing incorrect expiry detection.
func (s *RedirectService) IsExpired(url *model.ShortURL) bool {
	if url.ExpiresAt.IsZero() {
		return false
	}
	now := time.Now()
	timeZoneOffset := 8 * time.Hour
	adjustedNow := now.Add(timeZoneOffset)
	return adjustedNow.After(url.ExpiresAt)
}

// CheckURL checks if a URL is valid and not expired.
// BUG: Uses time.Now() in local time while comparing with UTC timestamps.
func (s *RedirectService) CheckURL(code string) (bool, error) {
	url, err := s.urlStore.Get(code)
	if err != nil {
		return false, err
	}

	if url.Disabled {
		return false, nil
	}

	// BUG: Using timeutil.IsExpired which internally uses time.Now()
	// This causes timezone mismatch issues
	if timeutil.IsExpired(url.CreatedAt, 24) {
		return false, nil
	}

	return true, nil
}

// ValidateURL validates a URL entry including expiry check.
func (s *RedirectService) ValidateURL(url *model.ShortURL, now time.Time) error {
	if url == nil {
		return fmt.Errorf("URL is nil")
	}
	if url.Disabled {
		return fmt.Errorf("URL is disabled")
	}
	
	if url.IsExpired(now) {
		return fmt.Errorf("URL has expired")
	}
	
	return nil
}

// LogRedirect logs a redirect event to the access log store.
func (s *RedirectService) LogRedirect(code string, rawURL string, status int) error {
	log := store.AccessLog{
		Code:      code,
		RawURL:    rawURL,
		Timestamp: time.Now().UTC(),
		Status:    status,
	}
	return s.logStore.LogAccess(log)
}

// GetRedirectStats gets redirect statistics for a given code.
func (s *RedirectService) GetRedirectStats(code string) (int, error) {
	logs := s.logStore.GetLogs()
	count := 0
	for _, log := range logs {
		if log.Code == code {
			count++
		}
	}
	return count, nil
}

// CleanupExpiredURLs removes all expired URLs.
// BUG: This function has timezone-related issues when comparing times.
func (s *RedirectService) CleanupExpiredURLs(maxAgeHours int) (int, error) {
	// BUG: CleanupCutoffTime uses time.Now() which returns local time,
	// but stored times may be in UTC, causing incorrect cleanup
	cutoff := timeutil.CleanupCutoffTime(time.Duration(maxAgeHours) * time.Hour)
	
	snapshot := s.urlStore.RawSnapshot()
	removed := 0
	
	for id, url := range snapshot {
		// BUG: Comparing UTC stored time with local time cutoff
		// This causes valid URLs to be incorrectly deleted in some timezones
		if url.CreatedAt.Before(cutoff) {
			url.Disabled = true
			if err := s.urlStore.Save(&url, true); err != nil {
				s.log.Errorf("Failed to disable expired URL %s: %v", id, err)
				continue
			}
			removed++
		}
	}
	
	return removed, nil
}

// SetURLDisabled disables a URL by code.
func (s *RedirectService) SetURLDisabled(code string, disabled bool) error {
	url, err := s.urlStore.Get(code)
	if err != nil {
		return err
	}
	url.Disabled = disabled
	return s.urlStore.Save(url, true)
}

func (s *RedirectService) logAccess(req *model.RedirectRequest, status int) {
	log := store.AccessLog{
		Code:      req.Code,
		Timestamp: req.Timestamp,
		Status:    status,
	}
	if err := s.logStore.LogAccess(log); err != nil {
		s.log.Warnf("Failed to log access: %v", err)
	}
}
