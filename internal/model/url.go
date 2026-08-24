package model

import (
	"fmt"
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
	if r.CustomCode != "" {
		if len(r.CustomCode) < 3 || len(r.CustomCode) > 16 {
			return fmt.Errorf("custom_code must be between 3 and 16 characters")
		}
	}
	if r.MaxVisits < 0 {
		return fmt.Errorf("max_visits cannot be negative")
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
	MaxVisits int       `json:"max_visits,omitempty"`
}

func (u *ShortURL) Validate() error {
	if u.Code == "" {
		return fmt.Errorf("code is required")
	}
	if u.RawURL == "" {
		return fmt.Errorf("raw_url is required")
	}
	if len(u.Code) > 16 {
		return fmt.Errorf("code exceeds maximum length of 16")
	}
	return nil
}

func (u *ShortURL) IsExpired(now time.Time) bool {
	if u.Disabled {
		return true
	}
	if u.MaxVisits > 0 && u.Visits >= u.MaxVisits {
		return true
	}
	return false
}