package model

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// customCodeForbiddenSeqs lists byte sequences that must never appear in a
// custom short code. Any one of them enables path traversal or null-byte
// tricks against the on-disk store, so a code containing them is rejected
// outright rather than silently rewritten.
var customCodeForbiddenSeqs = []string{"..", "/", "\\", "\x00", "."}

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
		if err := validateCustomCode(r.CustomCode); err != nil {
			return err
		}
	}
	return nil
}

// validateCustomCode enforces the strict whitelist for custom short codes:
// only ASCII letters and digits plus '-' and '_', length 3..16, and none of
// the forbidden traversal sequences. Rejecting here keeps bad codes from ever
// reaching the store's path construction.
func validateCustomCode(code string) error {
	if len(code) < 3 || len(code) > 16 {
		return fmt.Errorf("custom_code must be between 3 and 16 characters")
	}
	for _, r := range code {
		if r != '-' && r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return fmt.Errorf("custom_code contains invalid character: %q", r)
		}
	}
	lower := strings.ToLower(code)
	for _, seq := range customCodeForbiddenSeqs {
		if strings.Contains(lower, seq) {
			return fmt.Errorf("custom_code contains forbidden sequence: %q", seq)
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
	if err := validateCustomCode(s.Code); err != nil {
		return fmt.Errorf("code: %w", err)
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
