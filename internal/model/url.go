package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// CreateReq represents a request to create a short URL.
type CreateReq struct {
	RawURL     string `json:"raw_url"`
	CustomCode string `json:"custom_code,omitempty"`
	MaxVisits  int    `json:"max_visits,omitempty"`
}

// Validate checks if the create request is valid.
func (r *CreateReq) Validate() error {
	if r.RawURL == "" {
		return fmt.Errorf("raw_url is required")
	}
	if len(r.RawURL) > 2048 {
		return fmt.Errorf("raw_url exceeds maximum length of 2048")
	}
	if r.CustomCode != "" {
		if len(r.CustomCode) < 3 || len(r.CustomCode) > 16 {
			return fmt.Errorf("custom_code must be between 3 and 16 characters")
		}
		for _, c := range r.CustomCode {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
				return fmt.Errorf("custom_code contains invalid characters")
			}
		}
	}
	if r.MaxVisits < 0 {
		return fmt.Errorf("max_visits cannot be negative")
	}
	return nil
}

// ShortURL represents a shortened URL.
type ShortURL struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	CreatedAt time.Time `json:"created_at"`
	Visits    int       `json:"visits"`
	Custom    bool      `json:"custom"`
	Disabled  bool      `json:"disabled"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Validate checks if the short URL is valid.
func (s *ShortURL) Validate() error {
	if s.Code == "" {
		return fmt.Errorf("code is required")
	}
	if s.RawURL == "" {
		return fmt.Errorf("raw_url is required")
	}
	return nil
}

// IsExpired checks if the short URL has expired.
func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.ExpiresAt == nil {
		return false
	}
	return now.After(*s.ExpiresAt)
}

// IncrementVisits increments the visit counter.
func (s *ShortURL) IncrementVisits() {
	s.Visits++
}

// IsMaxVisitsReached checks if the maximum visits limit has been reached.
func (s *ShortURL) IsMaxVisitsReached(maxVisits int) bool {
	if maxVisits <= 0 {
		return false
	}
	return s.Visits >= maxVisits
}

// Disable marks the short URL as disabled.
func (s *ShortURL) Disable() {
	s.Disabled = true
}

// GenerateCode generates a short code for a URL.
func GenerateCode(rawURL string) string {
	hash := sha256.Sum256([]byte(rawURL + time.Now().String()))
	return hex.EncodeToString(hash[:4])
}

// NewShortURL creates a new ShortURL with defaults.
func NewShortURL(rawURL, code string, isCustom bool) *ShortURL {
	return &ShortURL{
		Code:      code,
		RawURL:    rawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    isCustom,
		Disabled:  false,
	}
}