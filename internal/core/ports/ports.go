package ports

import (
	"context"

	"github.com/alexperezortuno/cloudrip/internal/core/domain"
)

// ScannerService define el puerto de entrada para el escaneo
type ScannerService interface {
	Scan(ctx context.Context, config domain.ScannerConfig) (*domain.ScanResult, error)
	GetMetrics() domain.Metrics
	HealthCheck() domain.HealthStatus
}

// CloudflareService define las operaciones con Cloudflare
type CloudflareService interface {
	GetRanges(ctx context.Context, noFetch bool) (domain.CFRanges, error)
	IsCloudflareIP(ip string, ranges domain.CFRanges) bool
}

// DNSResolver define las operaciones de resolución DNS
type DNSResolver interface {
	LookupIP(ctx context.Context, fqdn string) ([]string, error)
	LookupCNAME(ctx context.Context, fqdn string) (string, error)
}

// FileRepository maneja operaciones de archivo
type FileRepository interface {
	LoadWordlist(path string) ([]string, error)
	SaveResults(results map[string][]domain.ResultEntry, config domain.ScannerConfig) error
	LoadConfig(path string) (*domain.ScannerConfig, error)
	SaveConfig(config *domain.ScannerConfig, path string) error
}

// ResultHandler maneja los resultados del escaneo
type ResultHandler interface {
	Handle(result domain.ResultEntry)
	GetResults() map[string][]domain.ResultEntry
	Reset()
}

// ProgressReporter reporta el progreso del escaneo
type ProgressReporter interface {
	Start(total int)
	Increment()
	Stop()
	GetProgress() float64
}

// MetricsCollector recolecta métricas de performance
type MetricsCollector interface {
	Start()
	Stop()
	IncrementSuccess()
	IncrementError()
	IncrementDNSQuery()
	RecordWorkerActivity(workerID int)
	GetMetrics() domain.Metrics
}

// ConfigManager maneja la configuración
type ConfigManager interface {
	LoadFromFile(path string) (*domain.ScannerConfig, error)
	LoadFromEnv() (*domain.ScannerConfig, error)
	SaveToFile(config *domain.ScannerConfig, path string) error
	Validate(config *domain.ScannerConfig) error
	CreateDefaultConfig(path string) error
}

// HealthChecker verifica la salud del sistema
type HealthChecker interface {
	Check() domain.HealthStatus
}
