## ADDED Requirements

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
