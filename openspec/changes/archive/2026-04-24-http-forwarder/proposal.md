## Why

The proxy currently resolves the upstream target and responds `204` without forwarding anything. To make the proxy actually useful (and to unblock the token-acquisition change, which needs somewhere to inject the JWT), we need to replace the stub with a real HTTP forwarder that proxies the inbound request to the resolved upstream and streams the response back.

## What Changes

- Replace the stub `204` handler with a streaming reverse proxy built on `net/http/httputil.ReverseProxy`.
- Forward all inbound request headers to the upstream as-is, except standard hop-by-hop headers (which `ReverseProxy` strips by default) and the configured `--host-header` when running in header-driven mode (otherwise the upstream would see the proxy-routing header).
- Rewrite the outbound `Host` to the resolved target host.
- Stream request and response bodies in both directions (no buffering), preserving support for chunked transfers and HTTP upgrades (WebSocket included, which `httputil.ReverseProxy` handles natively).
- Return `502 Bad Gateway` on upstream dial/TLS failure; return `504 Gateway Timeout` on upstream read timeout (using a configurable `--upstream-timeout`, default `30s`).
- Add `--upstream-timeout` flag + env var for the overall upstream response timeout.
- Keep stdout request logging (one line per request: method, path, resolved target, status, duration).

## Capabilities

### New Capabilities
- `request-forwarding`: Proxy inbound HTTP requests to the resolved upstream, streaming bodies and preserving headers, with well-defined behavior on upstream failures.

### Modified Capabilities
- `upstream-routing`: Removes the "stub forwarding behavior" requirement (the `204 No Content` response). The resolver itself is unchanged.
- `proxy-configuration`: Adds `--upstream-timeout` flag + env var.

## Impact

- Code: new `internal/forwarder` package; `internal/server` handler delegates to it instead of writing `204`; `internal/config` gets `UpstreamTimeout`.
- No new third-party dependencies (stdlib `net/http/httputil` is used).
- Out of scope: JWT injection (that's `token-acquisition` / next change), request body size limits (defer until we see a reason), retries, circuit breakers, response transformation, TLS verification toggles for the upstream.
- Existing tests for the stub `204` path will be replaced by forwarding tests using `httptest.NewServer` as a fake upstream.
