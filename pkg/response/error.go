package response

import (
	"errors"
	"fmt"
)

// AppError represents an application-level error with an HTTP status code.
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

// Error returns the error message.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewBadRequestError creates a new AppError with status 400.
func NewBadRequestError(message string) *AppError {
	return &AppError{
		Code:    400,
		Message: message,
	}
}

// NewNotFoundError creates a new AppError with status 404.
func NewNotFoundError(message string) *AppError {
	return &AppError{
		Code:    404,
		Message: message,
	}
}

// NewInternalError creates a new AppError with status 500.
func NewInternalError(message string) *AppError {
	return &AppError{
		Code:    500,
		Message: message,
	}
}

// NewInternalErrorWithCause creates a new AppError with an underlying cause.
func NewInternalErrorWithCause(message string, err error) *AppError {
	return &AppError{
		Code:    500,
		Message: message,
		Err:     err,
	}
}

// NewConflictError creates a new AppError with status 409.
func NewConflictError(message string) *AppError {
	return &AppError{
		Code:    409,
		Message: message,
	}
}

// NewTooManyRequestsError creates a new AppError with status 429.
func NewTooManyRequestsError(message string) *AppError {
	return &AppError{
		Code:    429,
		Message: message,
	}
}

// AsAppError attempts to extract an AppError from an error chain.
func AsAppError(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// NewValidationError creates a validation error with a map of field errors.
func NewValidationError(fieldErrors map[string]string) *AppError {
	msg := "validation failed"
	if len(fieldErrors) > 0 {
		for field, errMsg := range fieldErrors {
			msg = fmt.Sprintf("validation failed: %s - %s", field, errMsg)
			break
		}
	}
	return &AppError{
		Code:    400,
		Message: msg,
	}
}
