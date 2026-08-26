package model

import (
	"fmt"
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
		return fmt.Errorf("raw_url is required")
	}
	if len(r.RawURL) > 2048 {
		return fmt.Errorf("raw_url exceeds maximum length of 2048 characters")
	}
	if r.MaxVisits < 0 {
		return fmt.Errorf("max_visits cannot be negative")
	}
	if r.MaxVisits > 1000000 {
		return fmt.Errorf("max_visits cannot exceed 1000000")
	}
	if r.CustomCode != "" {
		if len(r.CustomCode) < 4 || len(r.CustomCode) > 16 {
			return fmt.Errorf("custom_code must be between 4 and 16 characters")
		}
		for _, c := range r.CustomCode {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
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
	if strings.TrimSpace(s.Code) == "" {
		return fmt.Errorf("code is required")
	}
	if strings.TrimSpace(s.RawURL) == "" {
		return fmt.Errorf("raw_url is required")
	}
	if len(s.Code) < 4 || len(s.Code) > 16 {
		return fmt.Errorf("code must be between 4 and 16 characters")
	}
	if s.Visits < 0 {
		return fmt.Errorf("visits cannot be negative")
	}
	return nil
}

func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.CreatedAt.IsZero() {
		return false
	}
	return now.Sub(s.CreatedAt) > 365*24*time.Hour
}

type RedirectRequest struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

type RedirectResult struct {
	RawURL string `json:"raw_url"`
	Status int    `json:"status"`
}
