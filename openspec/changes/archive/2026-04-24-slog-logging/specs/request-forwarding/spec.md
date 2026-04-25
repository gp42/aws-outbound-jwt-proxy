## MODIFIED Requirements

### Requirement: Request log line

The proxy SHALL emit a single structured log record per completed request at `info` level with at least these attributes: `method`, `path`, `target` (resolved upstream URL), `status`, `duration_ms` (integer milliseconds). Output SHALL follow the configured `--log-format`.

#### Scenario: Successful request logged with structured attrs

- **WHEN** a request resolves, is forwarded, and completes with `200`
- **THEN** one log record is emitted containing `method=GET`, `path`, `target`, `status=200`, and a positive `duration_ms` value

#### Scenario: Upstream failure logged

- **WHEN** the upstream is unreachable and the request returns `502`
- **THEN** two records are emitted: an `error`-level record identifying the target and underlying error, followed by the standard access record with `status=502`
