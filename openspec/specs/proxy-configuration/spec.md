# proxy-configuration Specification

## Purpose
TBD - created by archiving change jwt-token-library. Update Purpose after archive.
## Requirements
### Requirement: Token signing algorithm flag

The proxy SHALL accept `--token-signing-algorithm` (env `TOKEN_SIGNING_ALGORITHM`), specifying the signing algorithm sent to AWS STS `GetWebIdentityToken`. Valid values are `RS256` and `ES384`. Default `RS256`. Any other value SHALL be rejected at startup.

#### Scenario: Default algorithm

- **WHEN** the proxy is started without `--token-signing-algorithm`
- **THEN** the effective signing algorithm is `RS256`

#### Scenario: ES384 via flag

- **WHEN** the proxy is started with `--token-signing-algorithm=ES384`
- **THEN** the effective signing algorithm is `ES384`

#### Scenario: Invalid value rejected

- **WHEN** the proxy is started with `--token-signing-algorithm=HS256`
- **THEN** the proxy exits with a non-zero status and an error naming the invalid value and listing the valid options

### Requirement: Token duration flag

The proxy SHALL accept `--token-duration` (env `TOKEN_DURATION`), a Go duration string bounding the requested token lifetime sent as `DurationSeconds`. The value SHALL be in the range `[60s, 3600s]` inclusive. Default `3600s`. Values outside the range SHALL be rejected at startup.

#### Scenario: Default duration

- **WHEN** the proxy is started without `--token-duration`
- **THEN** the effective token duration is `3600s`

#### Scenario: Custom duration via env

- **WHEN** `TOKEN_DURATION=300s` is set and no CLI flag is passed
- **THEN** the effective token duration is `300s`

#### Scenario: Below minimum rejected

- **WHEN** the proxy is started with `--token-duration=30s`
- **THEN** the proxy exits with a non-zero status and an error stating the value must be between `60s` and `3600s`

#### Scenario: Above maximum rejected

- **WHEN** the proxy is started with `--token-duration=2h`
- **THEN** the proxy exits with a non-zero status and an error stating the value must be between `60s` and `3600s`

### Requirement: Token refresh skew flag

The proxy SHALL accept `--token-refresh-skew` (env `TOKEN_REFRESH_SKEW`), a Go duration string defining how far in advance of expiry a cached token SHALL be considered stale. The value SHALL be strictly greater than `0` and strictly less than the effective `--token-duration`. Default `300s`. Violations SHALL be rejected at startup.

#### Scenario: Default skew

- **WHEN** the proxy is started without `--token-refresh-skew`
- **THEN** the effective refresh skew is `300s`

#### Scenario: Zero skew rejected

- **WHEN** the proxy is started with `--token-refresh-skew=0`
- **THEN** the proxy exits with a non-zero status and an error stating the value must be positive

#### Scenario: Skew not less than duration rejected

- **WHEN** the proxy is started with `--token-duration=60s` and `--token-refresh-skew=60s`
- **THEN** the proxy exits with a non-zero status and an error stating the skew must be strictly less than the token duration

### Requirement: CLI flag and env-var parity

Every configuration flag the proxy accepts SHALL be settable via both a CLI flag and an environment variable. The env-var name SHALL be derived mechanically from the flag name by uppercasing it and replacing dashes with underscores (e.g., `--upstream-host` → `UPSTREAM_HOST`).

#### Scenario: Env var used when flag omitted

- **WHEN** the proxy is started with no `--upstream-host` flag and `UPSTREAM_HOST=api.example.com` in the environment
- **THEN** the effective upstream host is `api.example.com`

#### Scenario: CLI flag takes precedence over env var

- **WHEN** the proxy is started with `--upstream-host=cli.example.com` and `UPSTREAM_HOST=env.example.com` in the environment
- **THEN** the effective upstream host is `cli.example.com`

### Requirement: Routing-related flags

The proxy SHALL expose the following flags (and corresponding env vars) at startup:

- `--listen-addr` (default `:8080`) — host:port the server listens on.
- `--upstream-host` — pinned upstream host; empty disables pinned mode.
- `--upstream-scheme` (default `https`) — scheme used when forwarding; accepts only `http` or `https`.
- `--host-header` (default `X-Upstream-Host`) — request header name read for the upstream host when unpinned.

#### Scenario: Default values apply when nothing is set

- **WHEN** the proxy is started with no flags and no environment overrides
- **THEN** the server listens on `:8080`, the scheme is `https`, and the header is `X-Upstream-Host`

#### Scenario: Invalid upstream scheme is rejected

- **WHEN** the proxy is started with `--upstream-scheme=ftp`
- **THEN** the proxy exits with a non-zero status and an error message identifying the invalid scheme

#### Scenario: Upstream host containing a scheme is rejected

- **WHEN** the proxy is started with `--upstream-host=https://api.example.com`
- **THEN** the proxy exits with a non-zero status and an error message directing the user to `--upstream-scheme`

### Requirement: Server starts from root command

The proxy SHALL start its HTTP server when invoked as the root command with no subcommand. Flags and environment variables defined by proxy-configuration SHALL apply to the root command.

#### Scenario: Invoking without a subcommand starts the server

- **WHEN** the user runs `aws-outbound-jwt-proxy --upstream-host=api.example.com`
- **THEN** the server starts listening on the configured address with the given upstream host

#### Scenario: No dedicated serve subcommand

- **WHEN** the user runs `aws-outbound-jwt-proxy serve`
- **THEN** cobra reports an unknown command error

### Requirement: TLS flags

The proxy SHALL accept `--tls-cert` and `--tls-key` flags (and the env vars `TLS_CERT` and `TLS_KEY`), each naming a path to a PEM-encoded file. Both default to empty.

#### Scenario: Both flags set enable TLS

- **WHEN** the proxy is started with `--tls-cert=/path/to/cert.pem --tls-key=/path/to/key.pem`
- **THEN** the server listens on HTTPS using the provided certificate and key

#### Scenario: Neither flag set uses plain HTTP

- **WHEN** the proxy is started without either TLS flag
- **THEN** the server listens on plain HTTP

#### Scenario: Partial TLS config is rejected

- **WHEN** the proxy is started with `--tls-cert` set but no `--tls-key` (or vice versa)
- **THEN** the proxy exits with a non-zero status and an error stating that both must be set together

### Requirement: Upstream timeout flag

The proxy SHALL accept `--upstream-timeout` (env `UPSTREAM_TIMEOUT`), a Go duration string (e.g. `30s`, `1m`), bounding how long the proxy waits for the upstream to send response headers. Default `30s`. Zero or negative values SHALL be rejected.

#### Scenario: Default timeout

- **WHEN** the proxy is started without `--upstream-timeout`
- **THEN** the effective upstream header timeout is `30s`

#### Scenario: Custom timeout via env

- **WHEN** `UPSTREAM_TIMEOUT=5s` is set and no CLI flag is passed
- **THEN** the effective upstream header timeout is `5s`

#### Scenario: Invalid value rejected

- **WHEN** the proxy is started with `--upstream-timeout=0`
- **THEN** the proxy exits with a non-zero status and an error stating the value must be positive

### Requirement: Log level flag

The proxy SHALL accept `--log-level` (env `LOG_LEVEL`) with values `debug`, `info`, `warn`, `error` (case-insensitive). Default `info`. Invalid values SHALL cause a startup error.

#### Scenario: Default level is info

- **WHEN** the proxy is started without `--log-level`
- **THEN** the effective log level is `info` and `debug` records are suppressed

#### Scenario: Invalid level rejected

- **WHEN** the proxy is started with `--log-level=trace`
- **THEN** the proxy exits with a non-zero status and an error naming the invalid value

### Requirement: Log format flag

The proxy SHALL accept `--log-format` (env `LOG_FORMAT`) with values `json` or `text`. Default `json`. Invalid values SHALL cause a startup error.

#### Scenario: Default format is JSON

- **WHEN** the proxy is started without `--log-format` and emits a log record
- **THEN** the record is written to stdout as a single JSON object with `time`, `level`, `msg`, and any structured attributes

#### Scenario: Text format selected

- **WHEN** the proxy is started with `--log-format=text`
- **THEN** records are written in stdlib `slog.TextHandler` format (logfmt-like key=value pairs)

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

### Requirement: Token audience flag (repeatable, optional)

The proxy SHALL accept `--token-audience` (env `TOKEN_AUDIENCE`), supplying the audience values sent to AWS STS `GetWebIdentityToken` as the `Audience` array. The flag is repeatable on the command line; the environment variable is a comma-separated list. The flag is OPTIONAL: if no value is supplied, the proxy SHALL start and select the `HostAudience` resolver so the audience is derived per-request from the outbound target host. When values ARE supplied, the proxy SHALL still reject any empty value and any value containing ASCII whitespace. There is no default audience string.

#### Scenario: Single audience via flag

- **WHEN** the proxy is started with `--token-audience=https://api.example.com`
- **THEN** the configured audience set is `["https://api.example.com"]`, the resolver is `StaticAudiences`, and every request uses that slice

#### Scenario: Multiple audiences via repeated flag

- **WHEN** the proxy is started with `--token-audience=https://a.example.com --token-audience=https://b.example.com`
- **THEN** the configured audience set is `["https://a.example.com", "https://b.example.com"]` and the resolver is `StaticAudiences`

#### Scenario: Multiple audiences via env

- **WHEN** `TOKEN_AUDIENCE=https://a.example.com,https://b.example.com` is set and no CLI flag is passed
- **THEN** the configured audience set is `["https://a.example.com", "https://b.example.com"]` and the resolver is `StaticAudiences`

#### Scenario: CLI flag precedence over env

- **WHEN** `TOKEN_AUDIENCE=https://env.example.com` is set AND `--token-audience=https://cli.example.com` is passed
- **THEN** the configured audience set is `["https://cli.example.com"]` (the env value is ignored, matching existing flag-vs-env precedence rules)

#### Scenario: Missing audience selects the host-derived resolver

- **WHEN** the proxy is started with no `--token-audience` flag and no `TOKEN_AUDIENCE` env var
- **THEN** the proxy starts successfully, the configured audience set is empty, and the constructed resolver is `HostAudience`

#### Scenario: Empty string audience rejected

- **WHEN** the proxy is started with `TOKEN_AUDIENCE=https://a.example.com,,https://b.example.com`
- **THEN** the proxy exits with a non-zero status and an error stating audience values must be non-empty

#### Scenario: Whitespace in audience rejected

- **WHEN** the proxy is started with `--token-audience="api example.com"`
- **THEN** the proxy exits with a non-zero status and an error stating audience values must not contain whitespace

