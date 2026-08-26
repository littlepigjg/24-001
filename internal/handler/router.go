package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/response"
)

var corsConfig response.CORSConfig

// SetCORSConfig sets the package-level CORS configuration.
func SetCORSConfig(cfg response.CORSConfig) {
	corsConfig = cfg
}

// GetCORSConfig returns the current CORS configuration.
func GetCORSConfig() response.CORSConfig {
	return corsConfig
}

// RouterConfig holds the configuration for setting up routes.
type RouterConfig struct {
	ExecutionHandler *ExecutionHandler
	HistoryHandler   *HistoryHandler
	TemplateHandler  *TemplateHandler
	LanguageHandler  *LanguageHandler
	HealthHandler    *HealthHandler
	ValidateHandler  *ValidateHandler
	StaticDir        string // path to static files directory (optional)
}

// SetupRoutes configures all HTTP routes and returns an http.Handler.
func SetupRoutes(config RouterConfig) http.Handler {
	mux := http.NewServeMux()

	// Health endpoints (no API prefix)
	mux.HandleFunc("/health", config.HealthHandler.Health)
	mux.HandleFunc("/ready", config.HealthHandler.Ready)

	// Validate endpoint
	if config.ValidateHandler != nil {
		mux.HandleFunc("/api/validate", config.ValidateHandler.Validate)
	}

	// Static file serving for frontend
	if config.StaticDir != "" {
		fs := http.FileServer(http.Dir(config.StaticDir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if path == "/" || path == "/index.html" {
				// Serve index.html
				http.ServeFile(w, r, config.StaticDir+"/index.html")
				return
			}
			fs.ServeHTTP(w, r)
		})
	}

	// Execution API endpoints
	mux.HandleFunc("/api/execute", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			config.ExecutionHandler.Execute(w, r)
		case http.MethodGet:
			config.ExecutionHandler.List(w, r)
		default:
			response.BadRequest(w, "Method not allowed")
		}
	})

	mux.HandleFunc("/api/execute/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.BadRequest(w, "Method not allowed")
			return
		}
		config.ExecutionHandler.GetStats(w, r)
	})

	mux.HandleFunc("/api/execute/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			response.BadRequest(w, "Method not allowed")
			return
		}
		config.ExecutionHandler.BatchExecute(w, r)
	})

	// Execution by ID - uses path matching
	mux.HandleFunc("/api/execute/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/execute/")
		if id == "" {
			response.BadRequest(w, "execution ID is required")
			return
		}
		config.ExecutionHandler.GetResult(w, r)
	})

	// History API endpoints
	mux.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			config.HistoryHandler.List(w, r)
		case http.MethodDelete:
			config.HistoryHandler.Clear(w, r)
		default:
			response.BadRequest(w, "Method not allowed")
		}
	})

	mux.HandleFunc("/api/history/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.BadRequest(w, "Method not allowed")
			return
		}
		config.HistoryHandler.GetStats(w, r)
	})

	mux.HandleFunc("/api/history/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.BadRequest(w, "Method not allowed")
			return
		}
		config.HistoryHandler.Search(w, r)
	})

	mux.HandleFunc("/api/history/export", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.BadRequest(w, "Method not allowed")
			return
		}
		config.HistoryHandler.Export(w, r)
	})

	mux.HandleFunc("/api/history/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/history/")
		if id == "" {
			response.BadRequest(w, "history ID is required")
			return
		}
		if r.Method == http.MethodDelete {
			config.HistoryHandler.Delete(w, r)
		} else {
			config.HistoryHandler.Get(w, r)
		}
	})

	// Template API endpoints
	mux.HandleFunc("/api/templates", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			config.TemplateHandler.Create(w, r)
		case http.MethodGet:
			config.TemplateHandler.List(w, r)
		default:
			response.BadRequest(w, "Method not allowed")
		}
	})

	mux.HandleFunc("/api/templates/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.BadRequest(w, "Method not allowed")
			return
		}
		config.TemplateHandler.Search(w, r)
	})

	mux.HandleFunc("/api/templates/predefined", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.BadRequest(w, "Method not allowed")
			return
		}
		config.TemplateHandler.GetPredefined(w, r)
	})

	mux.HandleFunc("/api/templates/language/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.BadRequest(w, "Method not allowed")
			return
		}
		config.TemplateHandler.ListByLanguage(w, r)
	})

	mux.HandleFunc("/api/templates/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/templates/")
		if id == "" {
			response.BadRequest(w, "template ID is required")
			return
		}
		switch r.Method {
		case http.MethodGet:
			config.TemplateHandler.Get(w, r)
		case http.MethodPut:
			config.TemplateHandler.Update(w, r)
		case http.MethodDelete:
			config.TemplateHandler.Delete(w, r)
		default:
			response.BadRequest(w, "Method not allowed")
		}
	})

	// Language API endpoints
	mux.HandleFunc("/api/languages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.BadRequest(w, "Method not allowed")
			return
		}
		config.LanguageHandler.List(w, r)
	})

	mux.HandleFunc("/api/languages/available", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.BadRequest(w, "Method not allowed")
			return
		}
		config.LanguageHandler.GetAvailable(w, r)
	})

	mux.HandleFunc("/api/languages/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.BadRequest(w, "Method not allowed")
			return
		}
		config.LanguageHandler.Get(w, r)
	})

	// System status and version
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.BadRequest(w, "Method not allowed")
			return
		}
		config.HealthHandler.Status(w, r)
	})

	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.BadRequest(w, "Method not allowed")
			return
		}
		config.HealthHandler.Version(w, r)
	})

	// Apply middleware
	var handler http.Handler = mux
	handler = withCORS(handler)
	handler = withLogging(handler)
	handler = withRecovery(handler)

	return handler
}

// withCORS adds CORS headers to responses.
func withCORS(next http.Handler) http.Handler {
	cfg := corsConfig
	if len(cfg.AllowedOrigins) == 0 {
		cfg = response.DefaultCORSConfig()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.SetCORSHeaders(w, r, cfg)
		if r.Method != http.MethodOptions {
			next.ServeHTTP(w, r)
		}
	})
}

// WithCORS returns a new handler with the given CORS configuration applied.
func WithCORS(next http.Handler, cfg response.CORSConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.SetCORSHeaders(w, r, cfg)
		if r.Method == http.MethodOptions {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withLogging logs HTTP requests.
func withLogging(next http.Handler) http.Handler {
	log := logger.GetGlobal()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Infof("%s %s %d %s",
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration.String(),
		)
	})
}

// withRecovery recovers from panics and returns a 500 error.
func withRecovery(next http.Handler) http.Handler {
	log := logger.GetGlobal()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("Panic recovered: %v", err)
				response.InternalError(w, fmt.Sprintf("internal server error: %v", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Ensure JSON is used
var _ = json.Marshal
