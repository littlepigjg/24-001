package model

import (
	"fmt"
	"time"
)

type CreateReq struct {
	RawURL     string
	CustomCode string
	MaxVisits  int
}

type ShortURL struct {
	Code      string
	RawURL    string
	CreatedAt time.Time
	Visits    int
	MaxVisits int
	Custom    bool
	Disabled  bool
}

func (r *CreateReq) Validate() error {
	if r.RawURL == "" {
		return fmt.Errorf("raw URL is required")
	}
	if len(r.RawURL) > 2048 {
		return fmt.Errorf("raw URL exceeds maximum length of 2048")
	}
	if r.CustomCode != "" && len(r.CustomCode) > 16 {
		return fmt.Errorf("custom code exceeds maximum length of 16")
	}
	if r.MaxVisits < 0 {
		return fmt.Errorf("max visits cannot be negative")
	}
	return nil
}

func (s *ShortURL) Validate() error {
	if s.Code == "" {
		return fmt.Errorf("code is required")
	}
	if s.RawURL == "" {
		return fmt.Errorf("raw URL is required")
	}
	if len(s.Code) > 16 {
		return fmt.Errorf("code exceeds maximum length of 16")
	}
	return nil
}

func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.MaxVisits > 0 && s.Visits >= s.MaxVisits {
		return true
	}
	return now.Sub(s.CreatedAt) > 365*24*time.Hour
}

func (s *ShortURL) IncrVisits() {
	s.Visits++
}

func (s *ShortURL) Disable() {
	s.Disabled = true
}

func (s *ShortURL) Enable() {
	s.Disabled = false
}
