## ADDED Requirements

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
