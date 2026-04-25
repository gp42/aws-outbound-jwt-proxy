## 1. Dependencies

- [x] 1.1 `go get github.com/aws/aws-sdk-go-v2/config` and `github.com/aws/aws-sdk-go-v2/service/sts`
- [x] 1.2 `go get golang.org/x/sync/singleflight`
- [x] 1.3 `go mod tidy`

## 2. Configuration

- [x] 2.1 Add `TokenSigningAlgorithm string`, `TokenDuration time.Duration`, `TokenRefreshSkew time.Duration` to `config.Config`
- [x] 2.2 Register `--token-signing-algorithm` (default `RS256`, env `TOKEN_SIGNING_ALGORITHM`)
- [x] 2.3 Register `--token-duration` (default `900s`, env `TOKEN_DURATION`) via `fs.Duration`
- [x] 2.4 Register `--token-refresh-skew` (default `60s`, env `TOKEN_REFRESH_SKEW`) via `fs.Duration`
- [x] 2.5 Validate: algorithm ∈ {`RS256`, `ES384`}; duration ∈ `[60s, 3600s]`; skew > 0 and skew < duration
- [x] 2.6 Unit tests: defaults; env; CLI; invalid algorithm; duration below min / above max; skew zero; skew ≥ duration

## 3. Token package (`internal/token`)

- [x] 3.1 Create `internal/token` package and `Source` interface: `Token(ctx context.Context, audience string) (string, error)`
- [x] 3.2 Define an internal `stsClient` interface matching the `GetWebIdentityToken` shape used, so tests can stub it
- [x] 3.3 Implement `New(cfg *config.Config) (Source, error)` that loads AWS SDK default config, builds an STS client, and returns the concrete `Source`
- [x] 3.4 Cache: `sync.Map` keyed by audience, value `*cacheEntry{token string, exp time.Time}`
- [x] 3.5 Single-flight coalescing via `golang.org/x/sync/singleflight`, keyed by audience
- [x] 3.6 Fetch path: call `GetWebIdentityToken` with `Audience=[audience]`, `SigningAlgorithm`, `DurationSeconds`; store `(WebIdentityToken, Expiration)` in cache
- [x] 3.7 Validity check: cached entry usable iff `now + cfg.TokenRefreshSkew < entry.exp`
- [x] 3.8 Error passthrough: wrap with `fmt.Errorf("token: audience=%q: %w", ...)` — preserve underlying AWS error types for `errors.As`
- [x] 3.9 Propagate caller `ctx` to the STS client call

## 4. Metrics

- [x] 4.1 Register instruments under the existing OTel meter: `token.fetch.count` (counter), `token.cache.hit.count` (counter), `token.cache.miss.count` (counter), `token.cached.audiences` (gauge)
- [x] 4.2 Record on cache hit, cache miss, fetch ok, fetch error (with `error_class` attribute derived from `errors.As` against known AWS error types)
- [x] 4.3 Implement the gauge by iterating the cache `sync.Map` at collection time (or maintain an atomic counter on insert)

## 5. Test fake (`internal/token/tokentest`)

- [x] 5.1 Create `tokentest` subpackage
- [x] 5.2 `tokentest.New(map[string]string) Source` — returns pre-canned tokens keyed by audience
- [x] 5.3 Unknown-audience call returns a deterministic error

## 6. Unit tests (`internal/token`)

- [x] 6.1 Cache miss calls STS with expected audience/algorithm/duration
- [x] 6.2 Cache hit returns cached token without calling STS
- [x] 6.3 Near-expiry (`now + skew ≥ exp`) triggers refresh
- [x] 6.4 Two audiences cached independently; one fetch each
- [x] 6.5 Single-flight: 100 concurrent callers on cold cache → exactly one STS call, all receive same token
- [x] 6.6 `AccessDenied` from STS returns an error identifiable via `errors.As` to the AWS smithy error type
- [x] 6.7 `OutboundWebIdentityFederationDisabled` similarly preserved
- [x] 6.8 Transport error does NOT populate the cache
- [x] 6.9 Context cancellation before STS responds returns wrapped `ctx.Err()` and does not cache
- [x] 6.10 Metrics assertions: hit/miss/fetch counters increment with expected attributes

## 7. Wiring (`cmd/root.go`)

- [x] 7.1 After config load, construct the token `Source` via `token.New(cfg)`; fail fast if AWS SDK config load fails
- [x] 7.2 The `Source` is constructed but not yet consumed by the forwarder — verify startup succeeds and the instance is reachable for the follow-up change (e.g. stash on a server struct)

## 8. Verify

- [x] 8.1 `go build ./...` and `go test ./...` pass
- [x] 8.2 Start proxy with valid AWS credentials and an audience-permitting IAM role; call `Source.Token(ctx, "https://example.test")` via a small test harness or one-off integration test; decode the returned JWT and verify `aud`, `sub`, `exp`
- [x] 8.3 Start proxy with a role lacking `sts:GetWebIdentityToken` for the audience; confirm `AccessDenied` surfaces in logs and in the `token.fetch.count{result=error,error_class=AccessDenied}` metric
- [x] 8.4 Start proxy with invalid `--token-duration=30s`; confirm startup fails with a clear error
