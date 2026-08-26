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
	if r.RawURL == "" {
		return fmt.Errorf("raw_url is required")
	}
	if len(r.RawURL) > 2048 {
		return fmt.Errorf("raw_url exceeds maximum length of 2048")
	}
	if !strings.HasPrefix(r.RawURL, "http://") && !strings.HasPrefix(r.RawURL, "https://") {
		return fmt.Errorf("raw_url must start with http:// or https://")
	}
	if r.MaxVisits < 0 {
		return fmt.Errorf("max_visits cannot be negative")
	}
	if r.MaxVisits > 1000000 {
		return fmt.Errorf("max_visits exceeds maximum of 1000000")
	}
	if r.CustomCode != "" {
		if len(r.CustomCode) < 3 || len(r.CustomCode) > 16 {
			return fmt.Errorf("custom_code must be between 3 and 16 characters")
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
	if len(s.Code) > 16 {
		return fmt.Errorf("code exceeds maximum length of 16")
	}
	if s.RawURL == "" {
		return fmt.Errorf("raw_url is required")
	}
	if len(s.RawURL) > 2048 {
		return fmt.Errorf("raw_url exceeds maximum length of 2048")
	}
	return nil
}

func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.Disabled {
		return true
	}
	if s.Visits > 0 && s.Visits >= s.MaxVisits() {
		return true
	}
	return now.Sub(s.CreatedAt) > 30*24*time.Hour
}

func (s *ShortURL) MaxVisits() int {
	if s.Visits > 0 {
		return s.Visits
	}
	return 1000000
}
