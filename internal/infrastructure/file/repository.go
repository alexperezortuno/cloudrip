package file

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/alexperezortuno/cloudrip/internal/core/domain"
	"github.com/rs/zerolog"
)

type Repository struct {
	logger zerolog.Logger
}

func NewRepository(logger zerolog.Logger) *Repository {
	return &Repository{
		logger: logger,
	}
}

func (r *Repository) LoadWordlist(path string) ([]string, error) {
	r.logger.Debug().Str("path", path).Msg("Cargando wordlist")

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("abriendo archivo: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			r.logger.Warn().Err(err).Msg("Error cerrando archivo")
		}
	}(file)

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("leyendo archivo: %w", err)
	}

	r.logger.Debug().Int("lines", len(lines)).Msg("Wordlist cargada")
	return lines, nil
}

func (r *Repository) SaveResults(results map[string][]domain.ResultEntry, config domain.ScannerConfig) error {
	if config.Output == "" {
		return nil
	}

	r.logger.Debug().
		Str("path", config.Output).
		Str("format", config.OutputFmt).
		Msg("Guardando resultados")

	switch strings.ToLower(config.OutputFmt) {
	case "json":
		return r.saveJSON(results, config.Output)
	case "text":
		return r.saveText(results, config.Output)
	default:
		return fmt.Errorf("formato no soportado: %s", config.OutputFmt)
	}
}

func (r *Repository) saveJSON(results map[string][]domain.ResultEntry, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creando archivo: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			r.logger.Warn().Err(err).Msg("Error cerrando archivo")
		}
	}(file)

	// Convertir a slice plana y ordenar
	var flatResults []domain.ResultEntry
	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		entries := results[key]
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Type == entries[j].Type {
				return entries[i].IP < entries[j].IP
			}
			return entries[i].Type < entries[j].Type
		})
		flatResults = append(flatResults, entries...)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(flatResults); err != nil {
		return fmt.Errorf("escribiendo JSON: %w", err)
	}

	r.logger.Debug().Str("path", path).Int("results", len(flatResults)).Msg("Resultados guardados en JSON")
	return nil
}

func (r *Repository) saveText(results map[string][]domain.ResultEntry, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creando archivo: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			r.logger.Warn().Err(err).Msg("Error cerrando archivo")
		}
	}(file)

	writer := bufio.NewWriter(file)
	defer func(writer *bufio.Writer) {
		err := writer.Flush()
		if err != nil {
			r.logger.Warn().Err(err).Msg("Error guardando resultados")
		}
	}(writer)

	// Ordenar resultados
	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		entries := results[key]
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Type == entries[j].Type {
				return entries[i].IP < entries[j].IP
			}
			return entries[i].Type < entries[j].Type
		})
		for _, entry := range entries {
			r.logger.Info().Str("fqdn", entry.FQDN).Str("ip", entry.IP).Str("type", entry.Type).Msg("Result")
		}
	}

	r.logger.Debug().Str("path", path).Int("entries", len(results)).Msg("Results saved to text file")
	return nil
}

func (r *Repository) LoadConfig(path string) (*domain.ScannerConfig, error) {
	// Implementación simple - en una aplicación real usarías viper o similar
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("leyendo archivo de configuración: %w", err)
	}

	var config domain.ScannerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parseando configuración: %w", err)
	}

	return &config, nil
}

func (r *Repository) SaveConfig(config *domain.ScannerConfig, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("serializando configuración: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("guardando configuración: %w", err)
	}

	return nil
}
