package metrics

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gp42/aws-outbound-jwt-proxy/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const meterName = "github.com/gp42/aws-outbound-jwt-proxy"

// Provider wraps the OTel MeterProvider and the Prometheus scrape handler.
// When metrics are disabled, Provider still returns a usable (no-op) set of
// instruments so call sites don't need nil checks.
type Provider struct {
	enabled  bool
	mp       metric.MeterProvider
	shutdown func(context.Context) error
	registry *prometheus.Registry
}

// New constructs a Provider. When cfg.MetricsEnabled is false, the returned
// Provider hands out no-op instruments and has no HTTP handler.
func New(cfg *config.Config) (*Provider, error) {
	if !cfg.MetricsEnabled {
		return &Provider{enabled: false, mp: noop.NewMeterProvider()}, nil
	}

	reg := prometheus.NewRegistry()
	exporter, err := otelprom.New(otelprom.WithRegisterer(reg))
	if err != nil {
		return nil, fmt.Errorf("build prometheus exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))

	if err := otelruntime.Start(otelruntime.WithMeterProvider(mp)); err != nil {
		return nil, fmt.Errorf("start runtime metrics: %w", err)
	}

	return &Provider{
		enabled:  true,
		mp:       mp,
		shutdown: mp.Shutdown,
		registry: reg,
	}, nil
}

func (p *Provider) Enabled() bool { return p.enabled }

// Meter returns a Meter from the underlying provider. Packages that own their
// own instruments (e.g. the token cache) use this to register them.
func (p *Provider) Meter() metric.Meter {
	return p.mp.Meter(
		meterName,
		metric.WithInstrumentationVersion(moduleVersion()),
	)
}

// Handler returns an http.Handler that serves the Prometheus exposition at
// the given path and returns 404 for every other path.
func (p *Provider) Handler(path string) http.Handler {
	if !p.enabled {
		return http.NotFoundHandler()
	}
	promHandler := promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{Registry: p.registry})
	mux := http.NewServeMux()
	mux.Handle(path, promHandler)
	return mux
}

// Shutdown flushes and releases the meter provider. Safe to call when
// disabled and safe to call more than once.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.shutdown == nil {
		return nil
	}
	fn := p.shutdown
	p.shutdown = nil
	return fn(ctx)
}
