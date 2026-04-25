# token-acquisition Specification

## Purpose
TBD - created by archiving change jwt-token-library. Update Purpose after archive.
## Requirements
### Requirement: Acquire JWT from AWS STS for a given audience

The token library SHALL expose a `Source.Token(ctx, audiences)` operation where `audiences` is a non-empty `[]string`. The library SHALL normalize the slice by sorting lexicographically and removing exact duplicates before it is used as either the cache key or the argument to AWS. On cache miss, the library SHALL call `sts:GetWebIdentityToken` with `Audience` set to the normalized slice, `SigningAlgorithm` set to the configured value, and `DurationSeconds` set to the configured value, returning the token's `WebIdentityToken` field. An empty or nil `audiences` argument SHALL return an error without calling STS.

#### Scenario: Cache miss with single audience triggers STS call

- **WHEN** the library is asked for a token for audiences `["https://api.example.com"]` and the cache is empty
- **THEN** the library calls STS `GetWebIdentityToken` with `Audience=["https://api.example.com"]`, the configured algorithm, and the configured duration, and returns the resulting token

#### Scenario: Cache miss with multiple audiences sends all

- **WHEN** the library is asked for a token for audiences `["https://a.example.com", "https://b.example.com"]` and the cache is empty
- **THEN** the library calls STS `GetWebIdentityToken` exactly once with `Audience=["https://a.example.com", "https://b.example.com"]` (normalized order) and returns the single resulting token

#### Scenario: Audience order and duplicates are normalized

- **WHEN** the library receives `["b", "a", "b"]` and then later `["a", "b"]`
- **THEN** both calls resolve to the same cache entry, exactly one STS call is made across both, and the STS call's `Audience` argument is `["a", "b"]`

#### Scenario: Empty audience slice rejected

- **WHEN** the library is called with `Token(ctx, nil)` or `Token(ctx, []string{})`
- **THEN** the library returns an error without calling STS

### Requirement: Serve cached token when not near expiry

The library SHALL cache tokens keyed by the normalized audience set and SHALL return a cached token without calling STS when the current time plus the configured refresh skew is strictly less than the token's expiration time.

#### Scenario: Cache hit within validity window

- **WHEN** a token for audience set `{A}` is cached with `exp` at `now + 10m` and `refresh-skew` is `60s`
- **THEN** a second call with `["A"]` returns the cached token without a STS call

#### Scenario: Near-expiry triggers refresh

- **WHEN** a cached token for audience set `{A}` has `exp` at `now + 30s` and `refresh-skew` is `60s`
- **THEN** the library calls STS to fetch a new token, replaces the cache entry, and returns the new token

#### Scenario: Independent cache entries per distinct audience set

- **WHEN** tokens are requested for audience sets `{A}`, `{B}`, and `{A, B}`
- **THEN** each distinct normalized set has its own cache entry and STS is called at most once per set (per refresh cycle)

#### Scenario: Superset and subset do not share cache entries

- **WHEN** a token for `{A}` is cached and a caller requests a token for `{A, B}`
- **THEN** the library treats `{A, B}` as a cache miss and makes a new STS call

### Requirement: Coalesce concurrent fetches per audience

The library SHALL ensure that concurrent callers requesting a token for the same normalized audience set on a cache miss share a single STS call; all such callers SHALL receive the same token or the same error.

#### Scenario: Thundering herd produces one STS call

- **WHEN** 100 goroutines concurrently call `Token(ctx, []string{"https://api.example.com"})` on an empty cache
- **THEN** exactly one `GetWebIdentityToken` call is made to STS, and all 100 callers receive the resulting token

#### Scenario: Different audience sets do not coalesce

- **WHEN** concurrent callers request tokens for `{A}` and `{B}` on an empty cache
- **THEN** STS receives one call per set, executed concurrently

#### Scenario: Differently-ordered slices for the same set coalesce

- **WHEN** one goroutine calls `Token(ctx, []string{"A", "B"})` and another calls `Token(ctx, []string{"B", "A"})` concurrently on an empty cache
- **THEN** exactly one STS call is made and both callers receive the same token

### Requirement: Pass AWS error classes through to the caller

The library SHALL return errors from STS to the caller with the original AWS error type preserved (wrapped only to add audience/operation context), so callers can distinguish policy denials, feature-gate errors, and transport errors by inspecting the wrapped error.

#### Scenario: AccessDenied preserved

- **WHEN** STS returns `AccessDenied` (e.g. audience not permitted by IAM policy)
- **THEN** the library returns an error that wraps the original AWS error and is identifiable as `AccessDenied` via `errors.As`

#### Scenario: OutboundWebIdentityFederationDisabled preserved

- **WHEN** STS returns `OutboundWebIdentityFederationDisabled`
- **THEN** the library returns an error that wraps the original AWS error and is identifiable as that type via `errors.As`

#### Scenario: Transport error preserved

- **WHEN** the STS call fails due to network or DNS error
- **THEN** the library returns an error wrapping the underlying transport error and does NOT cache a failure

### Requirement: Context cancellation propagates to STS call

The library SHALL pass the caller's context to the STS client so that context cancellation or deadline interrupts the fetch.

#### Scenario: Caller cancels context

- **WHEN** a caller invokes `Token(ctx, "A")` on a cache miss and cancels `ctx` before STS responds
- **THEN** the library returns `ctx.Err()` (wrapped) and does not block on the STS call

### Requirement: Emit token-lifecycle metrics

The library SHALL emit metrics via the existing OTel meter for each operation: fetch count (labeled by result and error class), cache hit count, cache miss count, and a gauge of currently cached audiences.

#### Scenario: Cache hit increments hit counter

- **WHEN** a request is served from cache
- **THEN** `token.cache.hit.count` increments with the `audience` attribute, and STS is not called

#### Scenario: Successful fetch increments fetch counter

- **WHEN** a STS fetch succeeds
- **THEN** `token.fetch.count` increments with `result=ok` and the `audience` attribute

#### Scenario: Fetch error labeled by class

- **WHEN** a STS fetch fails with `AccessDenied`
- **THEN** `token.fetch.count` increments with `result=error` and `error_class=AccessDenied`

### Requirement: Provide an in-memory test fake

The library SHALL expose a `tokentest` subpackage whose `New` constructor returns a `Source` implementation driven by a caller-supplied map of audience → token, for use by downstream tests.

#### Scenario: Fake returns preconfigured token

- **WHEN** a test constructs a fake `Source` with `{"A": "token-a"}` and calls `Token(ctx, "A")`
- **THEN** the call returns `"token-a"` without invoking AWS

#### Scenario: Fake returns error for unknown audience

- **WHEN** a test calls `Token(ctx, "missing")` on a fake configured only for audience `"A"`
- **THEN** the call returns a non-nil error identifying the unknown audience

### Requirement: AudienceResolver interface for per-request audience selection

The library SHALL expose an `AudienceResolver` interface with a single method `Resolve(req *http.Request) ([]string, error)` and SHALL ship a built-in implementation `StaticAudiences(audiences []string) AudienceResolver` whose `Resolve` returns the configured slice for every request. The returned slice SHALL be passed verbatim (no filtering or sort) to `Source.Token`, which performs normalization internally.

#### Scenario: StaticAudiences returns configured slice verbatim

- **WHEN** `StaticAudiences([]string{"b", "a"})` is invoked with any `*http.Request`
- **THEN** `Resolve` returns `[]string{"b", "a"}` and `nil` error, and the slice is NOT sorted or deduped by the resolver

#### Scenario: StaticAudiences ignores the request

- **WHEN** `StaticAudiences(S)` is invoked with two different `*http.Request` values
- **THEN** both calls return the same slice `S` and the resolver does not read any field of the request

### Requirement: Token cache metric uses normalized audience-set key

The `token.cached.audiences` gauge and the `audience` metric attribute on `token.fetch.count`, `token.cache.hit.count`, and `token.cache.miss.count` SHALL use the normalized audience set joined with `,` as the attribute value (e.g. `a,b` for set `{a, b}`), so that callers passing the same set in different orders produce a single time series.

#### Scenario: Joined attribute value matches the normalized key

- **WHEN** `Token(ctx, []string{"b", "a"})` is called
- **THEN** the metric `audience` attribute value on the resulting fetch/hit/miss observation is `a,b`

#### Scenario: Gauge counts distinct normalized sets

- **WHEN** tokens for `["a"]`, `["a", "b"]`, and `["b", "a"]` have been fetched and cached
- **THEN** the `token.cached.audiences` gauge reports `2` (the `["a", "b"]` and `["b", "a"]` calls share one entry)

