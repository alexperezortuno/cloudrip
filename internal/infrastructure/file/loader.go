package file

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/alexperezortuno/cloudrip/internal/domain"
)

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

func LoadWordlist(path string) ([]string, error) {
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

func LoadDefaultCFRanges() domain.CfRanges {
	return domain.CfRanges{
		IPv4: mustParsePrefixes(defaultCFIPv4),
		IPv6: mustParsePrefixes(defaultCFIPv6),
	}
}

func FetchCFRanges(ctx context.Context) (domain.CfRanges, error) {
	type cfResp struct {
		Result struct {
			IPv4 []string `json:"ipv4_cidrs"`
			IPv6 []string `json:"ipv6_cidrs"`
		} `json:"result"`
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfAPI, nil)
	if err != nil {
		return domain.CfRanges{}, err
	}
	httpClient := &http.Client{Timeout: 10 * time.Second}
	res, err := httpClient.Do(req)
	if err != nil {
		return domain.CfRanges{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return domain.CfRanges{}, fmt.Errorf("cloudflare api status: %s", res.Status)
	}
	var d cfResp
	if err := json.NewDecoder(res.Body).Decode(&d); err != nil {
		return domain.CfRanges{}, err
	}
	return domain.CfRanges{
		IPv4: mustParsePrefixes(d.Result.IPv4),
		IPv6: mustParsePrefixes(d.Result.IPv6),
	}, nil
}
