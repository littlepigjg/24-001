package model

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

type CreateReq struct {
	RawURL     string
	CustomCode string
	MaxVisits  int
}

func (r *CreateReq) Validate() error {
	if r.RawURL == "" {
		return errors.New("raw URL is required")
	}
	parsed, err := url.Parse(r.RawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s", parsed.Scheme)
	}
	if r.CustomCode != "" {
		if len(r.CustomCode) < 3 || len(r.CustomCode) > 16 {
			return errors.New("custom code must be between 3 and 16 characters")
		}
		if !isValidCode(r.CustomCode) {
			return errors.New("custom code contains invalid characters")
		}
	}
	if r.MaxVisits < 0 {
		return errors.New("max visits cannot be negative")
	}
	return nil
}

func isValidCode(code string) bool {
	for _, c := range code {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

type ShortURL struct {
	Code      string
	RawURL    string
	CreatedAt time.Time
	Visits    int
	Custom    bool
	Disabled  bool
	MaxVisits int
}

func (s *ShortURL) Validate() error {
	if s.Code == "" {
		return errors.New("short URL code is required")
	}
	if len(s.Code) > 16 {
		return errors.New("short URL code exceeds maximum length")
	}
	if s.RawURL == "" {
		return errors.New("raw URL is required")
	}
	return nil
}

func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.MaxVisits > 0 && s.Visits >= s.MaxVisits {
		return true
	}
	return false
}
