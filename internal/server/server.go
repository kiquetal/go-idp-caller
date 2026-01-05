package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-idp-caller/internal/config"
	"github.com/go-idp-caller/internal/jwks"
)

type Server struct {
	config  config.ServerConfig
	manager *jwks.Manager
	logger  *slog.Logger
	server  *http.Server
}

func New(cfg config.ServerConfig, manager *jwks.Manager, logger *slog.Logger) *Server {
	return &Server{
		config:  cfg,
		manager: manager,
		logger:  logger,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/jwks", s.handleGetAllJWKS)
	mux.HandleFunc("/jwks/", s.handleGetIDPJWKS)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/status/", s.handleIDPStatus)

	// Wrap with logging middleware
	handler := s.loggingMiddleware(mux)

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	s.logger.Info("Starting HTTP server", "addr", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.server.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}); err != nil {
		s.logger.Error("Failed to encode health response", "error", err)
	}
}

func (s *Server) handleGetAllJWKS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	all := s.manager.GetAll()
	result := make(map[string]*jwks.JWKS)
	for name, data := range all {
		if data.JWKS != nil {
			result[name] = data.JWKS
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		s.logger.Error("Failed to encode JWKS response", "error", err)
	}
}

func (s *Server) handleGetIDPJWKS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract IDP name from path
	idpName := r.URL.Path[len("/jwks/"):]
	if idpName == "" {
		http.Error(w, "IDP name required", http.StatusBadRequest)
		return
	}

	keySet, exists := s.manager.GetJWKS(idpName)
	if !exists {
		http.Error(w, fmt.Sprintf("IDP '%s' not found", idpName), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(keySet); err != nil {
		s.logger.Error("Failed to encode JWKS response", "error", err, "idp", idpName)
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	all := s.manager.GetAll()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(all); err != nil {
		s.logger.Error("Failed to encode status response", "error", err)
	}
}

func (s *Server) handleIDPStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract IDP name from path
	idpName := r.URL.Path[len("/status/"):]
	if idpName == "" {
		http.Error(w, "IDP name required", http.StatusBadRequest)
		return
	}

	data, exists := s.manager.Get(idpName)
	if !exists {
		http.Error(w, fmt.Sprintf("IDP '%s' not found", idpName), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("Failed to encode status response", "error", err, "idp", idpName)
	}
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		s.logger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration_ms", duration.Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
