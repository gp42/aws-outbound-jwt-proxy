## ADDED Requirements

### Requirement: Metrics enabled flag

The proxy SHALL accept `--metrics-enabled` (env `METRICS_ENABLED`) as a boolean. Default `true`. When `false`, no metrics listener is started and no instruments are registered.

#### Scenario: Default enables metrics

- **WHEN** the proxy is started without `--metrics-enabled`
- **THEN** metrics collection is active and the metrics listener binds

#### Scenario: Disabled skips listener

- **WHEN** the proxy is started with `--metrics-enabled=false`
- **THEN** no listener is bound on the metrics address and the process does not fail if that address is in use

### Requirement: Metrics listen address flag

The proxy SHALL accept `--metrics-listen-addr` (env `METRICS_LISTEN_ADDR`) as a `host:port` string. Default `:9090`. The value SHALL be validated using the same rules as `--listen-addr` (non-empty, parseable as a TCP address). The metrics listen address SHALL NOT equal the proxy `--listen-addr`; if they match the proxy SHALL refuse to start.

#### Scenario: Default binds :9090

- **WHEN** the proxy is started without `--metrics-listen-addr`
- **THEN** the metrics listener binds `:9090`

#### Scenario: Conflict with proxy listener rejected

- **WHEN** the proxy is started with `--listen-addr=:8080` and `--metrics-listen-addr=:8080`
- **THEN** startup fails with an error naming both flags

### Requirement: Metrics path flag

The proxy SHALL accept `--metrics-path` (env `METRICS_PATH`) as a string. Default `/metrics`. The value SHALL begin with a `/` and SHALL NOT be empty.

#### Scenario: Default path

- **WHEN** the proxy is started without `--metrics-path`
- **THEN** `GET /metrics` on the metrics listener serves the Prometheus exposition

#### Scenario: Custom path

- **WHEN** the proxy is started with `--metrics-path=/telemetry/prom`
- **THEN** `GET /telemetry/prom` serves the exposition and `GET /metrics` on the metrics listener returns `404`

#### Scenario: Invalid path rejected

- **WHEN** the proxy is started with `--metrics-path=metrics` (missing leading slash)
- **THEN** startup fails with an error identifying `--metrics-path`
