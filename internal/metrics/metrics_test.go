package metrics

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gp42/aws-outbound-jwt-proxy/internal/config"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func enabledCfg() *config.Config {
	return &config.Config{MetricsEnabled: true, MetricsListenAddr: ":0", MetricsPath: "/metrics"}
}

func TestProviderDisabledReturnsNoopInstruments(t *testing.T) {
	p, err := New(&config.Config{MetricsEnabled: false})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p.Enabled() {
		t.Fatalf("expected disabled")
	}
	inst, err := p.Instruments()
	if err != nil {
		t.Fatalf("Instruments: %v", err)
	}
	ctx := context.Background()
	inst.HTTPServerActiveRequests.Add(ctx, 1)
	inst.HTTPServerRequestDuration.Record(ctx, 0.1)
	inst.HTTPClientRequestDuration.Record(ctx, 0.2)
	if err := p.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestProviderEnabledExposesSemconvMetrics(t *testing.T) {
	p, err := New(enabledCfg())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Shutdown(context.Background())

	inst, err := p.Instruments()
	if err != nil {
		t.Fatalf("Instruments: %v", err)
	}
	ctx := context.Background()
	inst.HTTPServerActiveRequests.Add(ctx, 1)
	inst.HTTPServerActiveRequests.Add(ctx, -1)
	inst.HTTPServerRequestDuration.Record(ctx, 0.05, metric.WithAttributes(
		semconv.HTTPRequestMethodKey.String("GET"),
		semconv.HTTPResponseStatusCodeKey.Int(200),
	))
	inst.HTTPClientRequestDuration.Record(ctx, 0.01, metric.WithAttributes(
		semconv.ServerAddressKey.String("api.example.com"),
	))

	srv := httptest.NewServer(p.Handler("/metrics"))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	text := string(body)

	for _, want := range []string{
		"http_server_request_duration_seconds",
		"http_server_active_requests",
		"http_client_request_duration_seconds",
		`http_request_method="GET"`,
		`http_response_status_code="200"`,
		`server_address="api.example.com"`,
		`otel_scope_schema_url="https://opentelemetry.io/schemas/1.26.0"`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
	// otel_scope_version must be populated (either a version, a VCS rev, or "(devel)").
	if !strings.Contains(text, `otel_scope_version="`) || strings.Contains(text, `otel_scope_version=""`) {
		t.Errorf("expected non-empty otel_scope_version on proxy series")
	}
	// Deleted series must NOT appear.
	for _, forbidden := range []string{
		"http_server_requests_total",
		"upstream_requests_total",
		`status_class=`,
		`outcome=`,
		`upstream_host=`,
	} {
		if strings.Contains(text, forbidden) {
			t.Errorf("unexpected legacy token %q in metrics output", forbidden)
		}
	}
}

func TestHandlerServes404OffPath(t *testing.T) {
	p, err := New(enabledCfg())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Shutdown(context.Background())
	srv := httptest.NewServer(p.Handler("/metrics"))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/other")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestShutdownIdempotent(t *testing.T) {
	p, err := New(enabledCfg())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()
	if err := p.Shutdown(ctx); err != nil {
		t.Fatalf("first Shutdown: %v", err)
	}
	if err := p.Shutdown(ctx); err != nil {
		t.Fatalf("second Shutdown: %v", err)
	}
}
