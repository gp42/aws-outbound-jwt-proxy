## ADDED Requirements

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

The proxy SHALL record the following instruments for every inbound HTTP request it accepts, regardless of whether forwarding succeeds:

- `http_server_requests_total` (counter): total inbound requests, with attributes `method` and `status_class` (one of `2xx`, `3xx`, `4xx`, `5xx`).
- `http_server_request_duration_seconds` (histogram): wall-clock duration from handler entry to response completion, same attributes as the counter.
- `http_server_requests_in_flight` (up-down counter / gauge): currently executing handler invocations, no attributes.

Request path SHALL NOT be used as a metric attribute.

#### Scenario: Successful request increments counter and histogram

- **WHEN** a `GET` request is proxied and responds with `200`
- **THEN** `http_server_requests_total{method="GET",status_class="2xx"}` increments by 1 and `http_server_request_duration_seconds` records one observation with the same attributes

#### Scenario: In-flight gauge tracks concurrency

- **WHEN** a slow request is in progress
- **THEN** `http_server_requests_in_flight` reads `1` during the request and returns to its prior value after completion

#### Scenario: Path cardinality is bounded

- **WHEN** requests are made to `/a`, `/b`, and `/c/123`
- **THEN** the metrics output contains a single series per `(method, status_class)` combination and no series distinguishing `/a` from `/b` from `/c/123`

### Requirement: Upstream forwarding metrics

The proxy SHALL record the following instruments for every upstream call initiated by the forwarder:

- `upstream_requests_total` (counter): upstream attempts, with attributes `upstream_host` and `outcome` (one of `success`, `error`, `timeout`).
- `upstream_request_duration_seconds` (histogram): duration of the upstream round-trip as observed by the forwarder, with attribute `upstream_host`.

`upstream_host` SHALL be the resolved upstream host (pinned or header-derived), not a client-controlled alias.

#### Scenario: Upstream timeout recorded as timeout

- **WHEN** the upstream does not respond within `--upstream-timeout`
- **THEN** `upstream_requests_total{upstream_host="api.example.com",outcome="timeout"}` increments by 1

#### Scenario: Upstream host is the resolved host

- **WHEN** the proxy runs with `--upstream-host=api.example.com` and receives a request carrying `X-Upstream-Host: attacker.example.com`
- **THEN** any recorded `upstream_host` attribute is `api.example.com`

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
