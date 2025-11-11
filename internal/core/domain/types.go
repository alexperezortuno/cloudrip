package domain

import (
	"time"
)

// ResultEntry representa un resultado de escaneo
type ResultEntry struct {
	FQDN string `json:"fqdn"`
	IP   string `json:"ip"`
	Type string `json:"type"`
}

// CFRanges representa los rangos de IP de Cloudflare
type CFRanges struct {
	IPv4 []string `json:"ipv4_cidrs"`
	IPv6 []string `json:"ipv6_cidrs"`
}

// ScannerConfig contiene la configuración del escaneo
type ScannerConfig struct {
	Domain      string        `yaml:"domain" json:"domain"`
	Wordlist    string        `yaml:"wordlist" json:"wordlist"`
	Threads     int           `yaml:"threads" json:"threads"`
	Retries     int           `yaml:"retries" json:"retries"`
	Backoff     time.Duration `yaml:"backoff" json:"backoff"`
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`
	Delay       time.Duration `yaml:"delay" json:"delay"`
	FollowCNAME bool          `yaml:"follow_cname" json:"follow_cname"`
	IncludeCF   bool          `yaml:"include_cf" json:"include_cf"`
	NoFetchCF   bool          `yaml:"no_fetch_cf" json:"no_fetch_cf"`
	Output      string        `yaml:"output" json:"output"`
	OutputFmt   string        `yaml:"output_format" json:"output_format"`
}

// ScanResult representa el resultado completo del escaneo
type ScanResult struct {
	TotalFound int                      `json:"total_found"`
	Duration   time.Duration            `json:"duration"`
	Results    map[string][]ResultEntry `json:"results"`
}

// Job representa un trabajo de escaneo
type Job struct {
	Subdomain string `json:"subdomain"`
}

// ResolverOptions contiene opciones para el resolver DNS
type ResolverOptions struct {
	Retries int
	Backoff time.Duration
	Timeout time.Duration
}

// Metrics contiene métricas de performance
type Metrics struct {
	StartTime     time.Time          `json:"start_time"`
	EndTime       time.Time          `json:"end_time"`
	TotalJobs     int                `json:"total_jobs"`
	CompletedJobs int                `json:"completed_jobs"`
	SuccessCount  int                `json:"success_count"`
	ErrorCount    int                `json:"error_count"`
	DNSQueries    int                `json:"dns_queries"`
	WorkerStats   map[int]WorkerStat `json:"worker_stats"`
}

// WorkerStat contiene estadísticas por worker
type WorkerStat struct {
	WorkerID     int       `json:"worker_id"`
	JobsDone     int       `json:"jobs_done"`
	Errors       int       `json:"errors"`
	LastActivity time.Time `json:"last_activity"`
}

// HealthStatus representa el estado de salud del sistema
type HealthStatus struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
	Metrics   Metrics   `json:"metrics"`
}
