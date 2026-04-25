## ADDED Requirements

### Requirement: Acquire JWT from AWS STS for a given audience

The token library SHALL expose a `Source.Token(ctx, audience)` operation that returns a JWT string identifying the caller to an external service named by `audience`. On cache miss, the library SHALL call `sts:GetWebIdentityToken` with `Audience=[audience]`, `SigningAlgorithm` set to the configured value, and `DurationSeconds` set to the configured value, returning the token's `WebIdentityToken` field.

#### Scenario: Cache miss triggers STS call

- **WHEN** the library is asked for a token for audience `https://api.example.com` and the cache is empty
- **THEN** the library calls STS `GetWebIdentityToken` with `Audience=["https://api.example.com"]`, the configured algorithm, and the configured duration, and returns the resulting token

#### Scenario: Audience passed verbatim

- **WHEN** the caller passes an audience string
- **THEN** the library sends that exact string as the single element of the STS `Audience` array, without modification or normalization

### Requirement: Serve cached token when not near expiry

The library SHALL cache tokens keyed by audience and SHALL return a cached token without calling STS when the current time plus the configured refresh skew is strictly less than the token's expiration time.

#### Scenario: Cache hit within validity window

- **WHEN** a token for audience `A` is cached with `exp` at `now + 10m` and `refresh-skew` is `60s`
- **THEN** a second call for audience `A` returns the cached token without a STS call

#### Scenario: Near-expiry triggers refresh

- **WHEN** a cached token for audience `A` has `exp` at `now + 30s` and `refresh-skew` is `60s`
- **THEN** the library calls STS to fetch a new token, replaces the cache entry, and returns the new token

#### Scenario: Independent cache entries per audience

- **WHEN** tokens are requested for audiences `A` and `B`
- **THEN** each audience has its own cache entry and STS is called at most once per audience (per refresh cycle)

### Requirement: Coalesce concurrent fetches per audience

The library SHALL ensure that concurrent callers requesting a token for the same audience on a cache miss share a single STS call; all such callers SHALL receive the same token or the same error.

#### Scenario: Thundering herd produces one STS call

- **WHEN** 100 goroutines concurrently call `Token(ctx, "https://api.example.com")` on an empty cache
- **THEN** exactly one `GetWebIdentityToken` call is made to STS, and all 100 callers receive the resulting token

#### Scenario: Different audiences do not coalesce

- **WHEN** concurrent callers request tokens for audiences `A` and `B` on an empty cache
- **THEN** STS receives one call per audience, executed concurrently

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
