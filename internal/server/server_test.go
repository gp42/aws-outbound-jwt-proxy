package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gp42/aws-outbound-jwt-proxy/internal/config"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/forwarder"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/metrics"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/router"
)

func newCfg(upstream *httptest.Server, timeout time.Duration) (*config.Config, *url.URL) {
	u, _ := url.Parse(upstream.URL)
	return &config.Config{
		UpstreamHost:    u.Host,
		UpstreamScheme:  u.Scheme,
		HostHeader:      "X-Upstream-Host",
		UpstreamTimeout: timeout,
	}, u
}

func TestHandlerForwardsToUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/items" || r.URL.RawQuery != "limit=10" {
			t.Errorf("upstream got path=%q query=%q", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("X-From-Upstream", "yes")
		w.WriteHeader(200)
		_, _ = w.Write([]byte("hello"))
	}))
	defer upstream.Close()

	cfg, _ := newCfg(upstream, time.Second)
	r := router.New(cfg.UpstreamHost, cfg.UpstreamScheme, cfg.HostHeader)
	h := handler(r, forwarder.New(cfg, metrics.NoopInstruments(), nil, nil), metrics.NoopInstruments())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/items?limit=10", nil))

	if rec.Code != 200 {
		t.Fatalf("status=%d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "hello" {
		t.Fatalf("body=%q", body)
	}
	if rec.Header().Get("X-From-Upstream") != "yes" {
		t.Fatalf("upstream headers not propagated")
	}
}

func TestHandlerNoUpstream400(t *testing.T) {
	cfg := &config.Config{HostHeader: "X-Upstream-Host", UpstreamScheme: "https", UpstreamTimeout: time.Second}
	r := router.New("", cfg.UpstreamScheme, cfg.HostHeader)
	h := handler(r, forwarder.New(cfg, metrics.NoopInstruments(), nil, nil), metrics.NoopInstruments())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/foo", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "X-Upstream-Host") {
		t.Fatalf("body missing header name: %s", body)
	}
}

func TestHandlerHeaderModeStripsRoutingHeader(t *testing.T) {
	var seen http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
		w.WriteHeader(204)
	}))
	defer upstream.Close()
	u, _ := url.Parse(upstream.URL)

	cfg := &config.Config{
		UpstreamHost:    "",
		UpstreamScheme:  u.Scheme,
		HostHeader:      "X-Upstream-Host",
		UpstreamTimeout: time.Second,
	}
	r := router.New("", cfg.UpstreamScheme, cfg.HostHeader)
	h := handler(r, forwarder.New(cfg, metrics.NoopInstruments(), nil, nil), metrics.NoopInstruments())

	req := httptest.NewRequest("GET", "/foo", nil)
	req.Header.Set("X-Upstream-Host", u.Host)
	req.Header.Set("X-Custom", "keep-me")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if seen.Get("X-Upstream-Host") != "" {
		t.Fatalf("routing header leaked: %v", seen)
	}
	if seen.Get("X-Custom") != "keep-me" {
		t.Fatalf("custom header not forwarded")
	}
	if seen.Get("X-Forwarded-For") == "" {
		t.Fatalf("X-Forwarded-For missing")
	}
}

func TestHandlerHostRewritten(t *testing.T) {
	var sawHost string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawHost = r.Host
		w.WriteHeader(204)
	}))
	defer upstream.Close()
	u, _ := url.Parse(upstream.URL)

	cfg := &config.Config{
		UpstreamHost:    u.Host,
		UpstreamScheme:  u.Scheme,
		HostHeader:      "X-Upstream-Host",
		UpstreamTimeout: time.Second,
	}
	r := router.New(cfg.UpstreamHost, cfg.UpstreamScheme, cfg.HostHeader)
	h := handler(r, forwarder.New(cfg, metrics.NoopInstruments(), nil, nil), metrics.NoopInstruments())

	req := httptest.NewRequest("GET", "/foo", nil)
	req.Host = "proxy.example.com"
	h.ServeHTTP(httptest.NewRecorder(), req)
	if sawHost != u.Host {
		t.Fatalf("upstream saw Host=%q, want %q", sawHost, u.Host)
	}
}

func TestHandlerUpstreamUnreachable502(t *testing.T) {
	// closed listener => connection refused
	l := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := l.Listener.Addr().String()
	l.Close()

	u, _ := url.Parse("http://" + addr)
	cfg := &config.Config{
		UpstreamHost:    u.Host,
		UpstreamScheme:  u.Scheme,
		HostHeader:      "X-Upstream-Host",
		UpstreamTimeout: time.Second,
	}
	r := router.New(cfg.UpstreamHost, cfg.UpstreamScheme, cfg.HostHeader)
	h := handler(r, forwarder.New(cfg, metrics.NoopInstruments(), nil, nil), metrics.NoopInstruments())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/foo", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestHandlerUpstreamSlow504(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer upstream.Close()
	u, _ := url.Parse(upstream.URL)
	cfg := &config.Config{
		UpstreamHost:    u.Host,
		UpstreamScheme:  u.Scheme,
		HostHeader:      "X-Upstream-Host",
		UpstreamTimeout: 20 * time.Millisecond,
	}
	r := router.New(cfg.UpstreamHost, cfg.UpstreamScheme, cfg.HostHeader)
	h := handler(r, forwarder.New(cfg, metrics.NoopInstruments(), nil, nil), metrics.NoopInstruments())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/foo", nil))
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("got %d", rec.Code)
	}
}
