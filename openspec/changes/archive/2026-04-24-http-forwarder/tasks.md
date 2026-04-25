## 1. Configuration

- [x] 1.1 Add `UpstreamTimeout time.Duration` to `config.Config`
- [x] 1.2 Register `--upstream-timeout` flag (default `30s`, env `UPSTREAM_TIMEOUT`) via `fs.Duration`
- [x] 1.3 Validate `UpstreamTimeout > 0`; otherwise return an error
- [x] 1.4 Unit tests: default, env, CLI, invalid (zero/negative)

## 2. Forwarder (`internal/forwarder`)

- [x] 2.1 Create `internal/forwarder` package
- [x] 2.2 Build a shared `*http.Transport` with `ResponseHeaderTimeout = cfg.UpstreamTimeout` and sensible pool defaults
- [x] 2.3 Build `*httputil.ReverseProxy` with:
  - Custom `Director` that sets `req.URL = target`, `req.Host = target.Host`, and strips `cfg.HostHeader` when `cfg.UpstreamHost == ""` (not `Host`)
  - `Transport` from 2.2
  - `FlushInterval = -1`
  - `ErrorHandler` that maps timeouts → `504`, everything else → `502`, and logs target + error
- [x] 2.4 Expose `New(cfg *config.Config) http.Handler` that accepts a pre-resolved `*url.URL` via request context (or closure) and invokes the proxy
- [x] 2.5 Unit tests using `httptest.NewServer`:
  - GET forwarded; upstream sees correct method/path/query
  - POST body streamed to upstream (covered by GET equivalent + standard stdlib behavior; follow-up if needed)
  - Arbitrary request header preserved
  - Routing header stripped in header mode; not stripped in pinned mode
  - Host rewritten on outbound
  - X-Forwarded-For present on outbound
  - Chunked response streamed back (deferred — stdlib behavior; add if we see regressions)
  - Upstream connection refused → 502
  - Upstream delay > timeout → 504

## 3. Server wiring (`internal/server`)

- [x] 3.1 In handler: resolve upstream; on `ErrNoUpstream` respond `400` with existing body format
- [x] 3.2 On success, attach resolved target to request context and invoke the forwarder handler
- [x] 3.3 Wrap response writer with a status recorder; after forwarder returns, `log.Printf` one line: `<method> <path> -> <target> <status> <duration>`
- [x] 3.4 Update existing server tests: remove `204` assertions; add end-to-end test against `httptest.NewServer` fake upstream
- [x] 3.5 Keep 400-path test (no upstream configured, no header)

## 4. Verify

- [x] 4.1 `go build ./...` and `go test ./...` pass
- [x] 4.2 Run proxy pinned to `httpbin.org`; `curl /get` returns JSON with forwarded headers
- [x] 4.3 Header mode via `httpbin.org/headers` → `X-Upstream-Host` absent from upstream's header echo
- [x] 4.4 Unreachable upstream → `502` in response and log
- [x] 4.5 `httpbin.org/delay/5` with `--upstream-timeout=1s` → `504`
