## Context

The proxy is a thin Go HTTP forwarder (`cmd/root.go` → `internal/server` → `internal/forwarder`) with structured slog output but no numeric telemetry. Operators running it in Kubernetes or ECS want the usual golden signals (rate, errors, duration, saturation) without standing up a log-to-metric pipeline. The repo is on Go 1.25 and already pulls in `cobra`/`pflag` for config; adding OTel + a Prometheus exporter is the smallest step that satisfies "metrics now" while keeping a path to OTLP push later.

## Goals / Non-Goals

**Goals:**
- Emit the four golden-signal metrics for inbound requests and upstream calls using OpenTelemetry instruments.
- Expose them via a Prometheus scrape endpoint on a **separate listener** so that scrape traffic, proxy traffic, and JWT injection are fully isolated.
- Configure the listener via CLI flags + env vars consistent with the existing config scheme.
- Keep label cardinality bounded by construction (no request path, bucketed status).
- Provide a single `MeterProvider` that future code (OTLP exporter, additional instrumentation) can plug into without rewiring call sites.

**Non-Goals:**
- OTel traces, OTel logs bridge, exemplars.
- OTLP push / gRPC exporter (add later behind a flag).
- Per-endpoint metrics, response-size histograms, request-body metrics.
- Authenticated metrics endpoint — operators are expected to restrict access at the network layer.
- TLS on the metrics listener (punt; can be layered later or fronted by a sidecar).

## Decisions

### OTel SDK + Prometheus exporter, not `prometheus/client_golang` directly

Use `go.opentelemetry.io/otel/sdk/metric` with `go.opentelemetry.io/otel/exporters/prometheus`. The exporter registers itself as a `prometheus.Collector` and is served via `promhttp.HandlerFor(prometheus.DefaultGatherer, ...)` (or the exporter's own gatherer).

**Why:** Instrument code once against the OTel API. Swapping Prometheus for OTLP later is a provider-wiring change, not a re-instrumentation. Using `client_golang` directly would lock us in and force a rewrite.

**Alternative considered:** `client_golang` only — simpler today but throws away the reason for picking OTel at all.

### Separate listener for `/metrics`

Run the Prometheus HTTP server on its own `http.Server` bound to `--metrics-listen-addr`, started in a goroutine from `cmd/root.go` before the proxy starts. Reject startup if both flags resolve to the same address.

**Why:**
- Scrapes bypass the proxy resolver/forwarder entirely — no risk of an operator accidentally proxying `/metrics` to an upstream, no JWT attached to scrape traffic.
- Lets operators bind the metrics port to localhost or an internal CIDR independently of the data port.
- Matches the shape of every other Go service doing this (kubernetes components, grafana agent, etc.).

**Alternative considered:** Mount `/metrics` on the proxy mux. Rejected: the proxy is a catch-all forwarder; there is no mux today, and introducing one to carve out `/metrics` means user requests to that path stop working — a surprising breaking change.

### Attribute discipline (low cardinality by construction)

- `method`: kept as-is (small closed set).
- `status_class`: `fmt.Sprintf("%dxx", status/100)` — 5 possible values.
- `upstream_host`: resolved host only (pinned flag or post-resolver value), never the raw header. Router already rejects unknown hosts upstream, so this is bounded by deployment topology.
- `path`: **never** an attribute.

**Why:** One slow-path regression from path cardinality blow-up costs more ops pain than every benefit we'd get from per-path metrics. If per-route metrics are needed later, they go behind an explicit opt-in flag with an allow-list.

### Histogram buckets: exporter defaults

Start with the OTel SDK's default explicit-bucket histogram boundaries (`[0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000]` ms for duration, converted to seconds). Do not add a flag for custom buckets yet — defer until someone actually needs it.

**Why:** No known SLO today; defaults are fine and changing buckets later is a non-breaking operational knob.

### Instrumentation call sites

- `internal/server.handler`: record `http_server_requests_total`, `http_server_request_duration_seconds`, `http_server_requests_in_flight` — same place the access log is emitted today.
- `internal/forwarder`: record `upstream_requests_total` and `upstream_request_duration_seconds` around the `RoundTrip` call. Classify `outcome` using the error sentinel (`os.IsTimeout` / `errors.Is(err, context.DeadlineExceeded)` → `timeout`, other non-nil errors → `error`, else `success`).

Pass a `metrics.Instruments` struct into the server and forwarder constructors so tests can inject a no-op or in-memory provider.

### Go runtime metrics

Use `go.opentelemetry.io/contrib/instrumentation/runtime` (or the newer `go.opentelemetry.io/otel/sdk/metric`-integrated runtime producer, whichever is current at implementation time). Register once at meter-provider construction.

### Startup / shutdown ordering

1. Build `MeterProvider` (only if `--metrics-enabled`).
2. Start metrics `http.Server` in a goroutine; surface bind errors synchronously via a `chan error` before the proxy listener starts.
3. Start proxy `http.Server`.
4. On `SIGTERM`: `Shutdown()` proxy server, then metrics server, then `MeterProvider.Shutdown()` (flushes pending exports — a no-op for Prometheus pull but correct for OTLP later).

## Risks / Trade-offs

- **[Risk]** The `otel/exporters/prometheus` module pulls in `prometheus/client_golang` transitively, inflating the binary. → **Mitigation:** accept it; the binary is still small and this is the standard Go metrics stack. Document in go.mod commit message.
- **[Risk]** Metrics endpoint leaks internal info (request rate, upstream hosts) if exposed to the public internet. → **Mitigation:** README warns operators to bind to loopback or a private CIDR; we also refuse to share a listener with the proxy so an operator has to actively expose the metrics port.
- **[Risk]** `status_class` hides specific error codes useful for alerting (e.g., 429 vs 500). → **Trade-off:** accepted to keep cardinality low; operators can still alert on `status_class="5xx"` rate and dig into logs for the specific code.
- **[Risk]** OTel SDK pre-1.0 for some sub-packages; metric semantic conventions shift. → **Mitigation:** pin minor versions in go.mod and keep metric names stable regardless of convention drift.
- **[Risk]** Startup failure if `:9090` is already in use (common on developer laptops). → **Mitigation:** clear error message; `--metrics-enabled=false` is the escape hatch.

## Migration Plan

No existing metrics to migrate. Rollout is additive:
1. Ship with `--metrics-enabled=true` default; operators who do not scrape simply ignore the new port.
2. Operators who already reserve `:9090` for something else set `--metrics-listen-addr` or disable.
3. Rollback: set `--metrics-enabled=false` or revert the binary — no persistent state to undo.
