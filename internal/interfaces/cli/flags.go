package cli

import (
	"flag"
	"fmt"
	"time"

	"github.com/alexperezortuno/cloudrip/internal/core/domain"
)

type CLIConfig struct {
	ScannerConfig domain.ScannerConfig
	ConfigFile    string
	HealthCheck   bool
	ShowMetrics   bool
	CreateConfig  string
}

func ParseFlags() (*CLIConfig, error) {
	var cliConfig CLIConfig

	// Flags principales
	flag.StringVar(&cliConfig.ScannerConfig.Domain, "d", "", "Dominio objetivo (ej: example.com) [requerido]")
	flag.StringVar(&cliConfig.ScannerConfig.Wordlist, "w", "dom.txt", "Ruta al wordlist (uno por línea)")
	flag.IntVar(&cliConfig.ScannerConfig.Threads, "t", 10, "Número de workers concurrentes")
	flag.StringVar(&cliConfig.ScannerConfig.Output, "o", "", "Ruta de salida (opcional)")
	flag.StringVar(&cliConfig.ScannerConfig.OutputFmt, "output-format", "text", "Formato de salida: text|json")
	flag.IntVar(&cliConfig.ScannerConfig.Retries, "retries", 2, "Reintentos por resolución")
	flag.DurationVar(&cliConfig.ScannerConfig.Backoff, "backoff", 500*time.Millisecond, "Backoff base entre reintentos")
	flag.DurationVar(&cliConfig.ScannerConfig.Timeout, "timeout", 5*time.Second, "Timeout por consulta DNS")
	flag.DurationVar(&cliConfig.ScannerConfig.Delay, "delay", 0, "Delay por job (throttling)")
	flag.BoolVar(&cliConfig.ScannerConfig.FollowCNAME, "follow-cname", false, "Seguir un nivel de CNAME")
	flag.BoolVar(&cliConfig.ScannerConfig.IncludeCF, "include-cf", false, "Incluir IPs pertenecientes a Cloudflare en resultados")
	flag.BoolVar(&cliConfig.ScannerConfig.NoFetchCF, "no-fetch-cf", false, "No intentar actualizar CIDRs de Cloudflare desde Internet")

	// Flags adicionales
	flag.StringVar(&cliConfig.ConfigFile, "config", "", "Ruta al archivo de configuración YAML/JSON")
	flag.BoolVar(&cliConfig.HealthCheck, "health", false, "Realizar health check y salir")
	flag.BoolVar(&cliConfig.ShowMetrics, "metrics", false, "Mostrar métricas y salir")
	flag.StringVar(&cliConfig.CreateConfig, "create-config", "", "Crear archivo de configuración por defecto en la ruta especificada")

	flag.Parse()

	// Crear configuración por defecto
	if cliConfig.CreateConfig != "" {
		return &cliConfig, nil
	}

	// Validaciones básicas
	if cliConfig.HealthCheck || cliConfig.ShowMetrics {
		return &cliConfig, nil
	}

	if cliConfig.ScannerConfig.Domain == "" {
		return nil, fmt.Errorf("el flag -d es requerido")
	}

	return &cliConfig, nil
}
