## Why

The proxy's reason for existing is to attach a short-lived AWS-issued JWT to outbound requests, but today nothing in the codebase talks to AWS STS. We need a self-contained library that can acquire, cache, and renew tokens from the AWS STS `GetWebIdentityToken` API so that the forwarder (in a follow-up change) can inject them without worrying about the acquisition lifecycle. Landing this as its own change keeps the token mechanics testable in isolation and unblocks a clean forwarder integration step.

## What Changes

- Add an `internal/token` package exposing a `Source` interface: `Token(ctx, audience string) (string, error)`.
- Implement an AWS STS–backed `Source` that calls `GetWebIdentityToken` to mint a JWT, passing the caller-supplied audience as the single-element `Audience` parameter and the configured signing algorithm and duration. No request tags in v1.
- Cache tokens in memory keyed by audience string. Serve a cached token until it is within a configurable skew (default `60s`) of expiry.
- Single-flight concurrent fetches per audience (via `golang.org/x/sync/singleflight`) so a thundering herd causes one STS call.
- Synchronous refresh on cache miss or near-expiry. Background/proactive refresh is deferred — keep the first version simple; add it once we see need.
- Pass AWS errors through with minimal wrapping so operators can distinguish `AccessDenied` (IAM policy mismatch on audience/duration/algorithm), `OutboundWebIdentityFederationDisabled` (feature not enabled for the account), and transport failures.
- Expose metrics via the existing OTel meter: fetch count, fetch error count (labeled by error class), cache hits/misses, and cached-audience count (gauge).
- Add configuration: `--token-signing-algorithm` (default `RS256`, accepts `RS256` or `ES384`), `--token-duration` (default `900s`, must be in `[60s, 3600s]`), `--token-refresh-skew` (default `60s`, must be `> 0` and `< token-duration`). AWS region and credentials come from the standard AWS SDK chain — no proxy-specific flags.
- Provide an in-memory fake `Source` in a `tokentest` subpackage for downstream tests.

Out of scope (explicit non-goals):
- Wiring the token into outbound requests — that is the follow-up forwarder change.
- Per-request tags / custom claims. IAM already supports restricting tags via `aws:RequestTag` and `aws:TagKeys`, and for identity attributes role tags land in `principal_tags` automatically. We revisit if a concrete use case needs per-call `request_tags`.
- Background / proactive refresh.
- Persisting tokens across restarts.
- Local signature verification — external services verify via AWS's JWKS.
- Per-audience overrides for algorithm/duration.

## Capabilities

### New Capabilities
- `token-acquisition`: Acquire, cache, and renew AWS-issued JWTs from `sts:GetWebIdentityToken`, keyed by audience, with single-flight semantics and expiry-aware caching.

### Modified Capabilities
- `proxy-configuration`: Adds `--token-signing-algorithm`, `--token-duration`, and `--token-refresh-skew` flags (env `TOKEN_SIGNING_ALGORITHM`, `TOKEN_DURATION`, `TOKEN_REFRESH_SKEW`).

## Impact

- Code: new `internal/token` package (with `tokentest` subpackage); `internal/config` gets three new fields; `cmd/root.go` wires them.
- Dependencies: adds `github.com/aws/aws-sdk-go-v2/config`, `github.com/aws/aws-sdk-go-v2/service/sts`, and `golang.org/x/sync` (for `singleflight`).
- Runtime: first request per audience incurs one STS round-trip; subsequent requests serve from cache until within `refresh-skew` of expiry. Regional STS endpoint only — the global endpoint does not support this API.
- IAM: the role the proxy runs as needs `sts:GetWebIdentityToken` allowed for the audiences it will request; this is deployment concern, not proxy code.
- Metrics: new instruments under the `token.*` namespace, scraped by the existing Prometheus endpoint.
- Tests: unit tests with a stubbed STS client cover cache hit, cache miss, expiry, single-flight coalescing, refresh on near-expiry, and error passthrough. No live AWS calls.
