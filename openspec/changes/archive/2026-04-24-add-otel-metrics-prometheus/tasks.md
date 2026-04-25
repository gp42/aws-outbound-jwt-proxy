## 1. Configuration

- [x] 1.1 Add `MetricsEnabled bool`, `MetricsListenAddr string`, `MetricsPath string` to `config.Config`
- [x] 1.2 Register flags: `--metrics-enabled` (default `true`), `--metrics-listen-addr` (default `:9090`), `--metrics-path` (default `/metrics`)
- [x] 1.3 Extend env fallback for the new flags (already covered by `applyEnvFallback`, verify names resolve: `METRICS_ENABLED`, `METRICS_LISTEN_ADDR`, `METRICS_PATH`)
- [x] 1.4 Validate: `MetricsPath` must begin with `/`; `MetricsListenAddr` must be non-empty and must not equal `ListenAddr`
- [x] 1.5 Unit tests for defaults, env fallback, and all validation failures

## 2. Dependencies

- [x] 2.1 `go get go.opentelemetry.io/otel go.opentelemetry.io/otel/metric go.opentelemetry.io/otel/sdk/metric go.opentelemetry.io/otel/exporters/prometheus`
- [x] 2.2 `go get go.opentelemetry.io/contrib/instrumentation/runtime` (Go runtime metrics producer)
- [x] 2.3 `go mod tidy`; commit `go.mod` / `go.sum`

## 3. Metrics package (`internal/metrics`)

- [x] 3.1 Create `internal/metrics` package with a `Provider` struct wrapping `*sdkmetric.MeterProvider` + the Prometheus exporter's `http.Handler`
- [x] 3.2 Implement `New(cfg *config.Config) (*Provider, error)`: builds the Prometheus exporter, constructs the `MeterProvider`, registers runtime metrics, returns a no-op-safe `*Provider` when `cfg.MetricsEnabled == false`
- [x] 3.3 Implement `Instruments` struct exposing the concrete instruments: `HTTPRequestsTotal`, `HTTPRequestDuration`, `HTTPRequestsInFlight`, `UpstreamRequestsTotal`, `UpstreamRequestDuration`
- [x] 3.4 Implement `(*Provider).Instruments() (*Instruments, error)` — instrument construction once, reused by server + forwarder
- [x] 3.5 Implement `(*Provider).Handler() http.Handler` returning the Prometheus exposition handler (serves on the configured path; 404 elsewhere)
- [x] 3.6 Implement `(*Provider).Shutdown(ctx)` calling `MeterProvider.Shutdown`
- [x] 3.7 Provide a `NoopInstruments()` constructor for tests that do not exercise metrics
- [x] 3.8 Unit tests: (a) instruments are non-nil when enabled; (b) Shutdown is idempotent; (c) handler serves `200` with `text/plain` body containing expected metric family names; (d) runtime metrics present (`go_goroutines` or OTel-normalized equivalent)

## 4. Instrument the HTTP handler

- [x] 4.1 Add `instruments *metrics.Instruments` parameter to `server.New`
- [x] 4.2 In `handler(...)`: wrap the request with `instruments.HTTPRequestsInFlight.Add(+1)` / `Add(-1)` via `defer`
- [x] 4.3 After request completes, record: `HTTPRequestsTotal.Add(1, method, status_class)` and `HTTPRequestDuration.Record(seconds, method, status_class)`
- [x] 4.4 Derive `status_class` as `fmt.Sprintf("%dxx", rec.status/100)`; treat `rec.status == 0` as `"5xx"` defensively
- [x] 4.5 Unit test: fire synthetic requests through the handler with an in-memory meter provider (manual reader) and assert counter / histogram values

## 5. Instrument the upstream forwarder

- [x] 5.1 Add `instruments *metrics.Instruments` to `forwarder.New` (signature change)
- [x] 5.2 Capture start time around the upstream RoundTrip
- [x] 5.3 Classify outcome: `success` when no error and status < 500, `timeout` when `errors.Is(err, context.DeadlineExceeded)` or the timeout path, `error` otherwise
- [x] 5.4 Record `UpstreamRequestsTotal.Add(1, upstream_host, outcome)` and `UpstreamRequestDuration.Record(seconds, upstream_host)` — `upstream_host` SHALL be the resolved host (not a header echo)
- [x] 5.5 Unit test covering success, 5xx, and timeout paths using a fake RoundTripper and in-memory meter

## 6. Wire startup in `cmd/root.go`

- [x] 6.1 After `logging.Install`, call `metrics.New(cfg)`; on error exit non-zero with a clear message
- [x] 6.2 If `cfg.MetricsEnabled`: build an `http.Server{Addr: cfg.MetricsListenAddr, Handler: mux}` where `mux` routes `cfg.MetricsPath` to the provider handler and returns `404` for everything else; start it in a goroutine, surface bind errors via a buffered error channel before the proxy starts
- [x] 6.3 Plumb `(*metrics.Instruments)` into `server.New` and `forwarder.New`
- [x] 6.4 Install a signal handler (`SIGINT`, `SIGTERM`) that shuts down proxy server, then metrics server, then `provider.Shutdown`
- [x] 6.5 Update existing `RunE` return path so `http.ErrServerClosed` from either listener is treated as a clean exit

## 7. Documentation

- [x] 7.1 Extend the README configuration table with the three new flags and their env vars
- [x] 7.2 Add a short "Observability" section noting the separate listener, default port `:9090`, lack of auth, and recommendation to bind loopback / restrict at the network layer
- [x] 7.3 Document a `curl :9090/metrics` example and expected metric names

## 8. Verify

- [x] 8.1 `go build ./...` and `go test ./...` pass
- [x] 8.2 Run the proxy against a test upstream; `curl http://localhost:9090/metrics` shows all five instrument families plus `go_` runtime metrics
- [x] 8.3 Generate load; confirm `http_server_request_duration_seconds_bucket` populates across buckets and `http_server_requests_in_flight` rises under concurrency
- [x] 8.4 Start a second instance with `--listen-addr=:8080 --metrics-listen-addr=:8080`; confirm startup fails with a conflict error
- [x] 8.5 Start with `--metrics-enabled=false`; confirm `:9090` is free and no metrics are emitted
- [x] 8.6 Start with metrics port already bound; confirm the process exits non-zero with a message naming `--metrics-listen-addr`
- [x] 8.7 Send `SIGTERM`; confirm both listeners drain and the process exits `0`
