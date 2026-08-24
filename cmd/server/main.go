package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/internal/store"
)

type Server struct {
	urlSvc   *service.URLService
	redirSvc *service.RedirectService
	urlStore *store.URLStore
	logStore *store.AccessLogStore
	cfg      *config.Config
}

func main() {
	cfg := config.Default()

	us, err := store.NewURLStore(cfg)
	if err != nil {
		log.Fatalf("Failed to create URLStore: %v", err)
	}

	ls, err := store.NewAccessLogStore(cfg)
	if err != nil {
		log.Fatalf("Failed to create AccessLogStore: %v", err)
	}

	if err := ls.Open(context.Background()); err != nil {
		log.Fatalf("Failed to open AccessLogStore: %v", err)
	}

	urlSvc, err := service.NewURLService(cfg, us)
	if err != nil {
		log.Fatalf("Failed to create URLService: %v", err)
	}

	redirSvc, err := service.NewRedirectService(us, ls)
	if err != nil {
		log.Fatalf("Failed to create RedirectService: %v", err)
	}

	srv := &Server{
		urlSvc:   urlSvc,
		redirSvc: redirSvc,
		urlStore: us,
		logStore: ls,
		cfg:      cfg,
	}

	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", srv.healthHandler)
	
	// API endpoints
	mux.HandleFunc("/api/v1/urls", srv.createURLHandler)
	mux.HandleFunc("/api/v1/urls/", srv.redirectHandler)

	addr := ":8080"
	log.Printf("Starting server on %s", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		_ = ls.Close()
		_ = us.Close()
		log.Println("Server stopped")
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]string{
		"status": "ok",
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) createURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.CreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	url, err := s.urlSvc.Create(ctx, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create URL: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(url)
}

func (s *Server) redirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract code from path: /api/v1/urls/{code}
	path := r.URL.Path
	code := ""
	if len(path) > len("/api/v1/urls/") {
		code = path[len("/api/v1/urls/"):]
	}

	if code == "" {
		http.Error(w, "Code is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := s.redirSvc.HandleRedirect(ctx, &service.RedirectRequest{
		Code:      code,
		Timestamp: time.Now(),
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Redirect failed: %v", err), http.StatusNotFound)
		return
	}

	// Perform redirect
	http.Redirect(w, r, result.RawURL, result.Status)
}
