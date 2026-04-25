## Why

The metrics we ship today use ad-hoc names and attributes (`method`, `status_class`, `upstream_host`, `outcome`, plus redundant `*_total` counters alongside duration histograms). That works, but it doesn't line up with the OpenTelemetry HTTP semantic conventions that every standard dashboard, exporter, and OTel-aware backend assumes. Moving to semconv gets us:

- Out-of-the-box compatibility with Grafana / OTel collector dashboards for HTTP servers and clients.
- Truthful scope metadata (`otel_scope_schema_url`, `otel_scope_version` stop being empty), making the meter self-describing when scraped from a fleet running mixed versions.
- Fewer redundant series (drop `*_total` counters once histograms carry the same count via `_count`).

Separately, hardcoding a version string in the meter is a lie the moment we cut a release. `runtime/debug.ReadBuildInfo()` gives us the real module version (or `(devel)` in dev, or the vcs stamp when built from a clean git tree) at zero maintenance cost.

## What Changes

- **BREAKING** (metrics-only, no API): rename metrics and attributes to match OTel HTTP semconv:
  - `http_server_requests_total` → **deleted**. Use `http_server_request_duration_seconds_count` instead.
  - `upstream_requests_total` → **deleted**. Use `http_client_request_duration_seconds_count` instead.
  - `http_server_request_duration_seconds` → `http.server.request.duration` (Prom: `http_server_request_duration_seconds`, name unchanged after translation).
  - `http_server_requests_in_flight` → `http.server.active_requests` (Prom: `http_server_active_requests`).
  - `upstream_request_duration_seconds` → `http.client.request.duration` (Prom: `http_client_request_duration_seconds`).
- Attribute renames on every instrument:
  - `method` → `http.request.method`, uppercased, with `"_OTHER"` for non-standard methods (per semconv).
  - `status_class` → `http.response.status_code` (full integer, not a bucketed class).
  - `upstream_host` → `server.address` (and `server.port` when a non-default port is set).
  - `outcome` → `error.type`, present **only on failures**. Success emits no `error.type`. Values: `"timeout"` for deadline exceeded, otherwise a short Go-error classifier (`"connection_refused"`, `"reset"`, `"unknown"`).
- Instrumentation scope:
  - Set `metric.WithInstrumentationVersion(...)` on the meter, pulled from `runtime/debug.ReadBuildInfo()` (module version or VCS revision; `"(devel)"` fallback).
  - Set `metric.WithSchemaURL(semconv.SchemaURL)` using the `go.opentelemetry.io/otel/semconv/v1.26.0` (or current stable) constant — now truthful since we've adopted semconv names.
- Update README metric/attribute table and example queries.

## Capabilities

### New Capabilities
<!-- None. Naming migration only. -->

### Modified Capabilities
- `metrics-observability`: renames every instrument and attribute, drops two counters, declares a schema URL, and mandates the semconv `"_OTHER"` method-bucketing rule.

## Impact

- Code: `internal/metrics` (meter construction, instrument registration, `NoopInstruments`), `internal/server` (record call sites + status_class removal), `internal/forwarder` (record call sites + `outcome`→`error.type` + host/port split), test files in `internal/metrics` and `internal/server` that assert on metric names and label values.
- Dependencies: add `go.opentelemetry.io/otel/semconv/v1.26.0` (same module tree, no new direct-module-boundary cost).
- Operations: **breaking for anyone already scraping the metrics from the previous change**. Dashboards and alerts need updating. Since the original metrics change has not shipped to any production scrape target yet (still in an un-archived change within this repo), we treat this as a rename in-place with no deprecation period. README will call the change out.
- Out of scope: OTel traces, OTel logs bridge, `http.route` (no route template available), `url.scheme` / `network.protocol.*` enrichment (can follow as a separate small change if someone wants the standard dashboards that require them), migrating off the Prometheus exporter.
