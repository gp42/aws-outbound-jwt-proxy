## ADDED Requirements

### Requirement: Forward inbound request to resolved upstream

The proxy SHALL forward each resolvable inbound request to the upstream URL returned by the resolver, preserving method, path, query, and body. The response body SHALL be streamed back to the client without buffering.

#### Scenario: GET forwarded and body streamed

- **WHEN** a client sends `GET /v1/items?limit=10` to the proxy and the upstream responds `200 OK` with a body
- **THEN** the upstream receives `GET /v1/items?limit=10` and the client receives the same status and body

#### Scenario: POST with body forwarded

- **WHEN** a client sends `POST /v1/items` with a JSON body
- **THEN** the upstream receives the same method, path, and body bytes

### Requirement: Preserve inbound headers, rewrite Host, strip routing header

The proxy SHALL forward all inbound request headers to the upstream, excluding standard hop-by-hop headers. The outbound `Host` SHALL be set to the resolved upstream host. When operating in header-driven routing mode, the configured routing header (e.g. `X-Upstream-Host`) SHALL be stripped from the outbound request.

#### Scenario: Arbitrary header preserved

- **WHEN** the client sets `X-Custom: value` on the request
- **THEN** the upstream receives `X-Custom: value`

#### Scenario: Routing header stripped in header mode

- **WHEN** the proxy runs without `--upstream-host`, the client sets `X-Upstream-Host: api.example.com` to route, and the request is forwarded
- **THEN** the upstream does not see `X-Upstream-Host`

#### Scenario: Host rewritten to target

- **WHEN** a request resolves to host `api.example.com`
- **THEN** the upstream receives `Host: api.example.com` (not the proxy's hostname)

### Requirement: X-Forwarded-For appended

The proxy SHALL append the client's IP to the `X-Forwarded-For` header on the outbound request (stdlib `httputil.ReverseProxy` default behavior).

#### Scenario: XFF header set

- **WHEN** a client with remote address `203.0.113.9` sends a request through the proxy
- **THEN** the upstream sees `X-Forwarded-For: 203.0.113.9` (or the prior XFF list with `203.0.113.9` appended)

### Requirement: Upstream failures mapped to gateway statuses

The proxy SHALL respond `502 Bad Gateway` when it cannot connect to or read the initial response from the upstream for reasons other than timeout. It SHALL respond `504 Gateway Timeout` when the upstream does not return response headers within the configured `--upstream-timeout`.

#### Scenario: Upstream refuses connection

- **WHEN** the upstream is unreachable (connection refused, DNS failure, TLS error)
- **THEN** the client receives `502 Bad Gateway`

#### Scenario: Upstream exceeds header timeout

- **WHEN** the upstream does not send response headers within `--upstream-timeout`
- **THEN** the client receives `504 Gateway Timeout`

### Requirement: Streaming support

The proxy SHALL flush response bytes to the client as they arrive from the upstream so that chunked/SSE responses reach clients without additional buffering, and it SHALL support HTTP `Upgrade` (including WebSocket) for bidirectional streaming.

#### Scenario: Chunked response reaches client incrementally

- **WHEN** the upstream sends a chunked response that pauses between chunks
- **THEN** the client receives each chunk as it is sent, without waiting for the whole body

### Requirement: Request log line

The proxy SHALL emit a single log line per completed request with method, path, resolved target URL, final response status, and duration.

#### Scenario: Successful request logged

- **WHEN** a request resolves, is forwarded, and completes with `200`
- **THEN** the proxy logs a line containing the method, path, resolved URL, `200`, and the elapsed time

