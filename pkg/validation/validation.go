// Package validation provides input validation utilities.
package validation

import (
	"regexp"
	"strings"
)

// Validator provides input validation methods.
type Validator struct {
	errors []string
}

// New creates a new Validator.
func New() *Validator {
	return &Validator{}
}

// Error returns the first validation error message.
func (v *Validator) Error() string {
	if len(v.errors) > 0 {
		return v.errors[0]
	}
	return ""
}

// Errors returns all validation errors.
func (v *Validator) Errors() []string {
	return v.errors
}

// HasErrors checks if there are any validation errors.
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Required checks if a string is not empty.
func (v *Validator) Required(value, field string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.errors = append(v.errors, field+" is required")
	}
	return v
}

// MinLength checks if a string has at least n characters.
func (v *Validator) MinLength(value string, n int, field string) *Validator {
	if len(value) < n {
		v.errors = append(v.errors, field+" must be at least "+itoa(n)+" characters")
	}
	return v
}

// MaxLength checks if a string has at most n characters.
func (v *Validator) MaxLength(value string, n int, field string) *Validator {
	if len(value) > n {
		v.errors = append(v.errors, field+" must be at most "+itoa(n)+" characters")
	}
	return v
}

// InList checks if a value is in a list of allowed values.
func (v *Validator) InList(value, field string, allowed []string) *Validator {
	for _, a := range allowed {
		if value == a {
			return v
		}
	}
	v.errors = append(v.errors, field+" must be one of: "+strings.Join(allowed, ", "))
	return v
}

// MatchesPattern checks if a string matches a regex pattern.
func (v *Validator) MatchesPattern(value, field, pattern string) *Validator {
	re, err := regexp.Compile(pattern)
	if err != nil {
		v.errors = append(v.errors, "invalid validation pattern")
		return v
	}
	if !re.MatchString(value) {
		v.errors = append(v.errors, field+" has an invalid format")
	}
	return v
}

// IsNotSensitive checks that code doesn't contain sensitive patterns.
func (v *Validator) IsNotSensitive(code, field string) *Validator {
	sensitive := []string{
		"os.system", "subprocess.call", "eval(", "exec(",
		"import os", "import subprocess", "__import__",
		"Runtime.exec", "ProcessBuilder",
	}
	for _, s := range sensitive {
		if strings.Contains(strings.ToLower(code), strings.ToLower(s)) {
			v.errors = append(v.errors, field+" contains disallowed keyword: "+s)
			break
		}
	}
	return v
}

// itoa converts an int to string.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
