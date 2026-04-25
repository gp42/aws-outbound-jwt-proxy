## 1. token package: multi-audience API

- [x] 1.1 Change `token.Source` interface to `Token(ctx context.Context, audiences []string) (string, error)`.
- [x] 1.2 Add an internal `normalizeAudiences([]string) ([]string, string, error)` helper that returns the sorted+deduped slice, the `,`-joined cache key / metric attribute value, and an error for empty/nil input.
- [x] 1.3 Update `source.Token` to call the normalizer, use the joined key for the cache `sync.Map`, use the normalized slice as `Audience` in the STS call, and use the joined value for the `audience` metric attribute.
- [x] 1.4 Update `source.fetch` to key the cache `Swap` on the joined key.
- [x] 1.5 Update `observeHit` / `observeMiss` / `observeFetch` to take the already-normalized joined attribute value instead of a raw string.
- [x] 1.6 Add `AudienceResolver` interface (`Resolve(req *http.Request) ([]string, error)`) in `internal/token` (new file `internal/token/resolver.go`).
- [x] 1.7 Add `StaticAudiences(audiences []string) AudienceResolver` implementation that returns the configured slice verbatim per `Resolve` call and ignores the request.
- [x] 1.8 Update `internal/token/tokentest` fake to match the new `Source.Token` signature; make the fake key its internal map on the joined-normalized form too, so test assertions behave like production.
- [x] 1.9 Update `internal/token/token_test.go` and `metrics_test.go`:
  - single-audience path still works
  - multi-audience slice `["a","b"]` produces one STS call with `Audience=["a","b"]`
  - `["b","a"]` and `["a","b"]` share a cache entry and emit the same `audience=a,b` attribute
  - empty / nil slice returns error, STS not called
  - gauge counts distinct normalized sets only
- [x] 1.10 Add a `resolver_test.go` covering `StaticAudiences` verbatim-return and request-independence.

## 2. config: token-audience flag

- [x] 2.1 Add `TokenAudiences []string` to `config.Config`.
- [x] 2.2 Add flag `--token-audience` as a repeatable `StringSlice` (or `StringArray` — pick the one that matches existing pflag usage and doesn't comma-split at the CLI level).
- [x] 2.3 Extend env-var fallback so `TOKEN_AUDIENCE` is parsed as a comma-separated list into the same slice when the flag is unset.
- [x] 2.4 Extend `Config.validate` to require `len(TokenAudiences) >= 1`, reject any empty entry, and reject any entry containing ASCII whitespace.
- [x] 2.5 Update `config_test.go`:
  - missing audience fails validation
  - single flag works
  - repeated flag works
  - env comma-list works
  - CLI flag overrides env (matches existing precedence rule)
  - empty element in env list is rejected
  - whitespace in value is rejected

## 3. forwarder: inject bearer token

- [x] 3.1 Change `forwarder.New` signature to accept a `token.Source` and a `token.AudienceResolver`.
- [x] 3.2 In `Director`, before any network call: resolve audiences, fetch the token, and set `req.Header.Set("Authorization", "Bearer "+tok)`. If the inbound request had a non-empty `Authorization`, emit a `slog.Debug` before replacing.
- [x] 3.3 On resolver or token error inside `Director`: stash the error and its kind (`resolver_error` / `fetch_error`) in the request context via a new ctx key, clear `req.URL.Host` to force `ReverseProxy` into the `ErrorHandler` path, and record `error` at `slog.Error` (include AWS error via `errors.Unwrap` where applicable, but NOT in the HTTP response body).
- [x] 3.4 Extend `ErrorHandler`: if the stashed error is one of the token sentinels, respond `502 Bad Gateway` with fixed body `token unavailable\n`. Otherwise keep current upstream-error behaviour.
- [x] 3.5 Add a `token.result` attribute to the `http.client.*` recording in `recordClient` (and the 502 path), with values `ok` / `fetch_error` / `resolver_error`. Default when the attribute is not set is `ok` for the existing upstream-error path to avoid accidental cardinality.
- [x] 3.6 Update `forwarder_test.go`:
  - token ok → header attached, upstream sees `Authorization: Bearer <jwt>`
  - inbound `Authorization` replaced (and not forwarded)
  - `tokentest.Source` returning an error → 502, upstream NOT called, metric carries `token.result=fetch_error`
  - resolver returning an error → 502, token source NOT called, metric carries `token.result=resolver_error`
  - response body on 502 is the fixed string, does NOT contain AWS error detail

## 4. server + cmd wiring

- [x] 4.1 Change `server.New` signature to take a `token.Source` and a `token.AudienceResolver`; pass both into `forwarder.New`.
- [x] 4.2 Update `server_test.go` / `metrics_test.go` to construct a `tokentest.Source` and `token.StaticAudiences(...)` where needed.
- [x] 4.3 In `cmd/root.go`: remove the `_ = tokenSource` placeholder; build `resolver := token.StaticAudiences(cfg.TokenAudiences)`; pass `tokenSource` and `resolver` to `server.New`.

## 5. docs

- [x] 5.1 Add `--token-audience` / `TOKEN_AUDIENCE` row to the README flag table with a note that it is required and repeatable.
- [x] 5.2 Update the README "How it works" section: the forwarder now attaches the JWT (remove any "in a follow-up" hedge).
- [x] 5.3 Add a short "First-request latency" note: the first request per unique audience set incurs one STS round-trip.

## 6. verification

- [x] 6.1 `make test` (or `go test ./...`) passes.
- [x] 6.2 `make lint` / `go vet ./...` clean.
- [x] 6.3 Manual smoke: start with `--token-audience=https://example.test`, send a request to an `httptest` upstream that echoes headers, confirm `Authorization: Bearer …` arrives.
- [x] 6.4 Manual failure smoke: point at a bogus STS (invalid creds in env), confirm client receives `502` with body `token unavailable` and logs show the AWS error at `error` level.
