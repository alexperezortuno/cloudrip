package service

import (
	"context"
	"net"
	"sync"
	"time"

	d "github.com/alexperezortuno/cloudrip/internal/domain"
	"github.com/alexperezortuno/cloudrip/internal/infrastructure/provider"
)

func Worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	jobs <-chan d.Job,
	results chan<- d.ResultEntry,
	r *net.Resolver,
	domain string,
	cf d.CfRanges,
	includeCF bool,
	followCNAME bool,
	delayPerJob time.Duration,
	opts d.ResolverOptions,
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

			fqdn := jb.Subdomain
			if jb.Subdomain != "" {
				fqdn = jb.Subdomain + "." + domain
			} else {
				fqdn = domain
			}

			addrs, err := provider.LookupIPs(ctx, r, fqdn, opts)
			if err == nil && len(addrs) > 0 {
				for _, a := range addrs {
					ip := a.IP.String()

					ipType := "A"
					if a.IP.To4() == nil {
						ipType = "AAAA"
					}
					isCF := provider.IsCloudflareIP(ip, cf)

					if !isCF || includeCF {
						select {
						case <-ctx.Done():
							return
						case results <- d.ResultEntry{FQDN: fqdn, IP: ip, Type: ipType}:
						}

						if !isCF {
							continue
						}
					}
				}
				continue
			}

			// Si no encontramos y se pide seguir CNAME (un nivel)
			if followCNAME {
				target, err := provider.LookupCNAME(ctx, r, fqdn, opts)
				if err == nil && target != "" {
					addrs2, err2 := provider.LookupIPs(ctx, r, target, opts)
					if err2 == nil && len(addrs2) > 0 {
						for _, a := range addrs2 {
							ip := a.IP.String()
							ipType := "A"
							if a.IP.To4() == nil {
								ipType = "AAAA"
							}
							isCF := provider.IsCloudflareIP(ip, cf)
							if !isCF || includeCF {
								select {
								case <-ctx.Done():
									return
								case results <- d.ResultEntry{FQDN: fqdn, IP: ip, Type: ipType}:
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
