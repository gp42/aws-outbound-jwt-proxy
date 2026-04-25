package config

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"
)

func newFS(t *testing.T, args []string) *pflag.FlagSet {
	t.Helper()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	BindFlags(fs)
	// Give every test a valid default audience so validation passes unless
	// the test is explicitly exercising the audience flag itself.
	hasAudience := false
	for _, a := range args {
		if strings.HasPrefix(a, "--token-audience") {
			hasAudience = true
			break
		}
	}
	if !hasAudience {
		args = append([]string{"--token-audience=https://default.test"}, args...)
	}
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse: %v", err)
	}
	return fs
}

// newFSRaw is like newFS but does not inject a default --token-audience.
// Use this only in tests that exercise the audience flag's validation.
func newFSRaw(t *testing.T, args []string) *pflag.FlagSet {
	t.Helper()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	BindFlags(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse: %v", err)
	}
	return fs
}

func noEnv(string) (string, bool) { return "", false }

func envMap(m map[string]string) func(string) (string, bool) {
	// Default TOKEN_AUDIENCE if the caller did not set one, so unrelated
	// tests don't fail validation.
	if _, ok := m["TOKEN_AUDIENCE"]; !ok {
		m["TOKEN_AUDIENCE"] = "https://default.test"
	}
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

// envMapRaw does not inject a default TOKEN_AUDIENCE; use for tests that
// exercise audience-flag validation.
func envMapRaw(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

func TestDefaults(t *testing.T) {
	fs := newFS(t, nil)
	cfg, err := Load(fs, noEnv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ListenAddr != ":8080" || cfg.UpstreamScheme != "https" || cfg.HostHeader != "X-Upstream-Host" || cfg.UpstreamHost != "" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestCLIOnly(t *testing.T) {
	fs := newFS(t, []string{"--upstream-host=api.example.com", "--upstream-scheme=http", "--host-header=X-T", "--listen-addr=:9000"})
	cfg, err := Load(fs, noEnv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.UpstreamHost != "api.example.com" || cfg.UpstreamScheme != "http" || cfg.HostHeader != "X-T" || cfg.ListenAddr != ":9000" {
		t.Fatalf("unexpected: %+v", cfg)
	}
}

func TestEnvOnly(t *testing.T) {
	fs := newFS(t, nil)
	env := envMap(map[string]string{
		"UPSTREAM_HOST":   "api.example.com",
		"UPSTREAM_SCHEME": "http",
		"HOST_HEADER":     "X-T",
		"LISTEN_ADDR":     ":9000",
	})
	cfg, err := Load(fs, env)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.UpstreamHost != "api.example.com" || cfg.UpstreamScheme != "http" || cfg.HostHeader != "X-T" || cfg.ListenAddr != ":9000" {
		t.Fatalf("unexpected: %+v", cfg)
	}
}

func TestCLIWinsOverEnv(t *testing.T) {
	fs := newFS(t, []string{"--upstream-host=cli.example.com"})
	env := envMap(map[string]string{"UPSTREAM_HOST": "env.example.com"})
	cfg, err := Load(fs, env)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.UpstreamHost != "cli.example.com" {
		t.Fatalf("expected cli.example.com, got %q", cfg.UpstreamHost)
	}
}

func TestInvalidScheme(t *testing.T) {
	fs := newFS(t, []string{"--upstream-scheme=ftp"})
	_, err := Load(fs, noEnv)
	if err == nil || !strings.Contains(err.Error(), "upstream-scheme") {
		t.Fatalf("expected scheme error, got %v", err)
	}
}

func TestUpstreamHostWithScheme(t *testing.T) {
	fs := newFS(t, []string{"--upstream-host=https://api.example.com"})
	_, err := Load(fs, noEnv)
	if err == nil || !strings.Contains(err.Error(), "upstream-scheme") {
		t.Fatalf("expected host-with-scheme error pointing to --upstream-scheme, got %v", err)
	}
}

func TestTLSNeitherSet(t *testing.T) {
	fs := newFS(t, nil)
	cfg, err := Load(fs, noEnv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TLSEnabled() {
		t.Fatalf("expected TLS disabled")
	}
}

func TestTLSBothSet(t *testing.T) {
	fs := newFS(t, []string{"--tls-cert=/c.pem", "--tls-key=/k.pem"})
	cfg, err := Load(fs, noEnv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.TLSEnabled() {
		t.Fatalf("expected TLS enabled")
	}
}

func TestTLSOnlyCert(t *testing.T) {
	fs := newFS(t, []string{"--tls-cert=/c.pem"})
	_, err := Load(fs, noEnv)
	if err == nil || !strings.Contains(err.Error(), "tls-cert") {
		t.Fatalf("expected both-or-neither error, got %v", err)
	}
}

func TestTLSOnlyKey(t *testing.T) {
	fs := newFS(t, []string{"--tls-key=/k.pem"})
	_, err := Load(fs, noEnv)
	if err == nil || !strings.Contains(err.Error(), "tls-key") {
		t.Fatalf("expected both-or-neither error, got %v", err)
	}
}

func TestUpstreamTimeoutDefault(t *testing.T) {
	fs := newFS(t, nil)
	cfg, err := Load(fs, noEnv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.UpstreamTimeout != 30*time.Second {
		t.Fatalf("got %s", cfg.UpstreamTimeout)
	}
}

func TestUpstreamTimeoutEnv(t *testing.T) {
	fs := newFS(t, nil)
	cfg, err := Load(fs, envMap(map[string]string{"UPSTREAM_TIMEOUT": "5s"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.UpstreamTimeout != 5*time.Second {
		t.Fatalf("got %s", cfg.UpstreamTimeout)
	}
}

func TestUpstreamTimeoutCLI(t *testing.T) {
	fs := newFS(t, []string{"--upstream-timeout=2s"})
	cfg, err := Load(fs, noEnv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.UpstreamTimeout != 2*time.Second {
		t.Fatalf("got %s", cfg.UpstreamTimeout)
	}
}

func TestUpstreamTimeoutInvalid(t *testing.T) {
	fs := newFS(t, []string{"--upstream-timeout=0s"})
	_, err := Load(fs, noEnv)
	if err == nil || !strings.Contains(err.Error(), "upstream-timeout") {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestLogDefaults(t *testing.T) {
	fs := newFS(t, nil)
	cfg, err := Load(fs, noEnv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LogLevel != "info" || cfg.LogFormat != "json" {
		t.Fatalf("defaults wrong: %+v", cfg)
	}
}

func TestLogLevelCaseInsensitive(t *testing.T) {
	fs := newFS(t, []string{"--log-level=DEBUG"})
	cfg, err := Load(fs, noEnv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("got %q", cfg.LogLevel)
	}
}

func TestLogLevelInvalid(t *testing.T) {
	fs := newFS(t, []string{"--log-level=trace"})
	_, err := Load(fs, noEnv)
	if err == nil || !strings.Contains(err.Error(), "log-level") {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestLogFormatInvalid(t *testing.T) {
	fs := newFS(t, []string{"--log-format=xml"})
	_, err := Load(fs, noEnv)
	if err == nil || !strings.Contains(err.Error(), "log-format") {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestMetricsDefaults(t *testing.T) {
	fs := newFS(t, nil)
	cfg, err := Load(fs, noEnv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.MetricsEnabled || cfg.MetricsListenAddr != ":9090" || cfg.MetricsPath != "/metrics" {
		t.Fatalf("metrics defaults wrong: %+v", cfg)
	}
}

func TestMetricsEnv(t *testing.T) {
	fs := newFS(t, nil)
	env := envMap(map[string]string{
		"METRICS_ENABLED":     "false",
		"METRICS_LISTEN_ADDR": ":7777",
		"METRICS_PATH":        "/m",
	})
	cfg, err := Load(fs, env)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MetricsEnabled || cfg.MetricsListenAddr != ":7777" || cfg.MetricsPath != "/m" {
		t.Fatalf("metrics env not applied: %+v", cfg)
	}
}

func TestMetricsConflictWithProxyListener(t *testing.T) {
	fs := newFS(t, []string{"--listen-addr=:8080", "--metrics-listen-addr=:8080"})
	_, err := Load(fs, noEnv)
	if err == nil || !strings.Contains(err.Error(), "metrics-listen-addr") {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestMetricsPathMissingSlash(t *testing.T) {
	fs := newFS(t, []string{"--metrics-path=metrics"})
	_, err := Load(fs, noEnv)
	if err == nil || !strings.Contains(err.Error(), "metrics-path") {
		t.Fatalf("expected path error, got %v", err)
	}
}

func TestMetricsEmptyAddr(t *testing.T) {
	fs := newFS(t, []string{"--metrics-listen-addr="})
	_, err := Load(fs, noEnv)
	if err == nil || !strings.Contains(err.Error(), "metrics-listen-addr") {
		t.Fatalf("expected empty-addr error, got %v", err)
	}
}

func TestTokenDefaults(t *testing.T) {
	fs := newFS(t, nil)
	cfg, err := Load(fs, noEnv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TokenSigningAlgorithm != "RS256" {
		t.Fatalf("algo: %q", cfg.TokenSigningAlgorithm)
	}
	if cfg.TokenDuration != time.Hour {
		t.Fatalf("duration: %s", cfg.TokenDuration)
	}
	if cfg.TokenRefreshSkew != 5*time.Minute {
		t.Fatalf("skew: %s", cfg.TokenRefreshSkew)
	}
}

func TestTokenEnv(t *testing.T) {
	fs := newFS(t, nil)
	env := envMap(map[string]string{
		"TOKEN_SIGNING_ALGORITHM": "ES384",
		"TOKEN_DURATION":          "10m",
		"TOKEN_REFRESH_SKEW":      "30s",
	})
	cfg, err := Load(fs, env)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TokenSigningAlgorithm != "ES384" || cfg.TokenDuration != 10*time.Minute || cfg.TokenRefreshSkew != 30*time.Second {
		t.Fatalf("env not applied: %+v", cfg)
	}
}

func TestTokenCLI(t *testing.T) {
	fs := newFS(t, []string{"--token-signing-algorithm=ES384", "--token-duration=5m", "--token-refresh-skew=10s"})
	cfg, err := Load(fs, noEnv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TokenSigningAlgorithm != "ES384" || cfg.TokenDuration != 5*time.Minute || cfg.TokenRefreshSkew != 10*time.Second {
		t.Fatalf("unexpected: %+v", cfg)
	}
}

func TestTokenSigningAlgorithmCaseInsensitive(t *testing.T) {
	fs := newFS(t, []string{"--token-signing-algorithm=rs256"})
	cfg, err := Load(fs, noEnv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TokenSigningAlgorithm != "RS256" {
		t.Fatalf("got %q", cfg.TokenSigningAlgorithm)
	}
}

func TestTokenSigningAlgorithmInvalid(t *testing.T) {
	fs := newFS(t, []string{"--token-signing-algorithm=HS256"})
	_, err := Load(fs, noEnv)
	if err == nil || !strings.Contains(err.Error(), "token-signing-algorithm") {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestTokenDurationBelowMin(t *testing.T) {
	fs := newFS(t, []string{"--token-duration=30s"})
	_, err := Load(fs, noEnv)
	if err == nil || !strings.Contains(err.Error(), "token-duration") {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestTokenDurationAboveMax(t *testing.T) {
	fs := newFS(t, []string{"--token-duration=2h"})
	_, err := Load(fs, noEnv)
	if err == nil || !strings.Contains(err.Error(), "token-duration") {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestTokenRefreshSkewZero(t *testing.T) {
	fs := newFS(t, []string{"--token-refresh-skew=0"})
	_, err := Load(fs, noEnv)
	if err == nil || !strings.Contains(err.Error(), "token-refresh-skew") {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestTokenRefreshSkewNotLessThanDuration(t *testing.T) {
	fs := newFS(t, []string{"--token-duration=60s", "--token-refresh-skew=60s"})
	_, err := Load(fs, noEnv)
	if err == nil || !strings.Contains(err.Error(), "token-refresh-skew") {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestTokenAudienceMissingAccepted(t *testing.T) {
	fs := newFSRaw(t, nil)
	cfg, err := Load(fs, envMapRaw(nil))
	if err != nil {
		t.Fatalf("expected Load to accept missing audience, got %v", err)
	}
	if len(cfg.TokenAudiences) != 0 {
		t.Fatalf("expected empty TokenAudiences, got %+v", cfg.TokenAudiences)
	}
}

func TestTokenAudienceSingleFlag(t *testing.T) {
	fs := newFSRaw(t, []string{"--token-audience=https://a.example.com"})
	cfg, err := Load(fs, envMapRaw(nil))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.TokenAudiences) != 1 || cfg.TokenAudiences[0] != "https://a.example.com" {
		t.Fatalf("got %+v", cfg.TokenAudiences)
	}
}

func TestTokenAudienceRepeatedFlag(t *testing.T) {
	fs := newFSRaw(t, []string{
		"--token-audience=https://a.example.com",
		"--token-audience=https://b.example.com",
	})
	cfg, err := Load(fs, envMapRaw(nil))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{"https://a.example.com", "https://b.example.com"}
	if len(cfg.TokenAudiences) != 2 || cfg.TokenAudiences[0] != want[0] || cfg.TokenAudiences[1] != want[1] {
		t.Fatalf("got %+v", cfg.TokenAudiences)
	}
}

func TestTokenAudienceEnvCommaList(t *testing.T) {
	fs := newFSRaw(t, nil)
	cfg, err := Load(fs, envMapRaw(map[string]string{"TOKEN_AUDIENCE": "https://a.example.com,https://b.example.com"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.TokenAudiences) != 2 || cfg.TokenAudiences[0] != "https://a.example.com" || cfg.TokenAudiences[1] != "https://b.example.com" {
		t.Fatalf("got %+v", cfg.TokenAudiences)
	}
}

func TestTokenAudienceCLIWinsOverEnv(t *testing.T) {
	fs := newFSRaw(t, []string{"--token-audience=https://cli.example.com"})
	cfg, err := Load(fs, envMapRaw(map[string]string{"TOKEN_AUDIENCE": "https://env.example.com"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.TokenAudiences) != 1 || cfg.TokenAudiences[0] != "https://cli.example.com" {
		t.Fatalf("expected CLI to win, got %+v", cfg.TokenAudiences)
	}
}

func TestTokenAudienceEmptyElementRejected(t *testing.T) {
	fs := newFSRaw(t, nil)
	_, err := Load(fs, envMapRaw(map[string]string{"TOKEN_AUDIENCE": "https://a.example.com,,https://b.example.com"}))
	if err == nil || !strings.Contains(err.Error(), "token-audience") {
		t.Fatalf("expected empty-value error, got %v", err)
	}
}

func TestTokenAudienceWhitespaceRejected(t *testing.T) {
	fs := newFSRaw(t, []string{"--token-audience=api example.com"})
	_, err := Load(fs, envMapRaw(nil))
	if err == nil || !strings.Contains(err.Error(), "whitespace") {
		t.Fatalf("expected whitespace error, got %v", err)
	}
}

func TestMetricsDisabledSkipsValidation(t *testing.T) {
	// With metrics disabled, same-address is allowed (and path need not start with /).
	fs := newFS(t, []string{"--metrics-enabled=false", "--metrics-listen-addr=:8080", "--metrics-path=whatever"})
	cfg, err := Load(fs, noEnv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MetricsEnabled {
		t.Fatalf("expected disabled")
	}
}
