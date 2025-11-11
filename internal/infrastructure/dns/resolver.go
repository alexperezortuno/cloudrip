package dns

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type Resolver struct {
	resolver *net.Resolver
	logger   zerolog.Logger
}

func NewResolver(logger zerolog.Logger) *Resolver {
	return &Resolver{
		resolver: &net.Resolver{
			PreferGo: true,
		},
		logger: logger,
	}
}

func (r *Resolver) LookupIP(ctx context.Context, fqdn string) ([]string, error) {
	r.logger.Debug().Str("fqdn", fqdn).Msg("Resolviendo IPs")

	ips, err := r.resolver.LookupIPAddr(ctx, fqdn)
	if err != nil {
		r.logger.Debug().Err(err).Str("fqdn", fqdn).Msg("Error resolviendo IPs")
		return nil, err
	}

	result := make([]string, 0, len(ips))
	for _, ip := range ips {
		result = append(result, ip.IP.String())
	}

	r.logger.Debug().Str("fqdn", fqdn).Int("ips", len(result)).Msg("IPs resueltas")
	return result, nil
}

func (r *Resolver) LookupCNAME(ctx context.Context, fqdn string) (string, error) {
	r.logger.Debug().Str("fqdn", fqdn).Msg("Resolviendo CNAME")

	target, err := r.resolver.LookupCNAME(ctx, fqdn)
	if err != nil {
		r.logger.Debug().Err(err).Str("fqdn", fqdn).Msg("Error resolviendo CNAME")
		return "", err
	}

	// Limpiar el punto final
	target = strings.TrimSuffix(target, ".")

	r.logger.Debug().Str("fqdn", fqdn).Str("target", target).Msg("CNAME resuelto")
	return target, nil
}

// LookupIPWithRetry implementa reintentos con backoff
func (r *Resolver) LookupIPWithRetry(ctx context.Context, fqdn string, retries int, backoff time.Duration) ([]string, error) {
	var lastErr error

	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			r.logger.Debug().
				Str("fqdn", fqdn).
				Int("attempt", attempt).
				Msg("Reintentando resolución DNS")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff * time.Duration(attempt)):
			}
		}

		ips, err := r.LookupIP(ctx, fqdn)
		if err == nil {
			return ips, nil
		}
		lastErr = err
	}

	return nil, lastErr
}
