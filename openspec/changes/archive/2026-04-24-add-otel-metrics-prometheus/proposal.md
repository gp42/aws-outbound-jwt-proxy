## Why

The proxy currently emits structured access logs but exposes no numeric telemetry — no way to answer "what's my p95 latency?", "how many 5xx per minute?", or "is the upstream timing out?" without shipping logs to a separate aggregator and re-aggregating them. Emitting metrics via OpenTelemetry lets us instrument once and choose the backend at deploy time; starting with a Prometheus scrape endpoint gives us a zero-infrastructure default (scrape from kube-prometheus, Grafana Agent, Datadog Agent, etc.) while leaving the door open for an OTLP exporter later.

## What Changes

- Instrument the HTTP handler with OpenTelemetry metrics covering request volume, latency, status-class counts, and in-flight gauge.
- Instrument the upstream forwarder with counters for upstream errors and latency histograms keyed by upstream host.
- Wire an OTel `MeterProvider` at startup backed by the Prometheus exporter (`go.opentelemetry.io/otel/exporters/prometheus`); register Go runtime metrics (goroutines, GC, memory) so operators get a baseline.
- Serve the Prometheus scrape endpoint on a **separate listener** so that application traffic and telemetry are isolated (different port, no TLS by default, no JWT injection).
- Add CLI flags / env vars (see proxy-configuration delta):
  - `--metrics-enabled` / `METRICS_ENABLED` (default `true`)
  - `--metrics-listen-addr` / `METRICS_LISTEN_ADDR` (default `:9090`)
  - `--metrics-path` / `METRICS_PATH` (default `/metrics`)
- Emit metrics attributes cautiously to avoid cardinality blow-ups: status code bucketed into `2xx/3xx/4xx/5xx`, method kept as-is, upstream host kept as-is, **request path is NOT an attribute**.

## Capabilities

### New Capabilities
- `metrics-observability`: Defines the OTel-backed metrics the proxy emits, the Prometheus scrape endpoint contract, and the attribute / cardinality rules.

### Modified Capabilities
- `proxy-configuration`: Adds `--metrics-enabled`, `--metrics-listen-addr`, `--metrics-path` flags and their env-var fallbacks; defines validation rules.

## Impact

- Code: new `internal/metrics` package (meter provider setup, instrument registration, Prometheus handler); instrumentation hooks in `internal/server` and `internal/forwarder`; wiring in `cmd/root.go` to start the second listener alongside the proxy listener and shut it down cleanly.
- Dependencies: adds `go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/metric`, `go.opentelemetry.io/otel/sdk/metric`, `go.opentelemetry.io/otel/exporters/prometheus`, and the transitive `github.com/prometheus/client_golang` required by that exporter. No direct `prometheus/client_golang` instrumentation — all metrics flow through OTel.
- Operations: operators must expose / scrape an additional port (`:9090` by default). The metrics port is HTTP-only and carries no auth; operators are expected to keep it off the public network (bind to localhost, NetworkPolicy, or similar) — documented in README.
- Out of scope: OTel traces, OTel logs bridge, OTLP push exporter, histogram bucket customization, request-body / upstream-body size metrics, per-path labels. These are follow-ups.
