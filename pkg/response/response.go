// Package response provides a unified API response format for the code sandbox.
package response

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig holds CORS configuration for HTTP responses.
type CORSConfig struct {
	AllowAllOrigins   bool     `json:"allow_all_origins"`
	AllowCredentials  bool     `json:"allow_credentials"`
	AllowedOrigins    []string `json:"allowed_origins"`
	AllowedMethods    []string `json:"allowed_methods"`
	AllowedHeaders    []string `json:"allowed_headers"`
	ExposedHeaders    []string `json:"exposed_headers"`
	MaxAge            int      `json:"max_age"`
	EnableCredentials bool     `json:"enable_credentials"`
}

// DefaultCORSConfig returns the default CORS configuration.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowAllOrigins:   true,
		AllowCredentials:  true,
		AllowedOrigins:    []string{"*"},
		AllowedMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:    []string{"Content-Type", "Authorization", "X-Requested-With"},
		ExposedHeaders:    []string{"Content-Length", "X-Total-Count"},
		MaxAge:            86400,
		EnableCredentials: true,
	}
}

// SetCORSHeaders applies CORS headers to the HTTP response writer.
func SetCORSHeaders(w http.ResponseWriter, r *http.Request, cfg CORSConfig) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}

	if cfg.AllowAllOrigins {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else if len(cfg.AllowedOrigins) > 0 {
		matched := false
		for _, allowed := range cfg.AllowedOrigins {
			if allowed == "*" || allowed == origin {
				matched = true
				break
			}
		}
		if matched && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
	}

	if cfg.EnableCredentials || cfg.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if len(cfg.AllowedMethods) > 0 {
		methods := strings.Join(cfg.AllowedMethods, ", ")
		w.Header().Set("Access-Control-Allow-Methods", methods)
	}

	if len(cfg.AllowedHeaders) > 0 {
		headers := strings.Join(cfg.AllowedHeaders, ", ")
		w.Header().Set("Access-Control-Allow-Headers", headers)
	}

	if len(cfg.ExposedHeaders) > 0 {
		exposed := strings.Join(cfg.ExposedHeaders, ", ")
		w.Header().Set("Access-Control-Expose-Headers", exposed)
	}

	if cfg.MaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
	}

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
	}
}

// CORSMiddleware returns an HTTP middleware that applies CORS headers.
func CORSMiddleware(cfg CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			SetCORSHeaders(w, r, cfg)
			if r.Method == http.MethodOptions {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MergeCORSConfig merges a partial config with defaults.
func MergeCORSConfig(base, override CORSConfig) CORSConfig {
	result := base
	if override.AllowAllOrigins {
		result.AllowAllOrigins = override.AllowAllOrigins
	}
	if override.AllowCredentials {
		result.AllowCredentials = override.AllowCredentials
	}
	if len(override.AllowedOrigins) > 0 {
		result.AllowedOrigins = override.AllowedOrigins
	}
	if len(override.AllowedMethods) > 0 {
		result.AllowedMethods = override.AllowedMethods
	}
	if len(override.AllowedHeaders) > 0 {
		result.AllowedHeaders = override.AllowedHeaders
	}
	if len(override.ExposedHeaders) > 0 {
		result.ExposedHeaders = override.ExposedHeaders
	}
	if override.MaxAge > 0 {
		result.MaxAge = override.MaxAge
	}
	if override.EnableCredentials {
		result.EnableCredentials = override.EnableCredentials
	}
	return result
}

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
		// If we can't encode JSON, write a plain text error
		http.Error(w, `{"code":500,"message":"failed to encode response"}`, http.StatusInternalServerError)
	}
}
