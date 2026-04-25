## Why

The token-acquisition library is done and `cmd/root.go` builds a `token.Source` that is currently unused (the `_ = tokenSource` placeholder flags it as wired in a follow-up). The proxy still forwards requests without an `Authorization` header, which defeats its reason for existing. This change closes that loop: every forwarded request gets a fresh AWS-issued JWT attached as a bearer token.

## What Changes

- Inject `Authorization: Bearer <jwt>` on every outbound proxied request, using the `token.Source` already constructed in `cmd/root.go`.
- Add a repeatable `--token-audience` flag (env `TOKEN_AUDIENCE`, comma-separated when set via env). At least one value is required; startup fails if none are supplied. All values are passed together as the `Audience` list of a single `sts:GetWebIdentityToken` call, producing one JWT whose `aud` claim carries every configured audience.
- Broaden `token.Source` to key its cache on the full audience set: `Token(ctx, audiences []string) (string, error)`. Order and duplicates in the caller's slice are normalized (sorted, deduped) before both the STS call and the cache key — so `["a","b"]` and `["b","a"]` hit the same entry. The AWS call itself sends the normalized slice as `Audience`. Single-audience callers continue to work (pass a one-element slice).
- Extend `forwarder.New` to accept a `token.Source` and the configured audience slice; its `Director` resolves the token and sets the `Authorization` header. Any pre-existing `Authorization` on the inbound request is overwritten.
- On token acquisition failure, short-circuit the proxy: return `502 Bad Gateway` with a short error body, do not forward upstream, and record the server-side metric with that status. Do not leak the underlying AWS error to the client body (log it at `error`).
- Wire `tokenSource` and `cfg.TokenAudiences` into `server.New` → `forwarder.New` and drop the `_ = tokenSource` placeholder in `cmd/root.go`.
- Add a `token.result` (`ok` / `error`) attribute to the `http.client.*` instruments so a single dashboard pane shows the proxy's token-failure rate per upstream.

Non-goals:
- Per-request / per-host audience selection. The configured audience set applies process-wide (sidecar model). Header-driven or host-driven audience routing is a later change.
- Reading the audience from the inbound request.
- Stripping upstream `WWW-Authenticate` / mutating response auth headers.
- Retrying on token fetch errors beyond what `token.Source` already does internally.

## Capabilities

### New Capabilities
- `outbound-auth-header`: Attach an AWS-issued JWT as a bearer token on every forwarded request, sourced from `token.Source`, keyed by a configured non-empty audience set, and fail the request cleanly if the token cannot be obtained.

### Modified Capabilities
- `proxy-configuration`: Adds repeatable `--token-audience` (env `TOKEN_AUDIENCE`, comma-separated); at least one value required at startup.
- `token-acquisition`: `Source.Token` takes `[]string` audiences (normalized: sorted + deduped) instead of a single string; cache key and `token.cached.audiences` gauge are keyed on the normalized set. `audience` metric attribute becomes the normalized `aud1,aud2,...` joined form.

## Impact

- Code: `internal/token` — interface and cache key change; `internal/forwarder` gains a `token.Source` dependency and a small "attach bearer" step in its `Director`; `internal/server.New` forwards both through; `internal/config` adds `TokenAudiences []string` plus validation (≥1, no empty strings); `cmd/root.go` passes `tokenSource` and `cfg.TokenAudiences` through instead of discarding the source.
- Runtime: first request per process incurs one STS round-trip (populates the cache under the normalized key); steady-state is a sync map lookup. Token fetch failures translate to `502` and are visible via existing `http.server.*` metrics plus the new `token.result` attribute on `http.client.*`.
- IAM: deployers must grant the proxy's role `sts:GetWebIdentityToken` allowed for the configured audience set. First change where the flag materially affects outbound traffic.
- Tests: `internal/token` tests updated for the slice API and normalization; `internal/forwarder` tests gain a `tokentest.Source` stub with cases for "token ok → header attached", "token error → 502, no upstream call", and "inbound Authorization is replaced"; `internal/config` tests cover "missing audience fails" and env-var comma-split.
- Docs: README's config table gains `--token-audience`; the "How it works" section no longer needs the "in a follow-up" caveat.
