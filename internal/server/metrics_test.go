package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gp42/aws-outbound-jwt-proxy/internal/config"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/forwarder"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/metrics"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/router"
)

func TestHandlerRecordsMetrics(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer upstream.Close()

	p, inst := newProvider(t)
	defer p.Shutdown(context.Background())

	cfg, u := newCfg(upstream, time.Second)
	r := router.New(u.Host, u.Scheme, cfg.HostHeader)
	h := handler(r, forwarder.New(cfg, inst, nil, nil), inst)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/ok", nil))
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}

	body := scrape(t, p)
	for _, want := range []string{
		`http_server_request_duration_seconds_count{`,
		`http_request_method="GET"`,
		`http_response_status_code="200"`,
		`http_client_request_duration_seconds_count{`,
		`server_address="127.0.0.1"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in metrics:\n%s", want, body)
		}
	}
	// Success path must not carry error.type.
	if strings.Contains(body, `error_type="`) {
		t.Errorf("success path should not emit error_type label:\n%s", body)
	}
	for _, forbidden := range []string{
		"http_server_requests_total",
		"upstream_requests_total",
		`status_class="`,
		`outcome="`,
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("legacy token %q still present", forbidden)
		}
	}
}

func TestHandlerNoUpstreamClassified400(t *testing.T) {
	p, inst := newProvider(t)
	defer p.Shutdown(context.Background())

	cfg := &config.Config{HostHeader: "X-Upstream-Host", UpstreamScheme: "https", UpstreamTimeout: time.Second}
	r := router.New("", cfg.UpstreamScheme, cfg.HostHeader)
	h := handler(r, forwarder.New(cfg, inst, nil, nil), inst)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/foo", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}

	body := scrape(t, p)
	if !strings.Contains(body, `http_response_status_code="400"`) {
		t.Errorf("expected 400 series:\n%s", body)
	}
}

func TestHandlerUnknownMethodBucketedOther(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer upstream.Close()

	p, inst := newProvider(t)
	defer p.Shutdown(context.Background())

	cfg, u := newCfg(upstream, time.Second)
	r := router.New(u.Host, u.Scheme, cfg.HostHeader)
	h := handler(r, forwarder.New(cfg, inst, nil, nil), inst)

	req := httptest.NewRequest("FOO", "/x", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)

	body := scrape(t, p)
	if !strings.Contains(body, `http_request_method="_OTHER"`) {
		t.Errorf("expected _OTHER bucket:\n%s", body)
	}
	if strings.Contains(body, `http_request_method="FOO"`) {
		t.Errorf("unknown method leaked as label:\n%s", body)
	}
}

func newProvider(t *testing.T) (*metrics.Provider, *metrics.Instruments) {
	t.Helper()
	p, err := metrics.New(&config.Config{MetricsEnabled: true, MetricsListenAddr: ":0", MetricsPath: "/metrics"})
	if err != nil {
		t.Fatalf("metrics.New: %v", err)
	}
	inst, err := p.Instruments()
	if err != nil {
		t.Fatalf("Instruments: %v", err)
	}
	return p, inst
}

func scrape(t *testing.T, p *metrics.Provider) string {
	t.Helper()
	srv := httptest.NewServer(p.Handler("/metrics"))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}
