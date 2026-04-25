## 1. Project scaffolding

- [x] 1.1 Keep existing cobra scaffold (`main.go`, `cmd/root.go`, `cmd/serve.go`)
- [x] 1.2 Create directory layout: `internal/config/`, `internal/router/`, `internal/server/`

## 2. Configuration (`internal/config`)

- [x] 2.1 Define `Config` struct with fields `ListenAddr`, `UpstreamHost`, `UpstreamScheme`, `HostHeader`
- [x] 2.2 Implement `BindFlags(fs *pflag.FlagSet)` registering the four flags with cobra defaults
- [x] 2.3 Implement env-fallback helper: walk `fs.VisitAll`; for any flag not `Changed`, look up `UPPER_SNAKE` env var and apply via `flag.Value.Set`
- [x] 2.4 Implement `Load(fs *pflag.FlagSet, env func(string) (string, bool)) (*Config, error)` that runs env fallback, reads values, and validates
- [x] 2.5 Validate `UpstreamScheme` is one of `http`/`https`; reject other values with a clear error
- [x] 2.6 Validate `UpstreamHost` does not contain `://`; on failure, error message directs user to `--upstream-scheme`
- [x] 2.7 Unit tests: defaults, CLI-only, env-only, CLI-wins-over-env, invalid scheme, full-URL upstream host

## 3. Upstream router (`internal/router`)

- [x] 3.1 Define `Resolver` type holding pinned host, scheme, and header name
- [x] 3.2 Implement `Resolve(r *http.Request) (*url.URL, error)` — pinned wins; otherwise read header
- [x] 3.3 Special-case header name equal to `Host` (case-insensitive) to read `r.Host` instead of `r.Header`
- [x] 3.4 Preserve `r.URL.Path` and `r.URL.RawQuery` on the resolved URL
- [x] 3.5 Define sentinel error `ErrNoUpstream` for unresolvable requests; expose configured header name via resolver API so the handler can mention it in the 400 body
- [x] 3.6 Unit tests covering every scenario in `specs/upstream-routing/spec.md`

## 4. HTTP server (`internal/server`)

- [x] 4.1 Implement handler that calls `Resolver.Resolve`, `log.Printf`s `resolved upstream: <url>` on success, writes `204 No Content`
- [x] 4.2 On `ErrNoUpstream`, respond `400 Bad Request` with body `missing upstream: set --upstream-host or provide <header-name>`
- [x] 4.3 Expose `New(cfg *config.Config) *http.Server` that wires address and handler
- [x] 4.4 Unit tests using `httptest` for the 204 path, 400 path, and pinned vs header modes

## 5. Wire serve subcommand (`cmd/serve.go`)

- [x] 5.1 Replace current placeholder; register flags via `config.BindFlags(serveCmd.Flags())`
- [x] 5.2 In `RunE`, call `config.Load` with `os.LookupEnv`; on error, return it (cobra prints + exits non-zero)
- [x] 5.3 Build `*http.Server` via `server.New(cfg)` and call `ListenAndServe`
- [x] 5.4 Log listen address on startup

## 6. Manual verification

- [x] 6.1 `go build ./...` succeeds and `go test ./...` passes
- [x] 6.2 Pinned mode: `./proxy serve --upstream-host=api.example.com`, curl any path, verify stdout log and `204`
- [x] 6.3 Header mode: `./proxy serve` (no pin), curl with `X-Upstream-Host: api.example.com`, verify log and `204`
- [x] 6.4 Missing upstream: `./proxy serve` (no pin), curl with no header, verify `400` and error body
- [x] 6.5 Env fallback: unset flag, export `UPSTREAM_HOST=...`, verify pinned mode activates
