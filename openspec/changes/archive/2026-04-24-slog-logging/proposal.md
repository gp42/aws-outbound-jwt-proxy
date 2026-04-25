## Why

We currently use `log.Printf` with ad-hoc line formats. That's fine for local hacking but unstructured output is a pain to ship to log aggregators and to grep reliably. Moving to stdlib `log/slog` gives us structured key/value logging with JSON output, configurable level, and a stable foundation we can later bridge to OTel Logs if needed — without adding a third-party dependency.

## What Changes

- Introduce a package-level slog logger configured at startup.
- Add two new flags (with env fallback): `--log-level` (default `info`; accepts `debug`, `info`, `warn`, `error`) and `--log-format` (default `json`; accepts `json` or `text`).
- Replace every existing `log.Printf` call with an equivalent structured `slog.*` call:
  - Access log line: method, path, target URL, status, duration_ms, protocol — as structured attrs.
  - Startup listening line: addr, tls attrs.
  - Upstream error: target, status, error — as structured attrs.
  - Config errors: left to cobra's usage output (unchanged).
- Introduce a tiny `internal/logging` package that owns logger construction and exposes a helper to derive a request-scoped logger (with trace/request IDs later).
- Stdout-only by default; no file writing, no rotation (12-factor).

## Capabilities

### New Capabilities
<!-- None. Logging is cross-cutting; no new capability spec. -->

### Modified Capabilities
- `proxy-configuration`: Adds `--log-level` and `--log-format` flags + env vars.
- `request-forwarding`: The "request log line" requirement changes from a single free-form log.Printf line to a structured slog record with defined attribute keys.

## Impact

- Code: new `internal/logging` package; call-site changes in `cmd/root.go`, `internal/server`, `internal/forwarder`.
- No new third-party dependencies — `log/slog` is stdlib (Go 1.21+; we're on 1.25).
- Log format at stdout changes from `2026/04/22 10:06:49 GET /foo -> ...` to either JSON (default) or logfmt-style text.
- Out of scope: OTel Logs bridge, trace/span ID enrichment (covered later once we add OTel metrics/traces), log sampling, file outputs, log rotation.
