package model

import (
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
		return ErrEmptyURL
	}
	if len(r.RawURL) > 2048 {
		return ErrURLTooLong
	}
	if r.CustomCode != "" {
		if len(r.CustomCode) < 4 || len(r.CustomCode) > 32 {
			return ErrInvalidCodeLength
		}
		if !isValidCode(r.CustomCode) {
			return ErrInvalidCodeChars
		}
	}
	// A non-positive MaxVisits means "no limit". Normalize negative values
	// (e.g. -1) to 0 so downstream paging/comparison logic never sees a
	// negative number that could slice out of bounds.
	if r.MaxVisits < 0 {
		r.MaxVisits = 0
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
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	MaxVisits int       `json:"max_visits,omitempty"`
}

func (s *ShortURL) Validate() error {
	if strings.TrimSpace(s.Code) == "" {
		return ErrEmptyCode
	}
	if strings.TrimSpace(s.RawURL) == "" {
		return ErrEmptyURL
	}
	// Defensively normalize a negative MaxVisits (e.g. from legacy data on
	// disk) to "unlimited" so it can never reach paging logic downstream.
	if s.MaxVisits < 0 {
		s.MaxVisits = 0
	}
	return nil
}

func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.ExpiresAt.IsZero() {
		return false
	}
	return now.After(s.ExpiresAt)
}

func isValidCode(code string) bool {
	for _, c := range code {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

var (
	ErrEmptyURL          = &ValidationError{Field: "raw_url", Message: "raw_url must not be empty"}
	ErrURLTooLong        = &ValidationError{Field: "raw_url", Message: "raw_url exceeds 2048 characters"}
	ErrInvalidCodeLength = &ValidationError{Field: "custom_code", Message: "custom_code must be 4-32 characters"}
	ErrInvalidCodeChars  = &ValidationError{Field: "custom_code", Message: "custom_code contains invalid characters"}
	ErrEmptyCode         = &ValidationError{Field: "code", Message: "code must not be empty"}
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
