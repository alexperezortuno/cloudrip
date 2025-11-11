package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/alexperezortuno/cloudrip/internal/core/domain"
	"github.com/rs/zerolog"
)

type Service struct {
	apiURL string
	client *http.Client
	logger zerolog.Logger
}

func NewService(logger zerolog.Logger) *Service {
	return &Service{
		apiURL: "https://api.cloudflare.com/client/v4/ips",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

func (s *Service) GetRanges(ctx context.Context, noFetch bool) (domain.CFRanges, error) {
	if noFetch {
		s.logger.Debug().Msg("Usando rangos Cloudflare por defecto")
		return s.loadDefaultRanges(), nil
	}

	s.logger.Debug().Msg("Obteniendo rangos Cloudflare desde API")
	return s.fetchRanges(ctx)
}

func (s *Service) IsCloudflareIP(ip string, ranges domain.CFRanges) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		s.logger.Debug().Err(err).Str("ip", ip).Msg("Error parseando IP")
		return false
	}

	// Convertir rangos de string a netip.Prefix
	ipv4Prefixes := s.parsePrefixes(ranges.IPv4)
	ipv6Prefixes := s.parsePrefixes(ranges.IPv6)

	if addr.Is4() {
		for _, prefix := range ipv4Prefixes {
			if prefix.Contains(addr) {
				return true
			}
		}
	} else {
		for _, prefix := range ipv6Prefixes {
			if prefix.Contains(addr) {
				return true
			}
		}
	}

	return false
}

func (s *Service) fetchRanges(ctx context.Context) (domain.CFRanges, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.apiURL, nil)
	if err != nil {
		return domain.CFRanges{}, fmt.Errorf("creando request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Error obteniendo rangos desde API, usando defaults")
		return s.loadDefaultRanges(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Warn().Int("status", resp.StatusCode).Msg("API retornó error, usando defaults")
		return s.loadDefaultRanges(), nil
	}

	var apiResponse struct {
		Success bool `json:"success"`
		Result  struct {
			IPv4 []string `json:"ipv4_cidrs"`
			IPv6 []string `json:"ipv6_cidrs"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		s.logger.Warn().Err(err).Msg("Error decodificando respuesta, usando defaults")
		return s.loadDefaultRanges(), nil
	}

	if !apiResponse.Success {
		s.logger.Warn().Msg("API retornó success=false, usando defaults")
		return s.loadDefaultRanges(), nil
	}

	s.logger.Debug().
		Int("ipv4_ranges", len(apiResponse.Result.IPv4)).
		Int("ipv6_ranges", len(apiResponse.Result.IPv6)).
		Msg("Rangos Cloudflare obtenidos desde API")

	return domain.CFRanges{
		IPv4: apiResponse.Result.IPv4,
		IPv6: apiResponse.Result.IPv6,
	}, nil
}

func (s *Service) loadDefaultRanges() domain.CFRanges {
	defaultIPv4 := []string{
		"103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22",
		"104.16.0.0/13", "104.24.0.0/14", "108.162.192.0/18",
		"131.0.72.0/22", "141.101.64.0/18", "162.158.0.0/15",
		"172.64.0.0/13", "173.245.48.0/20", "188.114.96.0/20",
		"190.93.240.0/20", "197.234.240.0/22", "198.41.128.0/17",
	}

	defaultIPv6 := []string{
		"2400:cb00::/32", "2606:4700::/32", "2803:f800::/32",
		"2405:b500::/32", "2405:8100::/32", "2a06:98c0::/29",
		"2c0f:f248::/32",
	}

	s.logger.Debug().
		Int("ipv4_ranges", len(defaultIPv4)).
		Int("ipv6_ranges", len(defaultIPv6)).
		Msg("Usando rangos Cloudflare por defecto")

	return domain.CFRanges{
		IPv4: defaultIPv4,
		IPv6: defaultIPv6,
	}
}

func (s *Service) parsePrefixes(cidrs []string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(cidrs))
	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(cidr)
		if err == nil {
			prefixes = append(prefixes, prefix)
		} else {
			s.logger.Debug().Err(err).Str("cidr", cidr).Msg("Error parseando CIDR")
		}
	}
	return prefixes
}
