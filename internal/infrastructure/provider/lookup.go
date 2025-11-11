package provider

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/alexperezortuno/cloudrip/internal/domain"
)

func LookupIPs(ctx context.Context, r *net.Resolver, fqdn string, opts domain.ResolverOptions) ([]net.IPAddr, error) {
	var (
		lastErr error
		delay   = opts.Backoff
	)
	for attempt := 0; attempt <= opts.Retries; attempt++ {
		ictx, cancel := context.WithTimeout(ctx, opts.Timeout)
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

func LookupCNAME(ctx context.Context, r *net.Resolver, fqdn string, opts domain.ResolverOptions) (string, error) {
	var (
		lastErr error
		delay   = opts.Backoff
	)
	for attempt := 0; attempt <= opts.Retries; attempt++ {
		ictx, cancel := context.WithTimeout(ctx, opts.Timeout)
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
