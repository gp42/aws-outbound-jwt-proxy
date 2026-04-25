## 1. Host audience resolver

- [x] 1.1 Add `HostAudience` type (zero-value usable) in `internal/token/resolver.go` implementing `AudienceResolver`.
- [x] 1.2 Implement host normalization: `net.SplitHostPort` to strip port (falling through when absent), lowercase the host, return single-element slice.
- [x] 1.3 Return a non-nil error from `Resolve` when `req.URL.Host` is empty; message clearly attributes the failure to the resolver.
- [x] 1.4 Add unit tests in `internal/token/resolver_test.go` covering: host with port, host without port, mixed-case host, IPv6 bracketed host with and without port, empty host error.

## 2. Configuration

- [x] 2.1 In `internal/config/config.go`, remove the "zero audiences rejected" branch from `cfg.validate()` while keeping the empty-string and whitespace checks.
- [x] 2.2 Update the flag help string for `--token-audience` to state the flag is optional and describe the host-derived fallback.
- [x] 2.3 Update `internal/config/config_test.go`: replace the "missing audience fails" case with "missing audience is accepted"; keep existing failure cases for empty and whitespace entries.

## 3. Wiring

- [x] 3.1 In `cmd/root.go`, replace the unconditional `token.StaticAudiences(cfg.TokenAudiences)` with a selection: `StaticAudiences` when `len(cfg.TokenAudiences) > 0`, otherwise `HostAudience{}`.
- [x] 3.2 Log at `info` which resolver was selected at startup (name only, no request-scope data).

## 4. Documentation

- [x] 4.1 README flag table: mark `--token-audience` default as "(empty; enables host-derived audience)" and revise the Purpose column.
- [x] 4.2 Add a "Dynamic audience" subsection under "How it works" explaining the host-derived fallback, cache keying per host, and IAM scope expansion for shared-proxy deployments.
- [x] 4.3 Call out `token.cached.audiences` in the metrics / observability section as the gauge to watch for cache cardinality under dynamic mode.
- [x] 4.4 Note in the IAM subsection that dynamic mode requires `sts:GetWebIdentityToken` for every host the proxy may forward to, and recommend scoping the trust policy to the expected host set.

## 5. Verification

- [x] 5.1 Run `go test ./...` and confirm new resolver and config tests pass.
- [x] 5.2 Run the binary locally with no `TOKEN_AUDIENCE` and confirm it starts, logs the `HostAudience` selection, and successfully proxies a request (manual smoke test against a mock upstream is sufficient).
- [x] 5.3 Run the binary locally with `TOKEN_AUDIENCE=https://api.example.com` and confirm behavior is unchanged from the prior release (regression gate).
- [x] 5.4 Run `openspec validate dynamic-token-audience` and fix any schema errors.
