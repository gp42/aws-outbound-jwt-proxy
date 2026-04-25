## Why

Today `--token-audience` is required at startup and the resulting slice is applied process-wide via `token.StaticAudiences`. That forces one proxy deployment per logical audience and leaves the shared-proxy (`X-Upstream-Host`-driven) mode awkward: every tenant must pick the same audience, or operators must run a fleet of near-identical sidecars. The `AudienceResolver` seam was introduced precisely to unlock this — it is time to ship a dynamic implementation so the audience follows the target host of each request.

## What Changes

- **BREAKING**: `--token-audience` (env `TOKEN_AUDIENCE`) becomes optional. When omitted, the proxy starts with a host-derived resolver instead of failing at boot. When set, behavior is unchanged (static resolver using the configured set).
- Introduce `token.HostAudience` — an `AudienceResolver` that returns a single-element audience slice derived from the outbound request's target host (the value set on `req.URL.Host` by the router, i.e. pinned host or the `--host-header` value). Port is stripped; host is lowercased.
- Wire the resolver selection in `cmd/root.go`: if `cfg.TokenAudiences` is non-empty, use `token.StaticAudiences`; otherwise use `token.HostAudience`.
- Config validation change: `TokenAudiences` is no longer required. All other audience-shape validations (no empty strings, no whitespace) still apply when values ARE provided.
- Resolver error path: if the target host is unavailable at resolve time (e.g. empty `URL.Host`), return an error so the existing `resolver_error` → `502` path in the forwarder handles it without an STS call. This can only occur if routing runs out of order; it is defensive.
- README updates: flag table marks `--token-audience` as optional with a short note on the fallback; a new "Dynamic audience" subsection under "How it works" explains the host-derived behavior and IAM implications.
- Metrics: no new instruments. The existing `token.cached.audiences` gauge already reports the number of cached audience sets — under `HostAudience` this is the count of distinct hosts currently cached and therefore the total cached-token count (one token per entry). The `audience` attribute on `token.*` metrics naturally carries the per-host value so operators can see which hosts are driving churn. Document the gauge explicitly in the README as the signal to watch under dynamic mode.
- Token cache: unchanged. The existing `sync.Map` keyed on normalized audience strings naturally produces one entry per distinct host under `HostAudience` — no code change required.

Non-goals:
- Cache eviction. The cache remains unbounded in this change. For shared-proxy deployments fronting an unbounded host set this grows without shrinking; operators are expected to watch `token.cached.audiences` and scope IAM to an expected host set. A follow-up change will add bounded eviction (LRU or TTL-based) once real-world cardinality is observable.
- Per-host audience overrides or mappings (e.g. `host → audience-set`). This change derives the audience from the target host verbatim; richer mapping is a later change.

## Capabilities

### New Capabilities
- `dynamic-audience-resolution`: Per-request audience derivation from the forwarded request's target host, selected automatically when `--token-audience` is not configured.

### Modified Capabilities
- `proxy-configuration`: `--token-audience` changes from required to optional; absence selects the dynamic resolver instead of failing validation.

## Impact

- Code: `internal/token` gains `HostAudience` + tests; `internal/config` relaxes the "≥1 audience" check and keeps per-value validation; `cmd/root.go` picks between `StaticAudiences` and `HostAudience` based on `cfg.TokenAudiences`.
- Runtime: one STS round-trip per distinct target host on first use; cache key cardinality is bounded by the set of hosts clients actually target. Shared-proxy deployments now need IAM that permits `sts:GetWebIdentityToken` for every host the proxy may forward to — this is a real operational change and must be called out in the README.
- Security: the audience now reflects client-supplied routing input (the `X-Upstream-Host` header in shared-proxy mode). This is acceptable because the resolver only *derives* the audience from the same host the request is being sent to — a client cannot cause the proxy to mint a token for one host and send it to another. Still, deployers should scope the IAM trust policy as narrowly as the expected host set.
- Tests: new `internal/token` tests for `HostAudience` (host normalization, port stripping, error on empty host); `internal/config` tests adjusted to accept zero audiences; `cmd/root.go` path exercised via existing integration coverage.
- Docs: README flag table + new "Dynamic audience" explainer; IAM guidance updated.
