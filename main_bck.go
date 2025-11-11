package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

type resultEntry struct {
	FQDN string `json:"fqdn"`
	IP   string `json:"ip"`
	Type string `json:"type"` // "A" o "AAAA"
}

type cfRanges struct {
	IPv4 []netip.Prefix
	IPv6 []netip.Prefix
}

var (
	defaultCFIPv4 = []string{
		"103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22",
		"104.16.0.0/13", "104.24.0.0/14", "108.162.192.0/18",
		"131.0.72.0/22", "141.101.64.0/18", "162.158.0.0/15",
		"172.64.0.0/13", "173.245.48.0/20", "188.114.96.0/20",
		"190.93.240.0/20", "197.234.240.0/22", "198.41.128.0/17",
	}
	defaultCFIPv6 = []string{
		"2400:cb00::/32", "2606:4700::/32", "2803:f800::/32",
		"2405:b500::/32", "2405:8100::/32", "2a06:98c0::/29",
		"2c0f:f248::/32",
	}
	cfAPI = "https://api.cloudflare.com/client/v4/ips"
)

func mustParsePrefixes(cidrs []string) []netip.Prefix {
	pfxs := make([]netip.Prefix, 0, len(cidrs))
	for _, c := range cidrs {
		p, err := netip.ParsePrefix(c)
		if err == nil {
			pfxs = append(pfxs, p)
		}
	}
	return pfxs
}

func loadDefaultCFRanges() cfRanges {
	return cfRanges{
		IPv4: mustParsePrefixes(defaultCFIPv4),
		IPv6: mustParsePrefixes(defaultCFIPv6),
	}
}

func fetchCFRanges(ctx context.Context) (cfRanges, error) {
	type cfResp struct {
		Result struct {
			IPv4 []string `json:"ipv4_cidrs"`
			IPv6 []string `json:"ipv6_cidrs"`
		} `json:"result"`
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfAPI, nil)
	if err != nil {
		return cfRanges{}, err
	}
	httpClient := &http.Client{Timeout: 10 * time.Second}
	res, err := httpClient.Do(req)
	if err != nil {
		return cfRanges{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return cfRanges{}, fmt.Errorf("cloudflare api status: %s", res.Status)
	}
	var d cfResp
	if err := json.NewDecoder(res.Body).Decode(&d); err != nil {
		return cfRanges{}, err
	}
	return cfRanges{
		IPv4: mustParsePrefixes(d.Result.IPv4),
		IPv6: mustParsePrefixes(d.Result.IPv6),
	}, nil
}

func isCloudflareIP(ipStr string, cf cfRanges) bool {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false
	}
	if addr.Is4() {
		for _, p := range cf.IPv4 {
			if p.Contains(addr) {
				return true
			}
		}
		return false
	}
	// IPv6
	for _, p := range cf.IPv6 {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

func loadWordlist(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		s := strings.TrimSpace(sc.Text())
		if s != "" {
			lines = append(lines, s)
		}
	}
	return lines, sc.Err()
}

type resolverOptions struct {
	retries int
	backoff time.Duration
	timeout time.Duration
}

func lookupIPs(ctx context.Context, r *net.Resolver, fqdn string, opts resolverOptions) ([]net.IPAddr, error) {
	var (
		lastErr error
		delay   = opts.backoff
	)
	for attempt := 0; attempt <= opts.retries; attempt++ {
		ictx, cancel := context.WithTimeout(ctx, opts.timeout)
		addrs, err := r.LookupIPAddr(ictx, fqdn)
		cancel()
		if err == nil {
			return addrs, nil
		}
		lastErr = err
		// Timed out or transient — backoff
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			delay *= 2
		}
	}
	return nil, lastErr
}

func lookupCNAME(ctx context.Context, r *net.Resolver, fqdn string, opts resolverOptions) (string, error) {
	var (
		lastErr error
		delay   = opts.backoff
	)
	for attempt := 0; attempt <= opts.retries; attempt++ {
		ictx, cancel := context.WithTimeout(ctx, opts.timeout)
		target, err := r.LookupCNAME(ictx, fqdn)
		cancel()
		if err == nil {
			// El stdlib retorna un FQDN con punto final; lo limpiamos
			return strings.TrimSuffix(target, "."), nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(delay):
			delay *= 2
		}
	}
	return "", lastErr
}

type job struct {
	subdomain string
}

func worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	jobs <-chan job,
	results chan<- resultEntry,
	r *net.Resolver,
	domain string,
	cf cfRanges,
	includeCF bool,
	followCNAME bool,
	delayPerJob time.Duration,
	opts resolverOptions,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case jb, ok := <-jobs:
			if !ok {
				return
			}
			if delayPerJob > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(delayPerJob):
				}
			}

			fqdn := jb.subdomain
			if jb.subdomain != "" {
				fqdn = jb.subdomain + "." + domain
			} else {
				fqdn = domain
			}

			addrs, err := lookupIPs(ctx, r, fqdn, opts)
			if err == nil && len(addrs) > 0 {
				for _, a := range addrs {
					ip := a.IP.String()
					// Determinar tipo
					ipType := "A"
					if a.IP.To4() == nil {
						ipType = "AAAA"
					}
					isCF := isCloudflareIP(ip, cf)
					if !isCF || includeCF {
						select {
						case <-ctx.Done():
							return
						case results <- resultEntry{FQDN: fqdn, IP: ip, Type: ipType}:
						}
						// si encontramos IP pública (no CF) devolvemos
						if !isCF {
							continue
						}
					}
				}
				continue
			}

			// Si no encontramos y se pide seguir CNAME (un nivel)
			if followCNAME {
				target, err := lookupCNAME(ctx, r, fqdn, opts)
				if err == nil && target != "" {
					addrs2, err2 := lookupIPs(ctx, r, target, opts)
					if err2 == nil && len(addrs2) > 0 {
						for _, a := range addrs2 {
							ip := a.IP.String()
							ipType := "A"
							if a.IP.To4() == nil {
								ipType = "AAAA"
							}
							isCF := isCloudflareIP(ip, cf)
							if !isCF || includeCF {
								select {
								case <-ctx.Done():
									return
								case results <- resultEntry{FQDN: fqdn, IP: ip, Type: ipType}:
								}
								if !isCF {
									// no short-circuit here; we still emit all if includeCF requested
								}
							}
						}
					}
				}
			}
		}
	}
}

func saveText(path string, m map[string][]resultEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// salida ordenada por fqdn
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	w := bufio.NewWriter(f)
	for _, k := range keys {
		entries := m[k]
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Type == entries[j].Type {
				return entries[i].IP < entries[j].IP
			}
			return entries[i].Type < entries[j].Type
		})
		for _, e := range entries {
			fmt.Fprintf(w, "%s -> %s (%s)\n", e.FQDN, e.IP, e.Type)
		}
	}
	return w.Flush()
}

func saveJSON(path string, m map[string][]resultEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var flat []resultEntry
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		entries := m[k]
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Type == entries[j].Type {
				return entries[i].IP < entries[j].IP
			}
			return entries[i].Type < entries[j].Type
		})
		flat = append(flat, entries...)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(flat)
}

func main() {
	// Flags
	domain := flag.String("d", "", "Dominio objetivo (ej: example.com) [requerido]")
	wordlist := flag.String("w", "dom.txt", "Ruta al wordlist (uno por línea)")
	threads := flag.Int("t", 10, "Número de workers concurrentes")
	output := flag.String("o", "", "Ruta de salida (opcional)")
	outputFmt := flag.String("output-format", "text", "Formato de salida: text|json")
	retries := flag.Int("retries", 2, "Reintentos por resolución")
	backoff := flag.Duration("backoff", 500*time.Millisecond, "Backoff base entre reintentos")
	timeout := flag.Duration("timeout", 5*time.Second, "Timeout por consulta DNS")
	delay := flag.Duration("delay", 0, "Delay por job (throttling)")
	followCNAME := flag.Bool("follow-cname", false, "Seguir un nivel de CNAME")
	includeCF := flag.Bool("include-cf", false, "Incluir IPs pertenecientes a Cloudflare en resultados")
	noFetchCF := flag.Bool("no-fetch-cf", false, "No intentar actualizar CIDRs de Cloudflare desde Internet")
	flag.Parse()

	if *domain == "" {
		fmt.Fprintln(os.Stderr, "Error: -d <domain> es requerido")
		os.Exit(2)
	}

	// Contexto con cancelación por señal
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Cargar wordlist
	subs, err := loadWordlist(*wordlist)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Wordlist: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[INFO] Cargados %d subdominios de %s\n", len(subs), *wordlist)

	// Rango Cloudflare
	var ranges cfRanges
	if *noFetchCF {
		ranges = loadDefaultCFRanges()
		fmt.Println("[INFO] Usando CIDRs Cloudflare por defecto (sin fetch).")
	} else {
		fmt.Println("[INFO] Obteniendo CIDRs Cloudflare oficiales...")
		ranges, err = fetchCFRanges(ctx)
		if err != nil {
			fmt.Printf("[WARN] No se pudo obtener desde API: %v. Usando defaults.\n", err)
			ranges = loadDefaultCFRanges()
		} else if len(ranges.IPv4) == 0 && len(ranges.IPv6) == 0 {
			fmt.Println("[WARN] Respuesta vacía. Usando defaults.")
			ranges = loadDefaultCFRanges()
		}
	}

	// Resolver
	resolver := net.Resolver{}

	// Canales y pool
	jobs := make(chan job)
	results := make(chan resultEntry, 1024)

	var wg sync.WaitGroup
	opts := resolverOptions{
		retries: *retries,
		backoff: *backoff,
		timeout: *timeout,
	}

	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go worker(ctx, &wg, jobs, results, &resolver, *domain, ranges, *includeCF, *followCNAME, *delay, opts)
	}

	// feeder
	go func() {
		defer close(jobs)
		for _, s := range subs {
			select {
			case <-ctx.Done():
				return
			case jobs <- job{subdomain: s}:
			}
		}
	}()

	// collector
	aggregated := make(map[string][]resultEntry)
	var collectWg sync.WaitGroup
	collectWg.Add(1)
	go func() {
		defer collectWg.Done()
		for r := range results {
			fmt.Printf("[FOUND] %s -> %s (%s)\n", r.FQDN, r.IP, r.Type)
			// evitar duplicados
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
	if *output != "" {
		switch strings.ToLower(*outputFmt) {
		case "json":
			if err := saveJSON(*output, aggregated); err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Guardando JSON: %v\n", err)
				os.Exit(1)
			}
		case "text":
			if err := saveText(*output, aggregated); err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Guardando texto: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintln(os.Stderr, "[ERROR] --output-format debe ser text|json")
			os.Exit(2)
		}
		fmt.Printf("[INFO] Resultados guardados en %s (formato=%s)\n", *output, *outputFmt)
	}

	fmt.Println("[INFO] Operación finalizada.")
}
