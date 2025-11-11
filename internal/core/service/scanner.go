package service

import (
	"context"
	"fmt"
	"time"

	"github.com/alexperezortuno/cloudrip/internal/core/domain"
	"github.com/alexperezortuno/cloudrip/internal/core/ports"
	"github.com/rs/zerolog"
)

type Scanner struct {
	dnsResolver       ports.DNSResolver
	cloudflareService ports.CloudflareService
	fileRepo          ports.FileRepository
	progressReporter  ports.ProgressReporter
	metricsCollector  ports.MetricsCollector
	healthChecker     ports.HealthChecker
	logger            zerolog.Logger
	startTime         time.Time
}

func NewScanner(
	dnsResolver ports.DNSResolver,
	cloudflareService ports.CloudflareService,
	fileRepo ports.FileRepository,
	progressReporter ports.ProgressReporter,
	metricsCollector ports.MetricsCollector,
	healthChecker ports.HealthChecker,
	logger zerolog.Logger,
) *Scanner {
	return &Scanner{
		dnsResolver:       dnsResolver,
		cloudflareService: cloudflareService,
		fileRepo:          fileRepo,
		progressReporter:  progressReporter,
		metricsCollector:  metricsCollector,
		healthChecker:     healthChecker,
		logger:            logger,
		startTime:         time.Now(),
	}
}

func (s *Scanner) Scan(ctx context.Context, config domain.ScannerConfig) (*domain.ScanResult, error) {
	s.logger.Info().
		Str("domain", config.Domain).
		Int("threads", config.Threads).
		Str("wordlist", config.Wordlist).
		Msg("Iniciando escaneo DNS")

	// Iniciar métricas
	//s.metricsCollector.Start()
	//defer s.metricsCollector.Stop()

	startTime := time.Now()

	// Cargar wordlist
	subdomains, err := s.fileRepo.LoadWordlist(config.Wordlist)
	if err != nil {
		s.logger.Error().Err(err).Str("wordlist", config.Wordlist).Msg("Error cargando wordlist")
		return nil, fmt.Errorf("error cargando wordlist: %w", err)
	}

	s.logger.Info().Int("subdomains", len(subdomains)).Msg("Wordlist cargada")

	// Obtener rangos de Cloudflare
	ranges, err := s.cloudflareService.GetRanges(ctx, config.NoFetchCF)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Error obteniendo rangos Cloudflare, usando defaults")
		// El servicio Cloudflare ya maneja los defaults internamente
	}

	// Configurar progreso
	if s.progressReporter != nil {
		s.progressReporter.Start(len(subdomains))
		defer s.progressReporter.Stop()
	}

	// Ejecutar workers
	results := s.startWorkers(ctx, config, subdomains, ranges)

	duration := time.Since(startTime)

	// Guardar resultados si es necesario
	if config.Output != "" {
		if err := s.fileRepo.SaveResults(results, config); err != nil {
			s.logger.Error().Err(err).Str("output", config.Output).Msg("Error guardando resultados")
			return nil, fmt.Errorf("error guardando resultados: %w", err)
		}
		s.logger.Info().Str("output", config.Output).Msg("Resultados guardados")
	}

	scanResult := &domain.ScanResult{
		TotalFound: len(results),
		Duration:   duration,
		Results:    results,
	}

	s.logger.Info().
		Int("found", scanResult.TotalFound).
		Dur("duration", scanResult.Duration).
		Msg("Escaneo completado")

	return scanResult, nil
}

func (s *Scanner) GetMetrics() domain.Metrics {
	return s.metricsCollector.GetMetrics()
}

func (s *Scanner) HealthCheck() domain.HealthStatus {
	uptime := time.Since(s.startTime)
	metrics := s.metricsCollector.GetMetrics()

	return domain.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Uptime:    uptime.String(),
		Metrics:   metrics,
	}
}
