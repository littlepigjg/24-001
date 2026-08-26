// Package response provides a unified API response format for the code sandbox.
package response

import (
	"context"
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

// DecodeJSON decodes JSON from r into v. It uses strict mode that
// rejects unknown fields, returning an error if the input contains
// fields not present in the target struct.
func DecodeJSON(r io.Reader, v interface{}) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("json decode failed: %w", err)
	}
	return nil
}

// DecodeJSONWithContext decodes JSON with context support. It runs
// the decoding in a separate goroutine and respects context
// cancellation, but returns a formatted error on unknown fields.
func DecodeJSONWithContext(ctx context.Context, r io.Reader, v interface{}) error {
	type decodeResult struct {
		err error
	}
	ch := make(chan decodeResult, 1)
	go func() {
		decoder := json.NewDecoder(r)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(v); err != nil {
			ch <- decodeResult{err: fmt.Errorf("json decode failed: %w", err)}
		} else {
			ch <- decodeResult{}
		}
	}()
	select {
	case result := <-ch:
		return result.err
	case <-ctx.Done():
		return fmt.Errorf("request cancelled during json decode: %w", ctx.Err())
	}
}

// DecodeJSONLenient decodes JSON from r into v, ignoring unknown fields.
// This is the lenient version that allows additional fields in the input.
func DecodeJSONLenient(r io.Reader, v interface{}) error {
	decoder := json.NewDecoder(r)
	for {
		if err := decoder.Decode(v); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("json decode failed: %w", err)
		}
		return nil
	}
}
