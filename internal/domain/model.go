package domain

import (
	"net/netip"
	"time"
)

type ResultEntry struct {
	FQDN string `json:"fqdn"`
	IP   string `json:"ip"`
	Type string `json:"type"` // "A" o "AAAA"
}

type CfRanges struct {
	IPv4 []netip.Prefix
	IPv6 []netip.Prefix
}

var (
	DefaultCFIPv4 = []string{
		"103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22",
		"104.16.0.0/13", "104.24.0.0/14", "108.162.192.0/18",
		"131.0.72.0/22", "141.101.64.0/18", "162.158.0.0/15",
		"172.64.0.0/13", "173.245.48.0/20", "188.114.96.0/20",
		"190.93.240.0/20", "197.234.240.0/22", "198.41.128.0/17",
	}
	DefaultCFIPv6 = []string{
		"2400:cb00::/32", "2606:4700::/32", "2803:f800::/32",
		"2405:b500::/32", "2405:8100::/32", "2a06:98c0::/29",
		"2c0f:f248::/32",
	}
	CfAPI = "https://api.cloudflare.com/client/v4/ips"
)

type ResolverOptions struct {
	Retries int
	Backoff time.Duration
	Timeout time.Duration
}

type Job struct {
	Subdomain string
}

type ScannerConfig struct {
	Domain      string
	WordList    string
	Threads     int
	Retries     int
	Backoff     time.Duration
	Timeout     time.Duration
	Delay       time.Duration
	FollowCNAME bool
	IncludeCF   bool
	NoFetchCF   bool
	Output      string
	OutputFmt   string
	SubDomain   []string
}
