package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexperezortuno/cloudrip/internal/core/domain"
	"github.com/alexperezortuno/cloudrip/internal/core/service"
	"github.com/alexperezortuno/cloudrip/internal/infrastructure/cloudflare"
	"github.com/alexperezortuno/cloudrip/internal/infrastructure/config"
	"github.com/alexperezortuno/cloudrip/internal/infrastructure/dns"
	"github.com/alexperezortuno/cloudrip/internal/infrastructure/file"
	"github.com/alexperezortuno/cloudrip/internal/infrastructure/logging"
	"github.com/alexperezortuno/cloudrip/internal/infrastructure/progress"
	"github.com/alexperezortuno/cloudrip/internal/interfaces/cli"
	"github.com/alexperezortuno/cloudrip/internal/interfaces/http"
	"github.com/rs/zerolog"
)

func main() {
	// Configurar logging
	logger := logging.NewLogger()

	// Parsear flags de CLI
	cliConfig, err := cli.ParseFlags()
	if err != nil {
		logger.Fatal().Err(err).Msg("Error parseando flags")
	}

	// Health check rápido
	if cliConfig.HealthCheck {
		fmt.Println("✅ Health check: OK")
		os.Exit(0)
	}

	// Crear configuración por defecto
	if cliConfig.CreateConfig != "" {
		configManager := config.NewConfigManager()
		if err := configManager.CreateDefaultConfig(cliConfig.CreateConfig); err != nil {
			logger.Fatal().Err(err).Msg("Error creando archivo de configuración")
		}
		fmt.Printf("✅ Archivo de configuración creado: %s\n", cliConfig.CreateConfig)
		os.Exit(0)
	}

	// Cargar configuración
	configManager := config.NewConfigManager()
	var cfg *domain.ScannerConfig

	if cliConfig.ConfigFile != "" {
		cfg, err = configManager.LoadFromFile(cliConfig.ConfigFile)
		if err != nil {
			logger.Warn().Err(err).Msg("Error cargando configuración desde archivo, usando CLI flags")
			cfg = &cliConfig.ScannerConfig
		} else {
			logger.Info().Str("config_file", cliConfig.ConfigFile).Msg("Configuración cargada desde archivo")
		}
	} else {
		cfg = &cliConfig.ScannerConfig
	}

	// Validar configuración
	if err := configManager.Validate(cfg); err != nil {
		logger.Fatal().Err(err).Msg("Configuración inválida")
	}

	// Mostrar métricas y salir
	if cliConfig.ShowMetrics {
		showDefaultMetrics()
		os.Exit(0)
	}

	// Configurar contexto con cancelación
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Inicializar dependencias
	dnsResolver := dns.NewResolver(logger)
	cloudflareService := cloudflare.NewService(logger)
	fileRepo := file.NewRepository(logger)
	progressReporter := progress.NewReporter()
	metricsCollector := service.NewMetricsCollector()
	healthChecker := service.NewHealthChecker(metricsCollector)

	// Crear servicio de escaneo
	scanner := service.NewScanner(
		dnsResolver,
		cloudflareService,
		fileRepo,
		progressReporter,
		metricsCollector,
		healthChecker,
		logger,
	)

	// Iniciar servidor HTTP en segundo plano
	//go startHTTPServer(scanner, logger)

	// Ejecutar escaneo
	//startTime := time.Now()
	result, err := scanner.Scan(ctx, *cfg)
	if err != nil {
		logger.Error().Err(err).Msg("Error durante el escaneo")
		os.Exit(1)
	}

	logger.Info().
		Int("found", result.TotalFound).
		Dur("duration", result.Duration).
		Msg("Escaneo completado exitosamente")
}

func startHTTPServer(scanner *service.Scanner, logger zerolog.Logger) {
	server := http.NewServer(scanner, logger, "8080")
	if err := server.Start(); err != nil {
		logger.Error().Err(err).Msg("Error iniciando servidor HTTP")
	}
}

func showDefaultMetrics() {
	fmt.Println("📊 Métricas por defecto:")
	fmt.Println("------------------------")
	fmt.Println("Threads: 10")
	fmt.Println("Retries: 2")
	fmt.Println("Timeout: 5s")
	fmt.Println("Backoff: 500ms")
	fmt.Println("Formato salida: text")
	fmt.Println("Wordlist: dom.txt")
	fmt.Println("------------------------")
}
