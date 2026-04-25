package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

type Config struct {
	ListenAddr        string
	UpstreamHost      string
	UpstreamScheme    string
	HostHeader        string
	TLSCert           string
	TLSKey            string
	UpstreamTimeout   time.Duration

	TokenSigningAlgorithm string
	TokenDuration         time.Duration
	TokenRefreshSkew      time.Duration
	TokenAudiences        []string

	LogLevel          string
	LogFormat         string
	MetricsEnabled    bool
	MetricsListenAddr string
	MetricsPath       string
}

const (
	flagListenAddr        = "listen-addr"
	flagUpstreamHost      = "upstream-host"
	flagUpstreamScheme    = "upstream-scheme"
	flagHostHeader        = "host-header"
	flagTLSCert           = "tls-cert"
	flagTLSKey            = "tls-key"
	flagUpstreamTimeout   = "upstream-timeout"
	flagTokenSigningAlgo  = "token-signing-algorithm"
	flagTokenDuration     = "token-duration"
	flagTokenRefreshSkew  = "token-refresh-skew"
	flagTokenAudience     = "token-audience"
	flagLogLevel          = "log-level"
	flagLogFormat         = "log-format"
	flagMetricsEnabled    = "metrics-enabled"
	flagMetricsListenAddr = "metrics-listen-addr"
	flagMetricsPath       = "metrics-path"
)

func BindFlags(fs *pflag.FlagSet) {
	fs.String(flagListenAddr, ":8080", "host:port the server listens on")
	fs.String(flagUpstreamHost, "", "pinned upstream host; when set, the host header is ignored")
	fs.String(flagUpstreamScheme, "https", "scheme used when forwarding upstream (http or https)")
	fs.String(flagHostHeader, "X-Upstream-Host", "request header read for the upstream host when unpinned")
	fs.String(flagTLSCert, "", "path to PEM-encoded TLS certificate; enables HTTPS when set together with --tls-key")
	fs.String(flagTLSKey, "", "path to PEM-encoded TLS key; enables HTTPS when set together with --tls-cert")
	fs.Duration(flagUpstreamTimeout, 30*time.Second, "max time to wait for the upstream to send response headers")
	fs.String(flagTokenSigningAlgo, "RS256", "signing algorithm for STS-issued JWTs: RS256 or ES384")
	fs.Duration(flagTokenDuration, time.Hour, "requested JWT lifetime (60s to 1h)")
	fs.Duration(flagTokenRefreshSkew, 5*time.Minute, "refresh cached tokens this far before their expiration")
	fs.StringArray(flagTokenAudience, nil, "audience value sent to AWS STS; repeat the flag to request a JWT whose `aud` claim covers multiple audiences (optional — when unset, the audience is derived per-request from the outbound target host)")
	fs.String(flagLogLevel, "info", "log level: debug, info, warn, error")
	fs.String(flagLogFormat, "json", "log output format: json or text")
	fs.Bool(flagMetricsEnabled, true, "enable OpenTelemetry metrics collection and the Prometheus scrape endpoint")
	fs.String(flagMetricsListenAddr, ":9090", "host:port for the metrics (Prometheus scrape) listener; must differ from --listen-addr")
	fs.String(flagMetricsPath, "/metrics", "URL path where Prometheus metrics are served on the metrics listener")
}

func Load(fs *pflag.FlagSet, env func(string) (string, bool)) (*Config, error) {
	applyEnvFallback(fs, env)

	audiences, err := loadTokenAudiences(fs, env)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		ListenAddr:        mustString(fs, flagListenAddr),
		UpstreamHost:      mustString(fs, flagUpstreamHost),
		UpstreamScheme:    mustString(fs, flagUpstreamScheme),
		HostHeader:        mustString(fs, flagHostHeader),
		TLSCert:           mustString(fs, flagTLSCert),
		TLSKey:            mustString(fs, flagTLSKey),
		UpstreamTimeout:   mustDuration(fs, flagUpstreamTimeout),

		TokenSigningAlgorithm: strings.ToUpper(mustString(fs, flagTokenSigningAlgo)),
		TokenDuration:         mustDuration(fs, flagTokenDuration),
		TokenRefreshSkew:      mustDuration(fs, flagTokenRefreshSkew),
		TokenAudiences:        audiences,

		LogLevel:          strings.ToLower(mustString(fs, flagLogLevel)),
		LogFormat:         strings.ToLower(mustString(fs, flagLogFormat)),
		MetricsEnabled:    mustBool(fs, flagMetricsEnabled),
		MetricsListenAddr: mustString(fs, flagMetricsListenAddr),
		MetricsPath:       mustString(fs, flagMetricsPath),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) TLSEnabled() bool {
	return c.TLSCert != "" && c.TLSKey != ""
}

func (c *Config) validate() error {
	switch c.UpstreamScheme {
	case "http", "https":
	default:
		return fmt.Errorf("invalid --upstream-scheme %q: must be http or https", c.UpstreamScheme)
	}
	if strings.Contains(c.UpstreamHost, "://") {
		return fmt.Errorf("invalid --upstream-host %q: pass only host[:port]; use --upstream-scheme to set the scheme", c.UpstreamHost)
	}
	if c.HostHeader == "" {
		return fmt.Errorf("--host-header must not be empty")
	}
	if (c.TLSCert == "") != (c.TLSKey == "") {
		return fmt.Errorf("--tls-cert and --tls-key must both be set or both be empty")
	}
	if c.UpstreamTimeout <= 0 {
		return fmt.Errorf("--upstream-timeout must be positive, got %s", c.UpstreamTimeout)
	}
	switch c.TokenSigningAlgorithm {
	case "RS256", "ES384":
	default:
		return fmt.Errorf("invalid --token-signing-algorithm %q: must be RS256 or ES384", c.TokenSigningAlgorithm)
	}
	if c.TokenDuration < 60*time.Second || c.TokenDuration > 3600*time.Second {
		return fmt.Errorf("--token-duration must be between 60s and 3600s, got %s", c.TokenDuration)
	}
	if c.TokenRefreshSkew <= 0 {
		return fmt.Errorf("--token-refresh-skew must be positive, got %s", c.TokenRefreshSkew)
	}
	if c.TokenRefreshSkew >= c.TokenDuration {
		return fmt.Errorf("--token-refresh-skew (%s) must be strictly less than --token-duration (%s)", c.TokenRefreshSkew, c.TokenDuration)
	}
	for _, a := range c.TokenAudiences {
		if a == "" {
			return fmt.Errorf("--token-audience values must not be empty")
		}
		if strings.ContainsAny(a, " \t\n\r") {
			return fmt.Errorf("--token-audience value %q must not contain whitespace", a)
		}
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid --log-level %q: must be debug, info, warn, or error", c.LogLevel)
	}
	switch c.LogFormat {
	case "json", "text":
	default:
		return fmt.Errorf("invalid --log-format %q: must be json or text", c.LogFormat)
	}
	if c.MetricsEnabled {
		if c.MetricsListenAddr == "" {
			return fmt.Errorf("--metrics-listen-addr must not be empty when metrics are enabled")
		}
		if c.MetricsListenAddr == c.ListenAddr {
			return fmt.Errorf("--metrics-listen-addr %q must differ from --listen-addr", c.MetricsListenAddr)
		}
		if !strings.HasPrefix(c.MetricsPath, "/") {
			return fmt.Errorf("invalid --metrics-path %q: must begin with '/'", c.MetricsPath)
		}
	}
	return nil
}

func applyEnvFallback(fs *pflag.FlagSet, env func(string) (string, bool)) {
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			return
		}
		// --token-audience is a repeatable flag; env fallback handled in
		// loadTokenAudiences so TOKEN_AUDIENCE can carry a comma-separated
		// list without being stored as a single joined value.
		if f.Name == flagTokenAudience {
			return
		}
		envName := strings.ReplaceAll(strings.ToUpper(f.Name), "-", "_")
		if v, ok := env(envName); ok {
			_ = f.Value.Set(v)
		}
	})
}

// loadTokenAudiences returns the audiences from the repeatable CLI flag if
// it was set; otherwise from the TOKEN_AUDIENCE env var, split on commas.
// If neither is provided, returns nil (validation rejects this).
func loadTokenAudiences(fs *pflag.FlagSet, env func(string) (string, bool)) ([]string, error) {
	f := fs.Lookup(flagTokenAudience)
	if f != nil && f.Changed {
		vals, err := fs.GetStringArray(flagTokenAudience)
		if err != nil {
			return nil, err
		}
		return vals, nil
	}
	if v, ok := env("TOKEN_AUDIENCE"); ok {
		return strings.Split(v, ","), nil
	}
	return nil, nil
}

func mustString(fs *pflag.FlagSet, name string) string {
	v, err := fs.GetString(name)
	if err != nil {
		panic(err)
	}
	return v
}

func mustDuration(fs *pflag.FlagSet, name string) time.Duration {
	v, err := fs.GetDuration(name)
	if err != nil {
		panic(err)
	}
	return v
}

func mustBool(fs *pflag.FlagSet, name string) bool {
	v, err := fs.GetBool(name)
	if err != nil {
		panic(err)
	}
	return v
}
