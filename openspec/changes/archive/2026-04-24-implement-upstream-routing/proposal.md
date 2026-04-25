## Why

To get an end-to-end skeleton running, we need the proxy to accept an HTTP request and determine which upstream it would be forwarded to. Implementing upstream routing first (with the rest stubbed) lets us validate the configuration surface and routing precedence rules without yet building token acquisition or request forwarding.

## What Changes

- Introduce a minimal HTTP server that accepts any inbound request and resolves the target upstream from configuration per the rules in `initial-proxy-architecture`:
  - Pinned upstream (`--upstream-host` + `--upstream-scheme`) wins when set.
  - Otherwise, read the upstream host from a configurable header (default `X-Upstream-Host`, overridable via `--host-header`).
- Introduce configuration loading from CLI flags with environment variable fallback (CLI wins when both are set) for the routing-relevant knobs: `--upstream-host`, `--upstream-scheme`, `--host-header`, and `--listen-addr` (host:port, e.g. `:8080`, which lets users override the listen port).
- On each request, print the resolved target (scheme + host + original path) to stdout as a debug line; respond with `204 No Content`.
- Return `400 Bad Request` when no upstream can be resolved (neither pinned config nor the header is present).
- Stub the rest of the architecture: no token acquisition, no forwarding, no TLS server mode yet — those arrive in later changes.
- Lay down the package layout under `internal/` (`internal/config`, `internal/router`, `internal/server`) and the `cmd/proxy` entrypoint.

## Capabilities

### New Capabilities
- `upstream-routing`: Resolve the target upstream URL from pinned configuration or a request header, with pinned configuration winning.
- `proxy-configuration`: Load runtime configuration from CLI flags with environment variable fallback, scoped in this change to the flags needed for routing.

### Modified Capabilities
<!-- None. No existing specs in openspec/specs/. -->

## Impact

- New code: `cmd/proxy/main.go`, `internal/config`, `internal/router`, `internal/server`.
- No new external dependencies (stdlib `net/http` + `flag` suffice).
- Establishes the CLI/env configuration contract that later changes (token acquisition, forwarding, TLS) will extend.
- Sets a deliberately stubbed response (`204` + stdout debug line) that later changes will replace with real forwarding.
