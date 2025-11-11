package cli

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/alexperezortuno/cloudrip/internal/domain"
	"github.com/alexperezortuno/cloudrip/internal/infrastructure/file"
)

type CLIParser struct{}

func NewCLIParser() *CLIParser {
	return &CLIParser{}
}

func (c *CLIParser) ParseFlags() (*domain.ScannerConfig, error) {
	config := &domain.ScannerConfig{}

	flag.StringVar(&config.Domain, "d", "", "Dominio objetivo (ej: example.com) [requerido]")
	flag.StringVar(&config.WordList, "w", "dom.txt", "Ruta al wordlist (uno por línea)")
	flag.IntVar(&config.Threads, "t", 10, "Número de workers concurrentes")
	flag.StringVar(&config.Output, "o", "", "Ruta de salida (opcional)")
	flag.StringVar(&config.OutputFmt, "output-format", "text", "Formato de salida: text|json")
	flag.IntVar(&config.Retries, "retries", 2, "Reintentos por resolución")
	flag.DurationVar(&config.Backoff, "backoff", 500*time.Millisecond, "Backoff base entre reintentos")
	flag.DurationVar(&config.Timeout, "timeout", 5*time.Second, "Timeout por consulta DNS")
	flag.DurationVar(&config.Delay, "delay", 0, "Delay por job (throttling)")
	flag.BoolVar(&config.FollowCNAME, "follow-cname", false, "Seguir un nivel de CNAME")
	flag.BoolVar(&config.IncludeCF, "include-cf", false, "Incluir IPs pertenecientes a Cloudflare en resultados")
	flag.BoolVar(&config.NoFetchCF, "no-fetch-cf", false, "No intentar actualizar CIDRs de Cloudflare desde Internet")

	flag.Parse()

	if config.Domain == "" {
		return nil, fmt.Errorf("error: -d <domain> es requerido")
	}

	subs, err := file.LoadWordlist(config.WordList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Wordlist: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[INFO] Cargados %d subdominios de %s\n", len(subs), config.WordList)
	config.SubDomain = subs

	return config, nil
}
