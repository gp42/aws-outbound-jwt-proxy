## Context

The existing handler resolves the upstream and writes `204`. The forwarder replaces that final step: given the resolved target URL from `router.Resolver`, proxy the request upstream and stream the response back. Go's stdlib `net/http/httputil.ReverseProxy` does the heavy lifting (streaming, hop-by-hop header handling, `101 Switching Protocols` for WebSockets), so this change is mostly about wiring and policy.

## Goals / Non-Goals

**Goals:**
- Forward every inbound request to the resolved upstream with all headers preserved (minus hop-by-hop).
- Stream bodies both directions. No request buffering; no response buffering.
- WebSocket pass-through works for free via `httputil.ReverseProxy`'s upgrade handling.
- Distinguish upstream connect failures (`502`) from upstream read timeouts (`504`) in responses and logs.
- One configurable timeout knob (`--upstream-timeout`) for the whole upstream exchange.

**Non-Goals:**
- JWT injection (separate change).
- Request/response body transformation or inspection.
- Retries, circuit breakers, rate limiting.
- Per-upstream configuration (one timeout, one transport for all targets).
- TLS verification overrides (`InsecureSkipVerify` remains off).
- Graceful shutdown (tracked as a separate change).

## Decisions

**Build on `net/http/httputil.ReverseProxy`.**
It already handles streaming, hop-by-hop headers, `Host` rewriting via `Director`, WebSocket upgrade, and `X-Forwarded-For`. Writing our own would re-implement all of that and get it subtly wrong. The things we customize:

- `Director`: sets `req.URL` to the resolved target; strips the configured `--host-header` when in header-driven mode so the upstream never sees our routing header; sets `req.Host` to the target host.
- `Transport`: a single shared `*http.Transport` with sensible defaults (`IdleConnTimeout`, `ResponseHeaderTimeout` = `UpstreamTimeout`).
- `ErrorHandler`: maps `context.DeadlineExceeded` / `ResponseHeaderTimeout` to `504`, everything else to `502`. Log the target + error regardless.
- `FlushInterval = -1`: flush immediately — important for SSE/streaming responses.

**One `Resolver` call per request, inside `Director`.**
The `Director` has access to `req`, so we resolve the target there. On resolve failure we set a sentinel on the request context and let `ErrorHandler` respond with `400` (preserving the existing error surface from the unpinned-no-header case). Alternative: resolve in the top-level `http.Handler` and short-circuit before invoking `ReverseProxy`. Rejected — duplicates the "is this resolvable?" check and makes the handler branch-heavy.

Actually, simpler: resolve *before* `ReverseProxy` in the outer handler, short-circuit `400` if `ErrNoUpstream`, and only invoke the proxy when we have a target. `ReverseProxy`'s `Director` then receives a pre-resolved URL via request context or a closure. This keeps `Director` pure-transform and concentrates error handling in one place.

**Strip the configured `--host-header` from the outbound request when in header-driven mode.**
Otherwise the upstream sees `X-Upstream-Host: ...` (or whatever the user configured), which at best is noise and at worst leaks proxy internals. In pinned mode the header isn't consulted, so stripping is conditional on `cfg.UpstreamHost == ""`. If the configured header is `Host`, no strip needed (we rewrote `Host` anyway).

**Timeouts: one knob, applied via `Transport.ResponseHeaderTimeout`.**
`ResponseHeaderTimeout` bounds how long we wait for the upstream to start responding. We intentionally do *not* set `http.Client.Timeout`-style whole-request deadlines, because those break long-lived streams (SSE, WebSocket). For the response body, the upstream controls pacing — the proxy just shuffles bytes.

Alternative: use `req.Context()` with `context.WithTimeout`. Rejected for the same reason — cancels streamed responses.

**`502` vs `504` classification.**
In `ErrorHandler`, inspect the error: `net.Error` with `Timeout() == true`, or wraps `context.DeadlineExceeded` → `504`. Everything else (connection refused, TLS handshake failure, DNS failure, upstream closed mid-response) → `502`. Log the target URL and error class in both cases.

**Access log: single line per request.**
Format: `method path -> scheme://host status duration`. Printed via `log.Printf` on response completion. We intercept by wrapping `ReverseProxy.ServeHTTP` with a `statusRecorder` ResponseWriter. Avoids introducing a logging framework for this change.

## Risks / Trade-offs

- [Risk] Forwarding `Authorization` (or cookies, API keys) that the client sent would leak credentials to the upstream. → Acceptable and in fact necessary: the proxy's purpose is to preserve client intent. The *next* change (token injection) will add an `Authorization` header set by us, which replaces any inbound `Authorization`. Until then, treat all inbound headers as forwarded.
- [Risk] Unbounded request body size lets a client tie up upstream resources. → Out of scope for this change; track as future work. Stdlib's `ReverseProxy` streams, so the proxy itself stays memory-stable.
- [Risk] `Transport` connection pool leaks across unrelated upstreams (in header mode, each request may target a different host). → `http.Transport` pools per host:port by default, which is correct. No per-target transport needed.
- [Trade-off] No retries: first-attempt failures surface immediately as 502/504. Simpler and predictable; retry policy can land later when we understand the failure modes operationally.

## Open Questions

<!-- none -->

## Decided

- `X-Forwarded-For` is left on (default `ReverseProxy` behavior). Upstreams often use it for logging / rate limiting, and preserving it matches what a well-behaved reverse proxy is expected to do.
