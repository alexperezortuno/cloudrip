package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/alexperezortuno/cloudrip/internal/application/service"
	"github.com/alexperezortuno/cloudrip/internal/domain"
	"github.com/alexperezortuno/cloudrip/internal/infrastructure/cli"
	"github.com/alexperezortuno/cloudrip/internal/infrastructure/file"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Parse CLI flags
	parser := cli.NewCLIParser()
	config, err := parser.ParseFlags()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// Rango Cloudflare
	var ranges domain.CfRanges
	if config.NoFetchCF {
		ranges = file.LoadDefaultCFRanges()
		fmt.Println("[INFO] Usando CIDRs Cloudflare por defecto (sin fetch).")
	} else {
		fmt.Println("[INFO] Obteniendo CIDRs Cloudflare oficiales...")
		ranges, err = file.FetchCFRanges(ctx)
		if err != nil {
			fmt.Printf("[WARN] No se pudo obtener desde API: %v. Usando defaults.\n", err)
			ranges = file.LoadDefaultCFRanges()
		} else if len(ranges.IPv4) == 0 && len(ranges.IPv6) == 0 {
			fmt.Println("[WARN] Respuesta vacía. Usando defaults.")
			ranges = file.LoadDefaultCFRanges()
		}
	}

	// Resolver
	resolver := net.Resolver{}

	// Canales y pool
	jobs := make(chan domain.Job)
	results := make(chan domain.ResultEntry, 1024)

	var wg sync.WaitGroup
	opts := domain.ResolverOptions{
		Retries: config.Retries,
		Backoff: config.Backoff,
		Timeout: config.Timeout,
	}

	for i := 0; i < config.Threads; i++ {
		wg.Add(1)
		go service.Worker(ctx, &wg, jobs, results, &resolver, config.Domain, ranges, config.IncludeCF, config.FollowCNAME, config.Delay, opts)
	}

	// feeder
	go func() {
		defer close(jobs)
		for _, s := range config.SubDomain {
			select {
			case <-ctx.Done():
				return
			case jobs <- domain.Job{Subdomain: s}:
			}
		}
	}()

	// collector
	aggregated := make(map[string][]domain.ResultEntry)
	var collectWg sync.WaitGroup
	collectWg.Add(1)
	go func() {
		defer collectWg.Done()
		for r := range results {
			fmt.Printf("[FOUND] %s -> %s (%s)\n", r.FQDN, r.IP, r.Type)
			list := aggregated[r.FQDN]
			dup := false
			for _, e := range list {
				if e.IP == r.IP && e.Type == r.Type {
					dup = true
					break
				}
			}
			if !dup {
				aggregated[r.FQDN] = append(aggregated[r.FQDN], r)
			}
		}
	}()

	// esperar workers
	wg.Wait()
	// cerrar results para que collector termine
	close(results)
	collectWg.Wait()

	// guardar si aplica
	if config.Output != "" {
		switch strings.ToLower(config.OutputFmt) {
		case "json":
			if err := file.SaveJSON(config.Output, aggregated); err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Guardando JSON: %v\n", err)
				os.Exit(1)
			}
		case "text":
			if err := file.SaveText(config.Output, aggregated); err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Guardando texto: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintln(os.Stderr, "[ERROR] --output-format debe ser text|json")
			os.Exit(2)
		}
		fmt.Printf("[INFO] Resultados guardados en %s (formato=%s)\n", config.Output, config.OutputFmt)
	}

	fmt.Println("[INFO] Operación finalizada.")
}
