## 1. Dependencies

- [x] 1.1 `go get go.opentelemetry.io/otel/semconv/v1.26.0` (or the current stable semconv version at implementation time — verify the Go SDK's latest stable release)
- [x] 1.2 `go mod tidy`; commit `go.mod` / `go.sum`

## 2. Version resolver (`internal/metrics`)

- [x] 2.1 Add a `moduleVersion() string` helper that reads `runtime/debug.ReadBuildInfo()`
- [x] 2.2 Return `info.Main.Version` when it is non-empty and not `"(devel)"`
- [x] 2.3 Otherwise, scan `info.Settings` for `vcs.revision`; if present return its first 12 characters
- [x] 2.4 Otherwise return the literal `"(devel)"`
- [x] 2.5 Unit tests: (a) returns the tagged version path by stubbing via a small indirection; (b) returns `"(devel)"` when neither is available

## 3. Meter construction

- [x] 3.1 In `internal/metrics.Provider.Instruments()` (or wherever the meter is constructed), pass `metric.WithInstrumentationVersion(moduleVersion())` and `metric.WithSchemaURL(semconv.SchemaURL)`
- [x] 3.2 Remove unused imports; confirm no other call sites construct their own meter

## 4. Rename instruments

- [x] 4.1 Delete the `HTTPRequestsTotal` field and its creation — gone entirely
- [x] 4.2 Delete the `UpstreamRequestsTotal` field and its creation
- [x] 4.3 Rename `HTTPRequestDuration` instrument name to `http.server.request.duration`
- [x] 4.4 Rename `HTTPRequestsInFlight` instrument name to `http.server.active_requests`
- [x] 4.5 Rename `UpstreamRequestDuration` instrument name to `http.client.request.duration`
- [x] 4.6 Update struct field names to mirror the new semconv-ish names (`HTTPServerRequestDuration`, `HTTPServerActiveRequests`, `HTTPClientRequestDuration`); update `NoopInstruments`
- [x] 4.7 Remove the `HTTPRequestsTotal.Add` / `UpstreamRequestsTotal.Add` call sites in `internal/server` and `internal/forwarder`

## 5. Attribute migration

- [x] 5.1 Add an `internal/metrics` helper `canonicalMethod(string) string` returning the uppercased method if it is one of `GET, HEAD, POST, PUT, DELETE, CONNECT, OPTIONS, TRACE, PATCH`, else `"_OTHER"`. Unit test each branch.
- [x] 5.2 In `internal/server`, replace `attribute.String("method", ...)` / `attribute.String("status_class", ...)` with `semconv.HTTPRequestMethodKey.String(canonicalMethod(method))` and `semconv.HTTPResponseStatusCodeKey.String(strconv.Itoa(status))`. (Use the semconv package's attribute keys, not raw strings.)
- [x] 5.3 In `internal/forwarder`, split the resolved upstream host: add `internal/metrics.hostAndPort(u *url.URL) (host string, port int, ok bool)` that uses `u.Port()` / `u.Hostname()` and infers the scheme default (80/443). Emit `server.port` ONLY when the port differs from the scheme default or was explicitly set.
- [x] 5.4 Replace `outcome` attribute with `error.type` logic:
  - Success (`err == nil && status < 500`): no `error.type`
  - Timeout (`errors.Is(err, context.DeadlineExceeded)` or `net.Error.Timeout()`): `"timeout"`
  - Connection refused (`errors.Is(err, syscall.ECONNREFUSED)`): `"connection_refused"`
  - Any other transport error: `"unknown"`
  - Upstream-returned 5xx without transport error: no `error.type`
- [x] 5.5 Factor the error classifier into a pure `classifyError(err error) (attrKey string, ok bool)` in `internal/metrics` (or `internal/forwarder`) and unit test each branch with synthetic errors

## 6. Tests

- [x] 6.1 Update `internal/config/config_test.go` — no changes expected, but re-run.
- [x] 6.2 Update `internal/metrics/metrics_test.go` scrape assertions: assert on `http_server_request_duration_seconds_count`, `http_server_active_requests`, `http_client_request_duration_seconds`, `http_request_method="GET"`, `http_response_status_code="200"`, `server_address="..."`. Assert that `otel_scope_version` is non-empty and `otel_scope_schema_url` matches `https://opentelemetry.io/schemas/`.
- [x] 6.3 Update `internal/server/metrics_test.go`: replace `method="GET"` → `http_request_method="GET"`, `status_class="2xx"` → `http_response_status_code="200"`, `outcome="success"` absent → assert no `error_type` label on the success row. Add a case for `_OTHER` bucketing with method `FOO`.
- [x] 6.4 Add a forwarder-level test covering all four `error.type` paths (success none, timeout, connection refused, unknown) using a fake `RoundTripper`.
- [x] 6.5 Remove any assertions on the deleted `*_total` metrics.

## 7. Documentation

- [x] 7.1 Update the README Observability section:
  - Replace the instrument list with the semconv names and their Prometheus-translated forms.
  - Replace the attribute list (`method`, `status_class`, `upstream_host`, `outcome`) with the semconv attributes (`http.request.method`, `http.response.status_code`, `server.address`, `server.port`, `error.type`).
  - Add a "Migration from the previous names" block with a PromQL rewrite table.
  - Note the scope-version behavior (released → tag; dev → `vcs.revision`; otherwise `(devel)`).

## 8. Verify

- [x] 8.1 `go build ./...` and `go test ./...` pass
- [x] 8.2 Run the proxy against a test upstream; `curl http://localhost:9090/metrics` shows the renamed series with `otel_scope_version` and `otel_scope_schema_url` populated on every metric line
- [x] 8.3 Generate load; confirm histogram `_count` tracks request volume one-for-one (check by comparing a manual count of `curl` invocations to `http_server_request_duration_seconds_count`)
- [x] 8.4 Send a request with method `FOO`; scrape shows `http_request_method="_OTHER"` and no `FOO` series
- [x] 8.5 Point the proxy at a closed TCP port; scrape shows `error_type="connection_refused"` on the client histogram; scrape a success case; confirm no `error_type` label on that series
- [x] 8.6 Build the binary with a dirty git tree (`go build`); scrape shows `otel_scope_version` populated with the short VCS revision
