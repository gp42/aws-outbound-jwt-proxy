# metrics-observability Specification

## Purpose
TBD - created by archiving change add-otel-metrics-prometheus. Update Purpose after archive.
## Requirements
### Requirement: Prometheus scrape endpoint

The proxy SHALL expose an HTTP endpoint that serves OpenTelemetry-produced metrics in the Prometheus text exposition format. The endpoint SHALL be served from a listener that is distinct from the proxy's forwarding listener so that scrapes do not compete with request traffic, do not inherit proxy TLS, and are not subject to JWT injection.

#### Scenario: Metrics endpoint served on dedicated port

- **WHEN** the proxy starts with defaults
- **THEN** an HTTP listener is bound on `:9090` and `GET /metrics` returns `200` with `Content-Type: text/plain; version=0.0.4` (or the exporter's current content-type) containing Prometheus-formatted metric families

#### Scenario: Metrics endpoint disabled

- **WHEN** the proxy starts with `--metrics-enabled=false`
- **THEN** no metrics listener is bound and no metrics are recorded

#### Scenario: Metrics listener isolated from proxy listener

- **WHEN** the proxy starts with `--listen-addr=:8080` and `--metrics-listen-addr=:9090`
- **THEN** `GET /metrics` on `:8080` is treated as a normal proxy request (not served locally), and `GET /anything-else` on `:9090` returns `404`

### Requirement: Request-level HTTP metrics

The proxy SHALL record the following OpenTelemetry instruments for every inbound HTTP request it accepts, regardless of whether forwarding succeeds:

- `http.server.request.duration` (float64 histogram, unit `s`): wall-clock duration from handler entry to response completion.
- `http.server.active_requests` (int64 up-down counter): currently executing handler invocations.

The duration histogram SHALL carry these attributes:
- `http.request.method`: the request method, uppercased, restricted to the nine canonical HTTP methods (`GET`, `HEAD`, `POST`, `PUT`, `DELETE`, `CONNECT`, `OPTIONS`, `TRACE`, `PATCH`). Any other method SHALL be replaced with the literal string `"_OTHER"`.
- `http.response.status_code`: the integer response status rendered as a string (`"200"`, `"404"`, `"502"`, ...).

`http.server.active_requests` SHALL carry no attributes. Request path SHALL NOT be used as a metric attribute on any instrument.

No separate request counter is emitted; request counts are read from the histogram's `_count` series.

#### Scenario: Successful request recorded with semconv attributes

- **WHEN** a `GET` request is proxied and responds with `200`
- **THEN** `http.server.request.duration` records one observation with attributes `http.request.method="GET"` and `http.response.status_code="200"` and the Prometheus scrape contains `http_server_request_duration_seconds_count{http_request_method="GET",http_response_status_code="200",...} 1`

#### Scenario: Unknown method bucketed to _OTHER

- **WHEN** a request is made with method `FOO`
- **THEN** the recorded `http.request.method` attribute is `"_OTHER"` and `http_request_method="FOO"` never appears in the scrape

#### Scenario: Full status code emitted (no class bucketing)

- **WHEN** requests complete with statuses `200`, `404`, and `502`
- **THEN** three series are emitted distinguished by `http.response.status_code="200"`, `="404"`, and `="502"`; no attribute named `status_class` is emitted

#### Scenario: In-flight gauge tracks concurrency

- **WHEN** a slow request is in progress
- **THEN** `http.server.active_requests` reads `1` during the request and returns to its prior value after completion

#### Scenario: Path cardinality is bounded

- **WHEN** requests are made to `/a`, `/b`, and `/c/123`
- **THEN** the metrics output contains a single series per `(http.request.method, http.response.status_code)` combination and no series distinguishing `/a` from `/b` from `/c/123`

### Requirement: Upstream forwarding metrics

The proxy SHALL record the following OpenTelemetry instrument for every upstream call initiated by the forwarder:

- `http.client.request.duration` (float64 histogram, unit `s`): duration of the upstream round-trip as observed by the forwarder.

The histogram SHALL carry these attributes:
- `server.address`: the resolved upstream host (pinned flag or post-resolver value), without port.
- `server.port`: the integer TCP port used for the upstream request, as a string. OMITTED when the port is the scheme default (`80` for `http`, `443` for `https`) and was not explicitly specified.
- `error.type`: present ONLY when the upstream call did not complete with a valid response. Values SHALL be drawn from the closed set: `"timeout"` (deadline exceeded or any `net.Error.Timeout()`), `"connection_refused"` (TCP connection refused), `"unknown"` (any other transport-level error). A 5xx response received from the upstream without a transport error SHALL NOT set `error.type`.

No separate upstream counter is emitted; upstream call counts are read from the histogram's `_count` series.

#### Scenario: Successful upstream call records no error.type

- **WHEN** the upstream responds `200`
- **THEN** `http.client.request.duration` records an observation with `server.address`, optionally `server.port`, and NO `error.type` attribute

#### Scenario: Upstream timeout recorded as timeout

- **WHEN** the upstream does not respond within `--upstream-timeout`
- **THEN** the recorded observation carries `error.type="timeout"`

#### Scenario: Upstream connection refused

- **WHEN** the upstream TCP port is closed
- **THEN** the recorded observation carries `error.type="connection_refused"`

#### Scenario: Upstream 5xx does not set error.type

- **WHEN** the upstream responds `500`
- **THEN** the recorded observation carries `http.response.status_code`-equivalent information on the server metric path and NO `error.type` on the client metric — the 5xx is carried by the series' status attribute on the inbound side, not by flagging the upstream as errored

#### Scenario: Default port omitted from attribute set

- **WHEN** the upstream is resolved as `api.example.com` (implicit `443` on `https`)
- **THEN** the recorded observation carries `server.address="api.example.com"` and NO `server.port` attribute

#### Scenario: Upstream host is the resolved host

- **WHEN** the proxy runs with `--upstream-host=api.example.com` and receives a request carrying `X-Upstream-Host: attacker.example.com`
- **THEN** any recorded `server.address` attribute is `api.example.com`

### Requirement: Go runtime metrics

The proxy SHALL register the standard Go runtime metrics (at minimum: goroutine count, GC pause, heap allocation, and process uptime) with the same meter provider so they appear in the `/metrics` output.

#### Scenario: Runtime metrics are exposed

- **WHEN** `GET /metrics` is scraped
- **THEN** the response contains series whose names begin with `go_` (or the OTel-normalized equivalents) covering goroutines, memory, and GC

### Requirement: Metrics lifecycle

The proxy SHALL start the metrics listener before the proxy listener begins accepting traffic and SHALL shut the metrics listener down cleanly on process termination. A failure to bind the metrics listener SHALL prevent the proxy from starting.

#### Scenario: Metrics bind failure aborts startup

- **WHEN** the metrics port is already in use and the proxy is started
- **THEN** the process exits with a non-zero status and an error identifying the metrics listen address

#### Scenario: Clean shutdown flushes

- **WHEN** the process receives `SIGTERM`
- **THEN** both listeners stop accepting new connections and the meter provider is shut down before the process exits

### Requirement: Instrumentation scope metadata

The meter the proxy uses to register instruments SHALL declare:

- an instrumentation name identifying the proxy module;
- an instrumentation version resolved at runtime from `runtime/debug.ReadBuildInfo()`: the module version when available, otherwise the VCS revision build setting (truncated to a short prefix) when available, otherwise the literal `"(devel)"`;
- a schema URL matching the OpenTelemetry semantic convention version the proxy's attribute names follow.

#### Scenario: Version populated from build info in released binary

- **WHEN** the proxy is run from a binary built with a module cache resolving a tagged version
- **THEN** the Prometheus scrape shows non-empty `otel_scope_version="vX.Y.Z"` on every series

#### Scenario: Version falls back to VCS revision in dev

- **WHEN** the proxy is run with `go run` or a `go build` without a tagged version but with `-buildvcs=true` default
- **THEN** `otel_scope_version` carries the short VCS revision (at most 12 hex chars)

#### Scenario: Schema URL populated

- **WHEN** the scrape is read
- **THEN** every series carries `otel_scope_schema_url="https://opentelemetry.io/schemas/<version>"` matching the semconv version the proxy's attribute names follow

