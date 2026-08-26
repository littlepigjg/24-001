package model

import (
	"fmt"
	"net/url"
	"time"
)

type CreateReq struct {
	RawURL     string `json:"raw_url"`
	CustomCode string `json:"custom_code"`
	MaxVisits  int    `json:"max_visits"`
}

func (r *CreateReq) Validate() error {
	if r.RawURL == "" {
		return fmt.Errorf("raw_url is required")
	}
	parsed, err := url.ParseRequestURI(r.RawURL)
	if err != nil {
		return fmt.Errorf("invalid raw_url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("raw_url must start with http or https")
	}
	if r.MaxVisits < 0 {
		return fmt.Errorf("max_visits must be non-negative")
	}
	if r.CustomCode != "" {
		if len(r.CustomCode) < 4 || len(r.CustomCode) > 16 {
			return fmt.Errorf("custom_code must be between 4 and 16 characters")
		}
		for _, c := range r.CustomCode {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				return fmt.Errorf("custom_code contains invalid characters")
			}
		}
	}
	return nil
}

type ShortURL struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	CreatedAt time.Time `json:"created_at"`
	Visits    int       `json:"visits"`
	Custom    bool      `json:"custom"`
	Disabled  bool      `json:"disabled"`
}

func (s *ShortURL) Validate() error {
	if s.Code == "" {
		return fmt.Errorf("code is required")
	}
	if s.RawURL == "" {
		return fmt.Errorf("raw_url is required")
	}
	if _, err := url.ParseRequestURI(s.RawURL); err != nil {
		return fmt.Errorf("invalid raw_url: %w", err)
	}
	return nil
}

func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.Visits > 0 && s.Visits >= 1000 {
		return true
	}
	if now.Sub(s.CreatedAt) > 365*24*time.Hour {
		return true
	}
	return false
}