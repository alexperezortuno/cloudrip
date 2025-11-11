package service

import (
	"runtime"
	"time"

	"github.com/alexperezortuno/cloudrip/internal/core/domain"
)

type HealthChecker struct {
	startTime time.Time
	metrics   *MetricsCollector
}

func NewHealthChecker(metrics *MetricsCollector) *HealthChecker {
	return &HealthChecker{
		startTime: time.Now(),
		metrics:   metrics,
	}
}

func (hc *HealthChecker) Check() domain.HealthStatus {
	uptime := time.Since(hc.startTime)
	metrics := hc.metrics.GetMetrics()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Calcular uso de memoria en MB
	memUsage := float64(m.Alloc) / 1024 / 1024

	status := "healthy"
	if memUsage > 1000 { // Más de 1GB de uso
		status = "warning"
	}

	return domain.HealthStatus{
		Status:    status,
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Uptime:    uptime.String(),
		Metrics:   metrics,
	}
}
