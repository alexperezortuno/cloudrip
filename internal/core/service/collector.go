package service

import (
	"sync"

	"github.com/alexperezortuno/cloudrip/internal/core/domain"
	"github.com/rs/zerolog"
)

type ResultCollector struct {
	mu      sync.RWMutex
	results map[string][]domain.ResultEntry
	logger  zerolog.Logger
}

func NewResultCollector(logger zerolog.Logger) *ResultCollector {
	return &ResultCollector{
		results: make(map[string][]domain.ResultEntry),
		logger:  logger,
	}
}

func (rc *ResultCollector) Collect(result domain.ResultEntry) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if !rc.isDuplicate(result) {
		rc.results[result.FQDN] = append(rc.results[result.FQDN], result)
		rc.logger.Debug().
			Str("fqdn", result.FQDN).
			Str("ip", result.IP).
			Str("type", result.Type).
			Msg("Resultado colectado")
	}
}

func (rc *ResultCollector) isDuplicate(result domain.ResultEntry) bool {
	existing, exists := rc.results[result.FQDN]
	if !exists {
		return false
	}

	for _, entry := range existing {
		if entry.IP == result.IP && entry.Type == result.Type {
			return true
		}
	}
	return false
}

func (rc *ResultCollector) GetResults() map[string][]domain.ResultEntry {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	// Devolver copia para evitar race conditions
	result := make(map[string][]domain.ResultEntry)
	for k, v := range rc.results {
		entries := make([]domain.ResultEntry, len(v))
		copy(entries, v)
		result[k] = entries
	}

	rc.logger.Debug().Int("total_results", len(result)).Msg("Resultados obtenidos")
	return result
}

func (rc *ResultCollector) Reset() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.results = make(map[string][]domain.ResultEntry)
	rc.logger.Debug().Msg("Resultados reseteados")
}
