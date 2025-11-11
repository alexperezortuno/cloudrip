package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/alexperezortuno/cloudrip/internal/core/domain"
	"github.com/alexperezortuno/cloudrip/internal/core/ports"
	"github.com/rs/zerolog"
)

type Server struct {
	scannerService ports.ScannerService
	logger         zerolog.Logger
	port           string
}

func NewServer(scannerService ports.ScannerService, logger zerolog.Logger, port string) *Server {
	return &Server{
		scannerService: scannerService,
		logger:         logger,
		port:           port,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", s.healthHandler)

	// Metrics endpoint
	mux.HandleFunc("/metrics", s.metricsHandler)

	// Scan endpoint (opcional)
	mux.HandleFunc("/scan", s.scanHandler)

	server := &http.Server{
		Addr:         ":" + s.port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	s.logger.Info().Str("port", s.port).Msg("Servidor HTTP iniciado")
	return server.ListenAndServe()
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := s.scannerService.HealthCheck()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		s.logger.Error().Err(err).Msg("Error encoding health response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := s.scannerService.GetMetrics()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		s.logger.Error().Err(err).Msg("Error encoding metrics response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (s *Server) scanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Esta sería una implementación básica - en producción necesitarías más validación
	var config domain.ScannerConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Ejecutar escaneo (en un goroutine para no bloquear)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		_, err := s.scannerService.Scan(ctx, config)
		if err != nil {
			s.logger.Error().Err(err).Msg("Error en escaneo HTTP")
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "scan_started",
		"domain": config.Domain,
	})
}
