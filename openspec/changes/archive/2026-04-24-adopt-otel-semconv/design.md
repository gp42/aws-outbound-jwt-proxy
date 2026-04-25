## Context

The `add-otel-metrics-prometheus` change instrumented the proxy with OTel metrics but used ad-hoc names and labels because getting *any* telemetry was the priority. Now we normalize to OTel HTTP semantic conventions so dashboards, alert rules, and collectors that key off standard names work without custom wiring. At the same time we stop hardcoding the meter version string.

## Goals / Non-Goals

**Goals:**
- Every metric and attribute name matches OTel HTTP semconv (v1.26.0 or current stable at implementation time).
- Instrumentation scope carries a truthful `version` and `schema_url`.
- Cardinality stays bounded — specifically, unknown HTTP methods collapse to `"_OTHER"` per semconv, and `error.type` values are drawn from a short closed set.
- No behavioral change to the proxy itself; only telemetry surface changes.

**Non-Goals:**
- Adding optional semconv attributes (`url.scheme`, `network.protocol.name`, `http.route`) — follow-up change if needed.
- Supporting both old and new names in parallel — the previous change isn't deployed anywhere we need to keep alive.
- Switching to exponential / native Prometheus histograms.
- Touching traces / logs.

## Decisions

### Pin a specific semconv version (`v1.26.0`)

Import `go.opentelemetry.io/otel/semconv/v1.26.0` and use its constants (`HTTPRequestMethodKey`, `HTTPResponseStatusCodeKey`, `ServerAddressKey`, `ServerPortKey`, `ErrorTypeKey`, `SchemaURL`). Don't type raw strings.

**Why:** Semconv strings have changed more than once (`http.method` → `http.request.method` was not long ago). Binding to constants means the compiler flags drift when we upgrade the semconv module; we get a diff to audit instead of silent attribute renames.

**Alternative considered:** hardcode the strings. Rejected — loses the compile-time signal.

### Delete the `*_total` counters

Drop `http_server_requests_total` and `upstream_requests_total` entirely. Consumers read request counts from the histograms' `_count` series (`http_server_request_duration_seconds_count`, `http_client_request_duration_seconds_count`).

**Why:** Semconv defines no standalone request counter; the histogram count is the canonical count. Keeping both is redundant cost for no signal.

**Trade-off:** every existing PromQL that referenced `*_requests_total` breaks. README gets an explicit table of query rewrites.

### Full status code, not bucketed class

Use `http.response.status_code` with the integer status as a string (`"200"`, `"404"`, `"502"`). Drop the `status_class` bucket.

**Why:** Semconv requires it. The real cardinality is ~20-30 distinct codes in practice — well below the threshold we worried about originally.

**Mitigation for backend-side aggregation:** dashboards that want the class can compute it in PromQL: `sum by (le) (rate(http_server_request_duration_seconds_bucket{http_response_status_code=~"5.."}[5m]))`.

### `"_OTHER"` bucket for unknown methods

Per semconv: only the nine canonical methods (`GET`, `HEAD`, `POST`, `PUT`, `DELETE`, `CONNECT`, `OPTIONS`, `TRACE`, `PATCH`) emit their literal name; everything else — lowercase methods, typos, weird custom verbs — emits the literal string `"_OTHER"`. We implement a tiny switch, not a semconv helper, because semconv Go SDK does not ship one.

**Why:** Closes the one cardinality hole in the attribute set. Attackers or buggy clients can't inflate label-set count by sending `req_method=FOO<N>` forever.

### `error.type` only on failure, closed-set values

- Success path: no `error.type` attribute at all.
- Timeout (`context.DeadlineExceeded` or `net.Error.Timeout()`): `"timeout"`.
- Connection refused: `"connection_refused"` (detected via `errors.Is(err, syscall.ECONNREFUSED)` where available, else string match as fallback).
- Everything else with an error: `"unknown"`.
- 5xx response *without* a transport error: no `error.type` — the status code carries the signal (this matches semconv guidance: `error.type` is for conditions the instrumented code couldn't complete, not HTTP-level errors the upstream returned cleanly).

**Why:** Semconv defines `error.type` as a low-cardinality enum-ish attribute and explicitly allows implementations to define their own vocabulary. Three values plus "unknown" is plenty for alerting and keeps the time series count sane.

**Alternative considered:** use Go error type names (`*net.OpError`, `*url.Error`). Rejected — those are too noisy and implementation-tied.

### `server.address` / `server.port` split

When `upstream_host` was `api.example.com:443` we now emit `server.address=api.example.com`, `server.port=443`. When no explicit port, emit only `server.address`. Parse via `net.SplitHostPort` with a fallback path for port-less hosts.

**Why:** Semconv treats address and port as separate attributes; joining them is a dashboarding hazard (e.g., grouping by host across ports).

### Version from `runtime/debug.ReadBuildInfo()`

Build a small helper in `internal/metrics`:

```go
func moduleVersion() string {
    info, ok := debug.ReadBuildInfo()
    if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
        return devFallback(info) // use vcs.revision setting if present, else "(devel)"
    }
    return info.Main.Version
}
```

Pass via `metric.WithInstrumentationVersion(moduleVersion())` when constructing the meter.

**Why:** Single source of truth. Released binaries (built with module proxy) report e.g. `v0.3.1`. Local `go run` reports `(devel)`. Binaries built from git with `-buildvcs=true` (default) report the VCS revision — which is more useful than `(devel)` for troubleshooting fleet drift.

**Alternative considered:** ldflags-injected version. Rejected as redundant with what the Go toolchain already records in build info.

### Keep the Prometheus exporter defaults

Don't add `otelprom.WithoutScopeInfo()` or `WithoutTargetInfo()`. Now that scope info carries real metadata, it's useful. If scrape noise becomes a complaint later, revisit as an operator-facing flag.

### Forward-looking: JWT and AWS STS metrics

This change intentionally only covers the HTTP surface. A follow-up will add instrumentation for the JWT lifecycle (token cache hits/misses, TTL, generation latency, refresh failures) and for the AWS STS calls that produce those tokens. To keep the instrumentation coherent across both changes, the following conventions apply to any future metric added to this proxy:

- **STS calls reuse HTTP client semconv.** STS is an outbound HTTPS call, so it is recorded on the same `http.client.request.duration` histogram introduced here, distinguished by `server.address="sts.<region>.amazonaws.com"`. A custom low-cardinality attribute `proxy.call_kind` with values `"upstream"` or `"sts"` MAY be added to that histogram when the follow-up lands, to let alerting split STS latency from upstream latency without regex on `server.address`.
- **JWT-lifecycle metrics are custom but semconv-styled.** No semconv vocabulary exists for token cache / credential operations, so new instruments are invented. They SHALL follow the same style rules we're locking in here:
  - Dotted namespace: `jwt.token.*` for the token lifecycle (`jwt.token.cache.hits`, `jwt.token.generation.duration`, `jwt.token.ttl`, ...); `aws.sts.*` only for STS-call-level facts not covered by HTTP client semconv.
  - Durations measured in seconds, histograms not counters-plus-histograms.
  - Attribute values drawn from closed sets.
  - Failure classification reuses the same `error.type` vocabulary defined in this change (`"timeout"`, `"connection_refused"`, `"unknown"`). New error classes are added centrally, not re-invented per instrument.

Deferred to the JWT change — not part of this one — are: the concrete instrument list, the `proxy.call_kind` attribute introduction, and any STS-specific attributes (role ARN, region) the token path needs.

## Risks / Trade-offs

- **[Risk]** PromQL written against the prior change's names will silently break. → **Mitigation:** the prior change isn't archived and hasn't been deployed to any long-lived scrape target; README gets a migration table; treat as breaking in the change-level BREAKING note.
- **[Risk]** `error.type` classification drifts between Go versions (e.g., new sentinel errors). → **Mitigation:** keep the classifier in one file with a unit test per known error type.
- **[Risk]** `debug.ReadBuildInfo()` returns `(devel)` for anyone running `go run` or `go build` without module awareness → labels become unhelpful. → **Mitigation:** fall back to the `vcs.revision` build setting (truncated to 12 chars) when `Main.Version` is `(devel)`; documented in README.
- **[Trade-off]** Higher series count from full `http.response.status_code` vs bucketed class. → **Accepted:** bounded in practice; dashboards reconstitute class with regex.
- **[Risk]** Semconv module version pin drifts when the transitive OTel SDK updates. → **Mitigation:** the semconv import is an explicit package path (`/v1.26.0`) — it cannot drift silently on `go get -u`.

## Migration Plan

No deployed consumers. Rollout is a single-step rename:
1. Ship the renamed metrics in one release.
2. Update README with a query-rewrite table so anyone who had started building dashboards against the old names has a clear diff.
3. Rollback: revert the commit; the prior metrics come back identically.
