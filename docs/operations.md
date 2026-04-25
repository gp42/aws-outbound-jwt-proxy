---
icon: lucide/activity
---

# Operations

## Metrics endpoint

The proxy emits OpenTelemetry metrics on a **separate listener** (default `:9090`) so scrape traffic is isolated from the forwarding path: no JWT is attached, no upstream is resolved, and proxy TLS is not applied.

```sh
curl -s http://localhost:9090/metrics | head
```

Metric names and attributes follow the OpenTelemetry HTTP [semantic conventions](https://opentelemetry.io/docs/specs/semconv/http/) (v1.26.0).

## Instruments

### `http_server_request_duration_seconds`
Histogram of inbound request latency. Attributes: `http_request_method`, `http_response_status_code`. Use the histogram's `_count` series to count requests; there is no separate request counter.

### `http_server_active_requests`
Gauge of concurrent in-flight requests.

### `http_client_request_duration_seconds`
Histogram of outbound upstream call latency. Attributes:

- `server_address`, `server_port` (omitted when the port is the scheme default)
- `error_type` (only on failures)
- `token_result` - `ok`, `fetch_error`, or `resolver_error` - so a single panel can distinguish upstream failures from token-acquisition failures.

### `token_fetch_count_total`
Counter of AWS STS `GetWebIdentityToken` calls. Attributes: `audience` (normalized comma-joined set), `result` (`ok` or `error`), and `error_class` on errors (AWS error code, or `transport`).

### `token_cache_hit_count_total` / `token_cache_miss_count_total`
Counters keyed by `audience` (normalized comma-joined set).

### `token_cached_audiences`
Gauge of distinct audience sets currently held in the token cache. Watch this under [dynamic audience](dynamic-audience.md) mode - the cache has no eviction.

### Go runtime metrics
Standard `go_*` instruments.

## Attribute rules

- `http_request_method` uses the nine canonical HTTP methods literally; anything else collapses to `_OTHER`, per semconv.
- `http_response_status_code` is the full integer status (no class bucketing); write PromQL like `{http_response_status_code=~"5.."}` for the class.
- `error_type` values: `timeout`, `connection_refused`, `unknown`. The attribute is **absent** on success - do not query for `error_type="success"`.
- `token_result` values: `ok` on success, `fetch_error` when STS fails, `resolver_error` when the audience resolver fails. Always present on `http_client_request_duration_seconds`.
- `audience` on `token_*` series is the normalized (sorted, deduped, comma-joined) audience set, so `a,b` and `b,a` share a time series.
- Request path is never emitted as an attribute.

Every series carries scope metadata: `otel_scope_name`, `otel_scope_version` (populated from `runtime/debug.ReadBuildInfo()` - module version in release builds, VCS revision in dev, `(devel)` otherwise), and `otel_scope_schema_url`.

## Failure modes

- **STS token fetch fails** → proxy returns `502 Bad Gateway` with body `token unavailable`. Upstream is **not** called.
- **Audience resolver fails** (dynamic mode only, unusual) → proxy returns `502 Bad Gateway`. Upstream is **not** called.
- **Upstream timeout** (`--upstream-timeout`, default `30s`) → proxy returns a 5xx and records `error_type=timeout` on `http_client_request_duration_seconds`.
- **Upstream connection refused / other transport failure** → `error_type=connection_refused` or `unknown`.

## Token cache behavior

Tokens are cached per audience set and reused until they near expiry. "Near expiry" is controlled by `--token-refresh-skew` (default `5m`) relative to `--token-duration` (default `1h`): a token is proactively refreshed when its remaining life drops below the skew.

Cache hits and misses are tracked by `token_cache_hit_count_total` and `token_cache_miss_count_total`. The first request per unique audience set incurs a single STS round-trip; subsequent requests are served from memory.

## Endpoint safety

The metrics endpoint carries no authentication. Bind it to loopback or a private CIDR, or gate it with a NetworkPolicy. Set `--metrics-enabled=false` to disable it entirely.
