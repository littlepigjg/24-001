// Package response provides a unified API response format for the code sandbox.
package response

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Response is the standard API response structure.
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success returns a successful response with data.
func Success(w http.ResponseWriter, data interface{}) {
	writeResponse(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessMessage returns a successful response with a custom message and data.
func SuccessMessage(w http.ResponseWriter, message string, data interface{}) {
	writeResponse(w, http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// Created returns a response for resource creation.
func Created(w http.ResponseWriter, data interface{}) {
	writeResponse(w, http.StatusCreated, Response{
		Code:    0,
		Message: "created",
		Data:    data,
	})
}

// Deleted returns a response for resource deletion.
func Deleted(w http.ResponseWriter) {
	writeResponse(w, http.StatusOK, Response{
		Code:    0,
		Message: "deleted",
	})
}

// BadRequest returns a 400 Bad Request response.
func BadRequest(w http.ResponseWriter, message string) {
	writeResponse(w, http.StatusBadRequest, Response{
		Code:    400,
		Message: message,
	})
}

// BadRequestWithError returns a 400 Bad Request response with error details.
func BadRequestWithError(w http.ResponseWriter, err error) {
	writeResponse(w, http.StatusBadRequest, Response{
		Code:    400,
		Message: err.Error(),
	})
}

// Unauthorized returns a 401 Unauthorized response.
func Unauthorized(w http.ResponseWriter, message string) {
	writeResponse(w, http.StatusUnauthorized, Response{
		Code:    401,
		Message: message,
	})
}

// Forbidden returns a 403 Forbidden response.
func Forbidden(w http.ResponseWriter, message string) {
	writeResponse(w, http.StatusForbidden, Response{
		Code:    403,
		Message: message,
	})
}

// NotFound returns a 404 Not Found response.
func NotFound(w http.ResponseWriter, message string) {
	writeResponse(w, http.StatusNotFound, Response{
		Code:    404,
		Message: message,
	})
}

// InternalError returns a 500 Internal Server Error response.
func InternalError(w http.ResponseWriter, message string) {
	writeResponse(w, http.StatusInternalServerError, Response{
		Code:    500,
		Message: message,
	})
}

// InternalErrorWithError returns a 500 Internal Server Error response with error details.
func InternalErrorWithError(w http.ResponseWriter, err error) {
	writeResponse(w, http.StatusInternalServerError, Response{
		Code:    500,
		Message: err.Error(),
	})
}

// TooManyRequests returns a 429 Too Many Requests response.
func TooManyRequests(w http.ResponseWriter, message string) {
	writeResponse(w, http.StatusTooManyRequests, Response{
		Code:    429,
		Message: message,
	})
}

// ServiceUnavailable returns a 503 Service Unavailable response.
func ServiceUnavailable(w http.ResponseWriter, message string) {
	writeResponse(w, http.StatusServiceUnavailable, Response{
		Code:    503,
		Message: message,
	})
}

// writeResponse writes the response as JSON with the given status code.
func writeResponse(w http.ResponseWriter, statusCode int, resp Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, `{"code":500,"message":"failed to encode response"}`, http.StatusInternalServerError)
	}
}

// DecodeJSONBody reads and decodes a JSON request body into the given value.
// It fully drains and closes the request body so the underlying keep-alive
// connection can be reused by the server.
func DecodeJSONBody(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}
	if len(body) == 0 {
		return fmt.Errorf("request body is empty")
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}
	return nil
}

// CloseRequestBody drains and closes the request body so that the underlying
// keep-alive connection can be reused. It must be called on every request that
// carries a body, including error paths where the body was only partially or
// never read. It is safe to call when r.Body is nil.
func CloseRequestBody(r *http.Request) {
	if r.Body == nil {
		return
	}
	// Drain any remaining bytes so the connection is in a reusable state.
	_, _ = io.Copy(io.Discard, r.Body)
	_ = r.Body.Close()
}

// ValidateRequestBody checks the request body for basic validity before parsing.
// It fully drains and closes the request body so the underlying keep-alive
// connection can be reused by the server.
func ValidateRequestBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body for validation: %w", err)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("request body is empty")
	}
	return body, nil
}
