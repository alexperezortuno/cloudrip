package config

import (
	"fmt"
	"os"
	"time"

	"github.com/alexperezortuno/cloudrip/internal/core/domain"
	"gopkg.in/yaml.v3"
)

type ConfigManager struct {
	defaultConfig domain.ScannerConfig
}

func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		defaultConfig: domain.ScannerConfig{
			Threads:     10,
			Retries:     2,
			Backoff:     500 * time.Millisecond,
			Timeout:     5 * time.Second,
			Delay:       0,
			FollowCNAME: false,
			IncludeCF:   false,
			NoFetchCF:   false,
			OutputFmt:   "text",
			Wordlist:    "dom.txt",
		},
	}
}

func (cm *ConfigManager) LoadFromFile(path string) (*domain.ScannerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error leyendo archivo de configuración: %w", err)
	}

	var config domain.ScannerConfig

	// Intentar parsear como YAML primero
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parseando configuración YAML: %w", err)
	}

	// Aplicar valores por defecto
	config = cm.applyDefaults(config)

	if err := cm.Validate(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (cm *ConfigManager) SaveToFile(config *domain.ScannerConfig, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error serializando configuración: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("error guardando configuración: %w", err)
	}

	return nil
}

func (cm *ConfigManager) Validate(config *domain.ScannerConfig) error {
	if config.Domain == "" {
		return fmt.Errorf("el dominio es requerido")
	}
	if config.Threads < 1 {
		return fmt.Errorf("el número de threads debe ser mayor a 0")
	}
	if config.Retries < 0 {
		return fmt.Errorf("los reintentos no pueden ser negativos")
	}
	if config.Timeout < 0 {
		return fmt.Errorf("el timeout no puede ser negativo")
	}
	if config.Backoff < 0 {
		return fmt.Errorf("el backoff no puede ser negativo")
	}
	if config.Delay < 0 {
		return fmt.Errorf("el delay no puede ser negativo")
	}

	// Validar formatos de salida
	validFormats := map[string]bool{"text": true, "json": true}
	if !validFormats[config.OutputFmt] {
		return fmt.Errorf("formato de salida inválido: %s. Debe ser 'text' o 'json'", config.OutputFmt)
	}

	// Validar que el wordlist existe si se especificó
	if config.Wordlist != "" {
		if _, err := os.Stat(config.Wordlist); os.IsNotExist(err) {
			return fmt.Errorf("el archivo wordlist no existe: %s", config.Wordlist)
		}
	}

	return nil
}

func (cm *ConfigManager) applyDefaults(config domain.ScannerConfig) domain.ScannerConfig {
	if config.Threads == 0 {
		config.Threads = cm.defaultConfig.Threads
	}
	if config.Retries == 0 {
		config.Retries = cm.defaultConfig.Retries
	}
	if config.Backoff == 0 {
		config.Backoff = cm.defaultConfig.Backoff
	}
	if config.Timeout == 0 {
		config.Timeout = cm.defaultConfig.Timeout
	}
	if config.OutputFmt == "" {
		config.OutputFmt = cm.defaultConfig.OutputFmt
	}
	if config.Wordlist == "" {
		config.Wordlist = cm.defaultConfig.Wordlist
	}

	return config
}

// CreateDefaultConfig crea un archivo de configuración por defecto
func (cm *ConfigManager) CreateDefaultConfig(path string) error {
	defaultConfig := &domain.ScannerConfig{
		Domain:      "example.com",
		Wordlist:    "dom.txt",
		Threads:     10,
		Retries:     2,
		Backoff:     500 * time.Millisecond,
		Timeout:     5 * time.Second,
		Delay:       0,
		FollowCNAME: false,
		IncludeCF:   false,
		NoFetchCF:   false,
		Output:      "results.txt",
		OutputFmt:   "text",
	}

	return cm.SaveToFile(defaultConfig, path)
}
