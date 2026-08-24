package model

import (
	"fmt"
	"time"
	"net/url"
)

// CreateReq represents a request to create a short URL.
type CreateReq struct {
	RawURL     string `json:"raw_url"`
	CustomCode string `json:"custom_code,omitempty"`
	MaxVisits  int    `json:"max_visits,omitempty"`
}

// Validate validates the create request.
func (r *CreateReq) Validate() error {
	if r.RawURL == "" {
		return fmt.Errorf("raw URL is required")
	}
	parsed, err := url.Parse(r.RawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL must start with http or https")
	}
	if r.CustomCode != "" {
		if len(r.CustomCode) < 4 || len(r.CustomCode) > 16 {
			return fmt.Errorf("custom code must be between 4 and 16 characters")
		}
	}
	return nil
}

// ShortURL represents a shortened URL entry.
type ShortURL struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	CreatedAt time.Time `json:"created_at"`
	Visits    int       `json:"visits"`
	Custom    bool      `json:"custom"`
	Disabled  bool      `json:"disabled"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// Validate validates the ShortURL.
func (s *ShortURL) Validate() error {
	if s.Code == "" {
		return fmt.Errorf("code is required")
	}
	if s.RawURL == "" {
		return fmt.Errorf("raw URL is required")
	}
	return nil
}

// IsExpired checks if the short URL is expired.
func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.ExpiresAt.IsZero() {
		return false
	}
	return now.After(s.ExpiresAt)
}

// RedirectRequest represents a redirect request.
type RedirectRequest struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

// RedirectResult represents a redirect result.
type RedirectResult struct {
	RawURL  string `json:"raw_url"`
	Status  int    `json:"status"`
}
