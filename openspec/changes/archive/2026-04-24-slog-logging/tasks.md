## 1. Configuration

- [x] 1.1 Add `LogLevel string` and `LogFormat string` to `config.Config`
- [x] 1.2 Register `--log-level` (default `info`) and `--log-format` (default `json`) flags
- [x] 1.3 Validate values (level ∈ {debug, info, warn, error}; format ∈ {json, text}); case-insensitive
- [x] 1.4 Unit tests: defaults, valid values, invalid values

## 2. Logging package (`internal/logging`)

- [x] 2.1 Create `internal/logging` package
- [x] 2.2 Implement `New(cfg *config.Config) *slog.Logger` that builds a handler (`JSONHandler` or `TextHandler`) with the configured level, writing to `os.Stdout`
- [x] 2.3 Expose `Install(*slog.Logger)` that sets `slog.SetDefault(...)`
- [x] 2.4 Unit tests: level filtering (debug suppressed at info), format switching writes to a buffer

## 3. Replace log.Printf call sites

- [x] 3.1 `cmd/root.go`: replace startup `log.Printf("listening ...")` with `slog.Info("server starting", "addr", addr, "tls", enabled)`
- [x] 3.2 `internal/server`: replace access log `log.Printf` with a single `slog.Info("request", ...)` carrying `method`, `path`, `target`, `status`, `duration_ms`
- [x] 3.3 `internal/forwarder`: replace upstream error `log.Printf` with `slog.Error("upstream error", ...)` carrying `target`, `status`, `err`
- [x] 3.4 Install the configured logger as default in `cmd/root.go` RunE before server boot

## 4. Verify

- [x] 4.1 `go build ./...` and `go test ./...` pass
- [x] 4.2 Start proxy (defaults) against httpbin; make a request; confirm stdout shows JSON line with expected keys
- [x] 4.3 Start proxy with `--log-format=text`; confirm logfmt output
- [x] 4.4 Log level filtering verified via unit tests
- [x] 4.5 Invalid `--log-level=trace` exits with clear error
