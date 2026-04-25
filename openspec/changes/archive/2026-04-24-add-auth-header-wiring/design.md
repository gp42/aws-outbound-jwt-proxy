## Context

`cmd/root.go` already constructs a `token.Source` but discards it (`_ = tokenSource`). The forwarder's `Director` sets the URL and strips the host header but never touches `Authorization`. Nothing else in the code path knows tokens exist. This change connects the two halves the project was built to connect.

Two design points drive the shape of the work:

1. `token.Source.Token` currently accepts a single `audience string`. The user wants to configure multiple audiences so one JWT can satisfy multiple external services. AWS STS `GetWebIdentityToken` already accepts an array — the library just flattened it.
2. The injection has to run per request (tokens expire, the cache rotates). That means touching `forwarder.Director`, which runs on the hot path. It also means the *audience* has to be resolved per request — even though today it's process-wide, tomorrow's per-host routing will need a request-scoped answer.

## Goals / Non-Goals

**Goals:**
- Every proxied request carries an AWS-issued JWT in `Authorization: Bearer …`.
- The token is always fresh enough to verify (delegated to `token.Source`'s existing skew logic).
- Token-fetch failures produce a clean `502` — the upstream is never called without a valid token.
- Multiple audiences are expressible as repeated flags and end up in the JWT's `aud` claim.
- The operator can tell, from metrics alone, whether the proxy is failing because of STS or because of the upstream.
- The design must accommodate a future per-host (or otherwise dynamic) audience resolver without rework to the forwarder injection point.

**Non-Goals:**
- Implementing dynamic per-host audience selection in this change. Only the seam is added; the implementation stays static.
- Retrying token fetches beyond what `token.Source` already does.
- Supporting bearer schemes other than raw JWT in `Authorization`.
- Pre-warming the token cache at startup.

## Decisions

### Inject in `forwarder.Director`, not as outer middleware

`httputil.ReverseProxy.Director` already rewrites `URL.Scheme`, `URL.Host`, and `Host`. Adding the `Authorization` header there keeps "outbound request shape" in one place. The alternative — an outer `http.Handler` that sets the header before `ServeHTTP` — works, but splits outbound header mutation across two packages.

Trade-off: `Director` has no error return. A token fetch failure inside `Director` can only signal by mutating the request (clear `URL.Host`, stash the error in request context). We then recognize the sentinel in `ErrorHandler`, write `502`, log the real cause at `error`, and record the metric. `ReverseProxy` will itself call `ErrorHandler` on an empty host, so the failure path is forced through it without us having to intercept the transport.

Considered alternative: do the fetch in `server.handler` before calling the forwarder. Cleaner error handling, but the forwarder no longer owns its outbound contract and test seams split across two packages. Rejected.

### `AudienceResolver` seam — static today, dynamic tomorrow

Introduce in `internal/token` (so the interface lives next to `Source`):

```go
// AudienceResolver returns the audience set for an outbound request.
// The returned slice is passed verbatim to Source.Token; normalization
// happens inside Source.
type AudienceResolver interface {
    Resolve(req *http.Request) ([]string, error)
}
```

Today's implementation is `token.StaticAudiences([]string)` which ignores the request and returns the configured slice. A future change adds (e.g.) `token.HostAudiences(map[string][]string)` or a regex-based resolver — neither requires changing `forwarder.New` or the `Director` logic.

`forwarder.New` takes `(cfg, instruments, tokenSource, audienceResolver)`. `Director` calls `audienceResolver.Resolve(req)` → `tokenSource.Token(ctx, audiences)` → set header. A resolver error follows the same `Director`-stash / `ErrorHandler`-502 path as a token error, with a distinct log message and a separate `token.result` attribute value (`resolver_error` vs `fetch_error` vs `ok`) so the dashboard can tell them apart without an additional instrument.

Why not pass a `func(*http.Request) ([]string, error)` directly? Named interface is easier to mock in tests, easier to document, and it's where we'll hang resolver-specific methods (e.g. an `AudiencesFor(host string)` helper) when we add the dynamic implementation. The function-type variant is also fine; interface is the lower-regret choice.

### Audience as `[]string`, normalized inside `token.Source`

`token.Source.Token(ctx, audiences []string)` — normalize by sort + dedupe, then use the joined form `aud1,aud2,...` as both cache key and metric `audience` attribute value. STS receives the normalized slice.

Why normalize inside `Source`:
- Per-host resolvers may produce the same audience set in different orders (e.g. from different map iterations). Normalization in one place prevents accidental double-minting.
- Metric cardinality stays bounded to actual distinct sets.
- Resolvers stay simple — they return whatever they have.

Breaking the existing `Token(ctx, string)` signature is acceptable: the only caller today is the about-to-land forwarder, and the `tokentest` fake updates in lockstep.

### Required, non-empty audience set at config validation

`--token-audience` repeatable; env `TOKEN_AUDIENCE` comma-separated. Config validation rejects: zero values, any empty string, any entry containing whitespace. Fails at boot.

When the dynamic resolver lands, `--token-audience` becomes "the default / fallback set" — but that's a future concern. Today it's the only source.

### `token.result` as an `http.client.*` attribute

`ok` / `fetch_error` / `resolver_error`. Attached to `http.client.request.duration` so a single panel answers "is the proxy failing on token acquisition, on audience resolution, or on the upstream?" No new instrument.

### Overwrite inbound `Authorization`

Replace any inbound `Authorization` header. The proxy's contract is "I attach the AWS identity." Forwarding a client-supplied bearer would contradict that. Log at `debug` when overwrite happens; `warn` would be noise in sidecar setups.

## Risks / Trade-offs

- **[Director has no error return]** → Stash error in context + clear `URL.Host`; `ErrorHandler` emits `502`. Add a unit test covering exactly that path so future refactors don't silently break it.
- **[Interface proliferation before a second implementation exists]** → One interface (`AudienceResolver`) with one implementation is a known smell. Accepted because the proposal explicitly commits to dynamic resolvers as "a later change" and the interface is tiny (one method). If six months pass with no second implementation, fold `StaticAudiences` back into `forwarder` and remove the interface.
- **[Breaking `token.Source.Token` signature]** → Only the in-tree forwarder and the `tokentest` fake call it. Updated in the same change. No downstream callers exist.
- **[Token fetch latency on first request per process]** → One STS call per normalized audience set, then cached. Documented in README.
- **[Dropping the inbound `Authorization` silently]** → `debug` log covers it.

## Migration Plan

Single-binary forward-only rollout.

1. Same commit: `token.Source.Token` signature change, `AudienceResolver` interface + `StaticAudiences`, `tokentest` fake update.
2. Same commit: config (`TokenAudiences`, validation), forwarder (takes source + resolver), server (threads both through), `cmd/root.go` (builds `StaticAudiences(cfg.TokenAudiences)` and passes to `server.New`).
3. Deployers MUST set `TOKEN_AUDIENCE` (or `--token-audience`) before deploying — the proxy fails to start otherwise. Release notes call this out as a breaking config change.
4. No data migration, no persistent state.

Rollback: revert the deploy. Nothing external is touched.

## Open Questions

- None blocking. The resolver interface's exact location (`internal/token` vs a new `internal/audience` package) can flip without changing callers; picking `internal/token` now keeps related abstractions together.
