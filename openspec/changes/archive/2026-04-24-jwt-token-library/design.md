## Context

The proxy needs to attach AWS-issued JWTs to outbound requests, but the token-acquisition mechanics (STS call, caching, lifetime management, single-flight) are independent of the forwarder. Building them as a standalone library lets us unit-test the lifecycle in isolation and gives the forwarder (next change) a narrow, stable interface to consume.

AWS's `sts:GetWebIdentityToken` API issues OIDC-style JWTs: top-level `iss/sub/aud/exp/iat/jti`, with AWS-specific details (account, source region, `principal_tags`, `request_tags`, session context) nested under a single `https://sts.amazonaws.com/` claim. The role ARN travels in `sub`; identity attributes travel via `principal_tags` (auto-populated from role/session tags). The request parameters are `Audience` (array, 1–10 strings), `SigningAlgorithm` (`RS256` or `ES384`), `DurationSeconds` (60–3600, default 300), and optional `Tags` (up to 50, surfaced as `request_tags`). The API is regional only (no global endpoint) and feature-gated per account.

## Goals / Non-Goals

**Goals:**
- Single narrow interface: given an audience, return a currently-valid JWT string.
- Cache by audience, serve until within a configurable skew of `exp`, refresh synchronously on miss.
- Coalesce concurrent fetches for the same audience into one STS call.
- Pass AWS error classes through so operators can diagnose IAM policy denials vs. outages.
- Observable via existing OTel meter.

**Non-Goals:**
- Token injection into outbound requests (forwarder's job, next change).
- Per-request `Tags` / `request_tags`. Identity attributes already flow via `principal_tags`; we only need to extend the interface when a concrete use case requires per-call context.
- Background/proactive refresh. Synchronous-on-near-expiry keeps the first cut simple.
- Persisting tokens across restarts.
- Multi-audience tokens (`Audience` supports 1–10; we always send exactly one).
- Local JWT signature verification.

## Decisions

**Interface: `Source.Token(ctx, audience string) (string, error)`.**
Audience is a single string; we send it to STS as a 1-element `Audience` array. This keeps callers trivial, keeps the cache key trivial, and matches how the forwarder will use it (one audience per upstream). When per-request tags or multi-audience tokens become a real requirement, we widen to a `TokenRequest` struct and extend the cache key — but adding that surface speculatively would mean designing a cache-key algorithm for a case we may never hit.

**Cache implementation: `sync.Map` keyed by audience, value `*cacheEntry{token string, exp time.Time}`.**
Audiences are low-cardinality in practice (one per upstream service), so a `sync.Map` is ample. No eviction policy beyond overwrite-on-refresh. Alternative: `map + sync.RWMutex`. Rejected because `sync.Map` reads are lock-free and the hit path is the hot path.

**Single-flight: `golang.org/x/sync/singleflight`, keyed by audience.**
On cache miss or near-expiry, we call `sf.Do(audience, fetch)`. Concurrent callers for the same audience share one STS round-trip. Alternative: per-entry `sync.Mutex`. Rejected as more code for the same behavior.

**Expiry decision uses `refresh-skew`, not TTL.**
A cached token is usable iff `now + skew < exp`. We rely on the `Expiration` timestamp returned alongside the token rather than parsing the JWT payload — the SDK returns it as a structured field, and parsing the JWT to get `exp` would couple us to the token format unnecessarily. Clock skew between the proxy and AWS is handled by the `refresh-skew` buffer.

**Synchronous refresh only.**
The first request after expiry blocks for one STS call (~100ms typical). Given `refresh-skew` (default 60s) and a 900s duration, the hot path almost always hits the cache. Proactive refresh adds a goroutine lifecycle and failure-mode ambiguity (what if the background refresh fails — do we serve stale or fail the caller?). Defer until we have operational signal that synchronous latency is a problem.

**STS client: AWS SDK v2, built via `config.LoadDefaultConfig`.**
Region comes from the standard SDK chain (`AWS_REGION`, EC2 IMDS, ECS metadata, `~/.aws/config`). Credentials likewise. The proxy doesn't re-implement AWS credential resolution. If the operator misconfigures region, the SDK returns a clear error — we pass it through. Explicit region/profile flags are out of scope; add them when someone asks.

**Error passthrough.**
We wrap only to add context (audience, operation), not to reclassify. The forwarder (or operator reading logs) can distinguish:
- `AccessDenied` → IAM policy mismatch on `sts:IdentityTokenAudience`, `sts:DurationSeconds`, or `sts:SigningAlgorithm`.
- `OutboundWebIdentityFederationDisabled` → feature-flag issue at the account level.
- Network/timeout errors → STS unreachable.
Alternative: map to a typed enum inside the library. Rejected — the SDK's error types are already the canonical classification; re-encoding them loses fidelity.

**Metrics.**
Under `token.*` via the existing OTel meter:
- `token.fetch.count` (counter, attrs: `audience`, `result`=`ok|error`, `error_class`)
- `token.cache.hit.count` (counter, attrs: `audience`)
- `token.cache.miss.count` (counter, attrs: `audience`)
- `token.cached.audiences` (gauge — current distinct-audience count)

`audience` as a metric label is fine in this proxy because deployments pin a small, known set of upstreams. If we ever move to high-cardinality audiences we revisit.

**Config validation.**
- `--token-signing-algorithm`: `RS256` or `ES384` exact match, default `RS256`.
- `--token-duration`: `60s ≤ d ≤ 3600s`, default `900s`.
- `--token-refresh-skew`: `0 < skew < token-duration`, default `60s`.
The cross-field check (`skew < duration`) runs in config validation and fails startup on violation.

**Test seam: `tokentest.New(map[string]tokentest.Entry)` returns a fake `Source`.**
Lets downstream code (forwarder tests) exercise token flows without the AWS SDK. The STS-backed `Source` is tested against an interface-shaped STS client stub so we can simulate expiry, error classes, and latency deterministically.

## Risks / Trade-offs

- [Risk] Synchronous refresh spikes tail latency when many audiences expire near-simultaneously. → Mitigation: `refresh-skew` staggers in practice (audiences are fetched at different times originally); we add proactive refresh if this becomes visible. Single-flight prevents amplification within an audience.
- [Risk] IAM policy misconfiguration produces persistent `AccessDenied` on every fetch — the cache never populates and every client request re-calls STS. → Mitigation: negative caching for denial errors, with a short TTL (e.g. 5s). Deferred — keep v1 simple; flag as a known follow-up if we see the pattern.
- [Risk] Clock skew between the proxy and AWS could cause us to treat an expired token as valid. → Mitigation: `refresh-skew` default 60s covers realistic NTP drift. Operators can raise it if they run in time-challenged environments.
- [Trade-off] `audience` is a metric label. Acceptable because deployments pin the audience set; revisit if cardinality grows.
- [Trade-off] No proactive refresh. Acceptable at current traffic assumptions; first request after near-expiry pays one STS call.
- [Trade-off] No negative caching. A misconfigured IAM role causes per-request STS calls until fixed. Acceptable because `AccessDenied` should surface loudly in logs/metrics anyway, and negative caching would delay recovery once the policy is fixed.

## Open Questions

<!-- none -->

## Decided

- We do NOT parse the JWT payload to get `exp`; we use the SDK's `Expiration` response field.
- We send exactly one audience per STS call even though `Audience` accepts up to 10. Multi-audience tokens are rare and would complicate the cache key.
- Region/credential resolution follows the AWS SDK default chain — no proxy-specific flags.
