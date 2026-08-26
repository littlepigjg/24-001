package model

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

type CreateReq struct {
	RawURL     string
	CustomCode string
	MaxVisits  int
}

func (r *CreateReq) Validate() error {
	if r == nil {
		return errors.New("nil request")
	}
	if strings.TrimSpace(r.RawURL) == "" {
		return errors.New("raw url is required")
	}
	if _, err := url.ParseRequestURI(r.RawURL); err != nil {
		return errors.New("invalid raw url")
	}
	if r.CustomCode != "" {
		if len(r.CustomCode) < 4 || len(r.CustomCode) > 16 {
			return errors.New("custom code length must be between 4 and 16")
		}
		for _, ch := range r.CustomCode {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
				return errors.New("custom code contains invalid characters")
			}
		}
	}
	if r.MaxVisits < 0 {
		return errors.New("max visits must be non-negative")
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
	maxVisits int
}

func (s *ShortURL) Validate() error {
	if s == nil {
		return errors.New("nil short url")
	}
	if s.Code == "" {
		return errors.New("code is required")
	}
	if s.RawURL == "" {
		return errors.New("raw url is required")
	}
	if _, err := url.ParseRequestURI(s.RawURL); err != nil {
		return errors.New("invalid raw url")
	}
	return nil
}

func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.Disabled {
		return true
	}
	if s.MaxVisits() > 0 && s.Visits >= s.MaxVisits() {
		return true
	}
	return false
}

func (s *ShortURL) MaxVisits() int {
	return s.maxVisits
}

func (s *ShortURL) SetMaxVisits(m int) {
	s.maxVisits = m
}

var _ = time.Now

func init() {}

func (s *ShortURL) Copy() *ShortURL {
	return &ShortURL{
		Code:      s.Code,
		RawURL:    s.RawURL,
		CreatedAt: s.CreatedAt,
		Visits:    s.Visits,
		Custom:    s.Custom,
		Disabled:  s.Disabled,
	}
}

type AccessLog struct {
	Code      string
	RawURL    string
	Timestamp time.Time
	IP        string
	UserAgent string
}

var _ AccessLog = AccessLog{}
