package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alexperezortuno/cloudrip/internal/core/domain"
	"github.com/rs/zerolog"
)

type workerPool struct {
	scanner *Scanner
	config  domain.ScannerConfig
	ranges  domain.CFRanges
	logger  zerolog.Logger
}

func (s *Scanner) startWorkers(ctx context.Context, config domain.ScannerConfig, subs []string, ranges domain.CFRanges) map[string][]domain.ResultEntry {
	pool := &workerPool{
		scanner: s,
		config:  config,
		ranges:  ranges,
		logger:  s.logger.With().Str("component", "worker_pool").Logger(),
	}

	return pool.execute(ctx, subs)
}

func (wp *workerPool) execute(ctx context.Context, subs []string) map[string][]domain.ResultEntry {
	jobs := make(chan domain.Job, len(subs))
	results := make(chan domain.ResultEntry, len(subs)*2)

	collector := NewResultCollector(wp.logger)

	// Iniciar workers
	var wg sync.WaitGroup

	for i := 0; i < wp.config.Threads; i++ {
		wg.Add(1)
		go wp.worker(ctx, &wg, i, jobs, results)
	}

	// Recolector de resultados
	var collectWg sync.WaitGroup
	collectWg.Add(1)
	go func() {
		defer collectWg.Done()
		for result := range results {
			collector.Collect(result)
			if wp.scanner.progressReporter != nil {
				wp.scanner.progressReporter.Increment()
			}
		}
	}()

	// Alimentar jobs
	go func() {
		defer close(jobs)
		for _, sub := range subs {
			select {
			case <-ctx.Done():
				wp.logger.Warn().Msg("Contexto cancelado, deteniendo alimentación de jobs")
				return
			case jobs <- domain.Job{Subdomain: sub}:
				wp.scanner.metricsCollector.IncrementDNSQuery()
			}
		}
		wp.logger.Debug().Msg("Todos los jobs han sido enviados")
	}()

	// Esperar finalización de workers
	wg.Wait()
	close(results)
	collectWg.Wait()

	wp.logger.Info().Int("results", len(collector.GetResults())).Msg("Workers finalizados")

	return collector.GetResults()
}

func (wp *workerPool) worker(ctx context.Context, wg *sync.WaitGroup, id int, jobs <-chan domain.Job, results chan<- domain.ResultEntry) {
	defer wg.Done()

	wp.logger.Debug().Int("worker_id", id).Msg("Worker iniciado")

	for job := range jobs {
		select {
		case <-ctx.Done():
			wp.logger.Debug().Int("worker_id", id).Msg("Worker interrumpido")
			return
		default:
			err := wp.processJob(ctx, job, results)
			if err != nil {
				wp.scanner.metricsCollector.IncrementError()
			} else {
				wp.scanner.metricsCollector.IncrementSuccess()
			}
			wp.scanner.metricsCollector.RecordWorkerActivity(id)
		}
	}

	wp.logger.Debug().Int("worker_id", id).Msg("Worker finalizado")
}

func (wp *workerPool) processJob(ctx context.Context, job domain.Job, results chan<- domain.ResultEntry) error {
	if wp.config.Delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wp.config.Delay):
		}
	}

	fqdn := wp.buildFQDN(job.Subdomain)

	// Resolver IPs
	ips, err := wp.scanner.dnsResolver.LookupIP(ctx, fqdn)
	if err != nil {
		wp.logger.Debug().Err(err).Str("fqdn", fqdn).Msg("Error en lookup IP")
	} else if len(ips) > 0 {
		wp.processIPs(fqdn, ips, results)
	}

	// Seguir CNAME si está habilitado
	if wp.config.FollowCNAME {
		wp.processCNAME(ctx, fqdn, results)
	}

	return err
}

func (wp *workerPool) buildFQDN(subdomain string) string {
	if subdomain == "" {
		return wp.config.Domain
	}
	return fmt.Sprintf("%s.%s", subdomain, wp.config.Domain)
}

func (wp *workerPool) processIPs(fqdn string, ips []string, results chan<- domain.ResultEntry) {
	for _, ip := range ips {
		ipType := "A"
		if len(ip) > 16 { // IPv6 básico check
			ipType = "AAAA"
		}

		isCF := wp.scanner.cloudflareService.IsCloudflareIP(ip, wp.ranges)
		if !isCF || wp.config.IncludeCF {
			results <- domain.ResultEntry{
				FQDN: fqdn,
				IP:   ip,
				Type: ipType,
			}
			wp.logger.Debug().
				Str("fqdn", fqdn).
				Str("ip", ip).
				Str("type", ipType).
				Bool("cloudflare", isCF).
				Msg("Resultado encontrado")
		}
	}
}

func (wp *workerPool) processCNAME(ctx context.Context, fqdn string, results chan<- domain.ResultEntry) {
	target, err := wp.scanner.dnsResolver.LookupCNAME(ctx, fqdn)
	if err != nil || target == "" {
		return
	}

	ips, err := wp.scanner.dnsResolver.LookupIP(ctx, target)
	if err != nil || len(ips) == 0 {
		return
	}

	wp.processIPs(fqdn, ips, results)
}
