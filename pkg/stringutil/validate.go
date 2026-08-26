package stringutil

import (
	"fmt"
	"regexp"
	"strings"
)

// Validator provides string validation utilities.
type Validator struct {
	errors map[string][]string
}

// NewValidator creates a new Validator.
func NewValidator() *Validator {
	return &Validator{
		errors: make(map[string][]string),
	}
}

// AddError adds a validation error for a field.
func (v *Validator) AddError(field, message string) {
	v.errors[field] = append(v.errors[field], message)
}

// HasErrors returns true if there are any validation errors.
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// GetErrors returns all validation errors.
func (v *Validator) GetErrors() map[string][]string {
	return v.errors
}

// ErrorMessage returns a formatted error message.
func (v *Validator) ErrorMessage() string {
	if len(v.errors) == 0 {
		return ""
	}
	var parts []string
	for field, msgs := range v.errors {
		for _, msg := range msgs {
			parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
		}
	}
	return strings.Join(parts, "; ")
}

// ValidateRequired checks if a string is not empty.
func (v *Validator) ValidateRequired(field, value, label string) *Validator {
	if IsEmpty(value) {
		v.AddError(field, fmt.Sprintf("%s is required", label))
	}
	return v
}

// ValidateMinLength checks if a string has at least min characters.
func (v *Validator) ValidateMinLength(field, value string, min int, label string) *Validator {
	if len(value) < min {
		v.AddError(field, fmt.Sprintf("%s must be at least %d characters", label, min))
	}
	return v
}

// ValidateMaxLength checks if a string has at most max characters.
func (v *Validator) ValidateMaxLength(field, value string, max int, label string) *Validator {
	if len(value) > max {
		v.AddError(field, fmt.Sprintf("%s must be at most %d characters", label, max))
	}
	return v
}

// ValidateLength checks if a string has between min and max characters.
func (v *Validator) ValidateLength(field, value string, min, max int, label string) *Validator {
	return v.ValidateMinLength(field, value, min, label).ValidateMaxLength(field, value, max, label)
}

// ValidatePattern checks if a string matches a regex pattern.
func (v *Validator) ValidatePattern(field, value, pattern, label string) *Validator {
	re, err := regexp.Compile(pattern)
	if err != nil {
		v.AddError(field, fmt.Sprintf("invalid pattern for %s", label))
		return v
	}
	if !re.MatchString(value) {
		v.AddError(field, fmt.Sprintf("%s has invalid format", label))
	}
	return v
}

// ValidateAlphaNumeric checks if a string contains only alphanumeric characters.
func (v *Validator) ValidateAlphaNumeric(field, value, label string) *Validator {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, value)
	if !matched {
		v.AddError(field, fmt.Sprintf("%s must contain only alphanumeric characters", label))
	}
	return v
}

// ValidateEmail checks if a string is a valid email format.
func (v *Validator) ValidateEmail(field, value, label string) *Validator {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`, value)
	if !matched {
		v.AddError(field, fmt.Sprintf("%s must be a valid email address", label))
	}
	return v
}

// ValidateURL checks if a string is a valid URL format.
func (v *Validator) ValidateURL(field, value, label string) *Validator {
	matched, _ := regexp.MatchString(`^https?://[a-zA-Z0-9.\-]+(:[0-9]+)?(/.*)?$`, value)
	if !matched {
		v.AddError(field, fmt.Sprintf("%s must be a valid URL", label))
	}
	return v
}

// ValidateInList checks if a string value is in the allowed list.
func (v *Validator) ValidateInList(field, value, label string, allowed []string) *Validator {
	for _, a := range allowed {
		if value == a {
			return v
		}
	}
	v.AddError(field, fmt.Sprintf("%s must be one of: %s", label, strings.Join(allowed, ", ")))
	return v
}

// ValidateNotContains checks if a string does not contain any forbidden substrings.
func (v *Validator) ValidateNotContains(field, value, label string, forbidden []string) *Validator {
	for _, f := range forbidden {
		if strings.Contains(strings.ToLower(value), strings.ToLower(f)) {
			v.AddError(field, fmt.Sprintf("%s contains forbidden content", label))
			break
		}
	}
	return v
}

// ValidateCodeSafety performs basic safety checks on code submissions.
func (v *Validator) ValidateCodeSafety(field, code, language string) *Validator {
	dangerousPatterns := []struct {
		pattern string
		message string
	}{
		{`rm\s+-rf\s+/`, "code contains destructive command (rm -rf /)"},
		{`wget\s+.*&&.*sh`, "code contains suspicious download pattern"},
		{`curl\s+.*&&.*bash`, "code contains suspicious download pattern"},
	}

	for _, p := range dangerousPatterns {
		matched, _ := regexp.MatchString(p.pattern, code)
		if matched {
			v.AddError(field, p.message)
			break
		}
	}

	// Check for very long code
	if len(code) > 100000 {
		v.AddError(field, "code exceeds maximum size limit")
	}

	return v
}

// IsValidIdentifier checks if a string is a valid identifier (letters, digits, underscore, starts with letter).
func IsValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, s)
	return matched
}
