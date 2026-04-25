package metrics

import (
	"fmt"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Instruments holds the instrument handles used throughout the proxy. All
// fields are always non-nil; when metrics are disabled they point at no-op
// implementations so callers don't need nil checks.
//
// Names and attribute vocabulary follow OpenTelemetry HTTP semantic
// conventions. Request/response counts are read from each histogram's
// automatic _count series; no standalone *_total counter is emitted.
type Instruments struct {
	HTTPServerRequestDuration metric.Float64Histogram
	HTTPServerActiveRequests  metric.Int64UpDownCounter
	HTTPClientRequestDuration metric.Float64Histogram
}

// Instruments constructs (once) the concrete instrument handles backed by this
// provider's meter. Call this at startup and share the returned *Instruments.
func (p *Provider) Instruments() (*Instruments, error) {
	meter := p.mp.Meter(
		meterName,
		metric.WithInstrumentationVersion(moduleVersion()),
		metric.WithSchemaURL(semconv.SchemaURL),
	)

	serverDur, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("Duration of inbound HTTP requests."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("http.server.request.duration: %w", err)
	}

	active, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of inbound HTTP requests currently being served."),
	)
	if err != nil {
		return nil, fmt.Errorf("http.server.active_requests: %w", err)
	}

	clientDur, err := meter.Float64Histogram(
		"http.client.request.duration",
		metric.WithDescription("Duration of outbound HTTP requests to upstreams (including STS)."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("http.client.request.duration: %w", err)
	}

	return &Instruments{
		HTTPServerRequestDuration: serverDur,
		HTTPServerActiveRequests:  active,
		HTTPClientRequestDuration: clientDur,
	}, nil
}

// NoopInstruments returns a set of no-op instruments usable by tests and
// by callers that don't want to wire a provider.
func NoopInstruments() *Instruments {
	meter := noop.NewMeterProvider().Meter(meterName)
	serverDur, _ := meter.Float64Histogram("noop")
	active, _ := meter.Int64UpDownCounter("noop")
	clientDur, _ := meter.Float64Histogram("noop")
	return &Instruments{
		HTTPServerRequestDuration: serverDur,
		HTTPServerActiveRequests:  active,
		HTTPClientRequestDuration: clientDur,
	}
}
