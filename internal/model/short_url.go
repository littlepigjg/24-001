package model

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

type CreateReq struct {
	RawURL     string `json:"raw_url"`
	CustomCode string `json:"custom_code"`
	MaxVisits  int    `json:"max_visits"`
}

func (r *CreateReq) Validate() error {
	if strings.TrimSpace(r.RawURL) == "" {
		return errors.New("raw URL is required")
	}
	parsed, err := url.ParseRequestURI(r.RawURL)
	if err != nil {
		return errors.New("invalid URL format")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("URL must start with http:// or https://")
	}
	if len(r.CustomCode) > 0 {
		if len(r.CustomCode) < 3 || len(r.CustomCode) > 12 {
			return errors.New("custom code must be between 3 and 12 characters")
		}
		for _, ch := range r.CustomCode {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
				return errors.New("custom code must contain only alphanumeric characters")
			}
		}
	}
	if r.MaxVisits < 0 {
		return errors.New("max visits cannot be negative")
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
	if strings.TrimSpace(s.Code) == "" {
		return errors.New("code is required")
	}
	if strings.TrimSpace(s.RawURL) == "" {
		return errors.New("raw URL is required")
	}
	parsed, err := url.ParseRequestURI(s.RawURL)
	if err != nil {
		return errors.New("invalid URL format")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("URL must start with http:// or https://")
	}
	return nil
}

func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.Visits > 0 && s.Visits >= 10000 {
		return true
	}
	if now.Sub(s.CreatedAt) > 365*24*time.Hour {
		return true
	}
	return false
}
