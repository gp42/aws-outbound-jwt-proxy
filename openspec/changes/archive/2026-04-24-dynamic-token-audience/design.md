## Context

The `add-auth-header-wiring` change landed the `token.AudienceResolver` seam with exactly one implementation (`token.StaticAudiences`) and explicitly deferred dynamic resolvers to a "later change". This is that change. Operationally the motivation is the shared-proxy (`X-Upstream-Host`) deployment mode: today every tenant behind the shared proxy must share a single audience set, or operators must run one sidecar per audience. Neither matches how the proxy is actually being deployed.

Current state:
- `cfg.TokenAudiences` is required at boot.
- `cmd/root.go` builds `token.StaticAudiences(cfg.TokenAudiences)` unconditionally.
- `token.Source` caches tokens keyed on the normalized joined audience string via a `sync.Map` with no eviction.
- `internal/router.Resolver` has already set `req.URL.Host` to the target host (pinned or header-driven) by the time the forwarder's `Director` runs.

The resolver seam is the whole point of this change — it was designed for this.

## Goals / Non-Goals

**Goals:**
- Audience follows the forwarded request's target host when no static audience is configured.
- No change to the forwarder injection point or `token.Source` API.
- Behavior with `--token-audience` set stays bit-for-bit identical.
- Host normalization is deterministic so the cache key is stable across equivalent hosts (`API.example.com:443` and `api.example.com` hit the same entry).
- Config still fails fast on malformed audience values when they ARE provided.

**Non-Goals:**
- Cache eviction. The cache stays unbounded; operators watch `token.cached.audiences` and scope IAM. Eviction is a separate follow-up once cardinality is observable.
- Per-host audience mapping (e.g. `host → [aud1, aud2]`). This change derives the audience from the host verbatim.
- Multi-audience JWTs in dynamic mode. `HostAudience` returns a single-element slice.
- Changing the router's host resolution logic. `HostAudience` reads whatever the router produced.

## Decisions

### Derive audience from `req.URL.Host`, not from the inbound `X-Upstream-Host` header

By the time `forwarder.Director` runs, `router.Resolver` has already written `req.URL.Host` — pinned-mode or header-driven, the same field holds the authoritative answer. `HostAudience` reads `req.URL.Host` and does not inspect the inbound header.

Why: keeping the derivation downstream of the router means `HostAudience` is automatically consistent with the actual upstream destination. If a future change adds host rewriting or aliasing in the router, the audience follows for free. It also means there is exactly one rule in the codebase for "what host is this request going to."

Considered alternative: read `X-Upstream-Host` (or `cfg.HostHeader`) directly inside the resolver. Rejected — duplicates router logic, breaks in pinned mode, and gives the wrong answer if routing ever rewrites the host.

### Normalization: lowercase scheme + "://" + lowercase host, strip port

OIDC verifiers conventionally expect URL-form audiences (e.g. `https://service.example.com`), not bare hostnames. The resolver:
1. Reads `req.URL.Scheme` and `req.URL.Host` (both set by the forwarder's Director from the router's output before the resolver is called).
2. Strips the port with `net.SplitHostPort` (tolerates IPv6, re-wraps bracketed IPv6 hosts; falls through if there is no port).
3. Lowercases both scheme and host.
4. Returns `[]string{scheme + "://" + host}`.

Including scheme also keeps cache keys and the `audience` metric attribute self-describing — `https://api.example.com` vs `http://api.example.com` are distinct audiences and are treated as such. Port is dropped so that `api.example.com:443` and `api.example.com` collapse to one cache entry; deployers who actually need port-discriminated audiences can fall back to the static flag.

Trade-off: an empty scheme or host is a programmer error from upstream code; the resolver returns a non-nil error in both cases, which the forwarder surfaces as `502` via the existing `resolver_error` path.

### `HostAudience` errors when host is empty

If `req.URL.Host == ""` at resolve time, `HostAudience` returns a non-nil error. This is defensive: the router already returns `ErrNoUpstream` on empty host, and `server.handler` converts that to `502` before the forwarder runs. But relying on ordering invariants from across packages is fragile, and the cost of the guard is one `if`. The error flows through the existing `resolver_error` → `502` path in `forwarder.ErrorHandler`.

### Resolver selection is config-driven, not flag-driven

There is no `--audience-mode` flag. The decision is implicit:
- `len(cfg.TokenAudiences) > 0` → `StaticAudiences`
- `len(cfg.TokenAudiences) == 0` → `HostAudience`

Why: a mode flag would be a third dimension on top of "flag set / env set / neither" with no failure modes that the simpler rule doesn't already cover. The rule also makes the upgrade path trivial — existing deployments with `TOKEN_AUDIENCE` set keep the exact same resolver.

### Validation: relax the "≥1 audience" rule, keep per-value checks

`cfg.validate` drops the "zero audiences rejected" clause. It keeps:
- No empty string entries (catches `TOKEN_AUDIENCE=,foo`).
- No whitespace in entries.

This means a malformed `TOKEN_AUDIENCE` still fails at boot rather than silently falling back to dynamic mode. "User meant static but typo'd" is a much worse failure than "user meant dynamic and got it."

### No change to `token.Source` or the cache

The cache keys on the normalized joined audience string. A single-element `[host]` slice normalizes to `host`, which naturally becomes one cache entry per distinct upstream host. Zero code changes to `internal/token/token.go`.

Cache growth is O(distinct hosts the proxy forwards to). The existing `token.cached.audiences` gauge exposes this, so operators have a signal. Unbounded growth in pathological cases is an accepted risk for this change; eviction is tracked as a follow-up.

## Risks / Trade-offs

- **[Unbounded cache growth under high host cardinality]** → Accepted for this change. `token.cached.audiences` gauge gives operators visibility; eviction is a planned follow-up. Document in README.
- **[IAM scope expansion]** → Shared-proxy deployments now need `sts:GetWebIdentityToken` for every host the proxy may forward to. Operationally real. Called out explicitly in the README IAM section. Deployers should narrow the trust policy to the expected host set.
- **[Client-controlled audience in header-driven mode]** → In shared-proxy mode the client sets `X-Upstream-Host`, which now also determines the audience. This is not a privilege escalation: the audience is for the same host the request is being sent to, so a client cannot mint a token for one service and send it to another. But it does mean a client who chooses hosts freely can cause the proxy to call STS for novel audiences (cost + cardinality). Mitigated by IAM scoping.
- **[Host normalization differs from STS expectations]** → STS audiences are opaque strings; no standards problem. But if external verifiers expect e.g. `https://host` including scheme, they will reject tokens minted with bare-host audiences. Deployers configuring dynamic mode must match the verifier's expectation; if they can't, they fall back to `--token-audience`. README will note this.
- **[Breaking change to config semantics]** → The flag goes from required to optional; existing configs continue to work unchanged, but a user who was relying on "proxy fails fast if I forget `TOKEN_AUDIENCE`" loses that safety. Release notes must call this out.

## Migration Plan

Single-binary forward-only rollout.

1. Same commit: `HostAudience` resolver + tests, config validation relaxation, `cmd/root.go` selection logic, README updates.
2. Existing deployments with `TOKEN_AUDIENCE` set: no action required; behavior is identical.
3. New dynamic deployments: operator omits `TOKEN_AUDIENCE`, widens the IAM trust policy to cover the expected set of upstream hosts, deploys.
4. Rollback: revert the deploy. Deployments that moved to dynamic mode must re-set `TOKEN_AUDIENCE` before rolling back to a prior binary (where the flag is required).

## Open Questions

- None blocking. Host-normalization choices (lowercase, strip port, no scheme) are the path of least surprise; if a deployer needs different behavior, they use the static flag.
