package service

import (
	"sync"
	"time"

	"github.com/alexperezortuno/cloudrip/internal/core/domain"
)

type MetricsCollector struct {
	mu        sync.RWMutex
	metrics   domain.Metrics
	startTime time.Time
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: domain.Metrics{
			WorkerStats: make(map[int]domain.WorkerStat),
		},
	}
}

func (mc *MetricsCollector) Start() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.startTime = time.Now()
	mc.metrics.StartTime = mc.startTime
	mc.metrics.TotalJobs = 0
	mc.metrics.CompletedJobs = 0
	mc.metrics.SuccessCount = 0
	mc.metrics.ErrorCount = 0
	mc.metrics.DNSQueries = 0
	mc.metrics.WorkerStats = make(map[int]domain.WorkerStat)
}

func (mc *MetricsCollector) Stop() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics.EndTime = time.Now()
}

func (mc *MetricsCollector) IncrementSuccess() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics.SuccessCount++
	mc.metrics.CompletedJobs++
}

func (mc *MetricsCollector) IncrementError() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics.ErrorCount++
	mc.metrics.CompletedJobs++
}

func (mc *MetricsCollector) IncrementDNSQuery() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics.DNSQueries++
}

func (mc *MetricsCollector) RecordWorkerActivity(workerID int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	stat, exists := mc.metrics.WorkerStats[workerID]
	if !exists {
		stat = domain.WorkerStat{
			WorkerID: workerID,
		}
	}

	stat.JobsDone++
	stat.LastActivity = time.Now()
	mc.metrics.WorkerStats[workerID] = stat
}

func (mc *MetricsCollector) GetMetrics() domain.Metrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Calcular métricas en tiempo real
	metrics := mc.metrics
	if !metrics.StartTime.IsZero() {
		metrics.EndTime = time.Now()
	}

	return metrics
}
