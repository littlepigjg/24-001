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

func (r *CreateReq) Validate() error {
	if r.RawURL == "" {
		return fmt.Errorf("raw URL is required")
	}
	if len(r.RawURL) > 2048 {
		return fmt.Errorf("raw URL too long")
	}
	if r.MaxVisits < 0 {
		return fmt.Errorf("max visits cannot be negative")
	}
	return nil
}

type ShortURL struct {
	Code      string
	RawURL    string
	CreatedAt time.Time
	Visits    int
	Custom    bool
	Disabled  bool
}

func (u *ShortURL) Validate() error {
	if u.Code == "" {
		return fmt.Errorf("code is required")
	}
	if u.RawURL == "" {
		return fmt.Errorf("raw URL is required")
	}
	return nil
}

func (u *ShortURL) IsExpired(now time.Time) bool {
	if u.CreatedAt.IsZero() {
		return false
	}
	return now.Sub(u.CreatedAt) > 90*24*time.Hour
}

type RedirectRequest struct {
	Code      string
	Timestamp time.Time
}

type RedirectResult struct {
	RawURL string
	Status int
}
