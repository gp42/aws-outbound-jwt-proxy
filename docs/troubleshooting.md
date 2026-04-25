---
icon: lucide/life-buoy
---

# Troubleshooting

## `502 Bad Gateway` with body `token unavailable`

**Symptom:** every request through the proxy returns 502 `token unavailable`; the upstream logs show no incoming traffic.

**Cause:** AWS STS `GetWebIdentityToken` is failing, so the proxy never forwards the request.

**Diagnose:**

1. Check the proxy logs - the underlying STS error is logged at `error` level.
2. Check the `token_fetch_count_total` metric with `result="error"` and look at the `error_class` attribute. Common values:
   - AWS error code (e.g. `AccessDenied`, `InvalidIdentityToken`) - an IAM / STS configuration problem.
   - `transport` - the proxy could not reach STS (network/DNS/credentials chain problem).
3. Confirm the proxy's credentials chain resolves: set `AWS_REGION`, check `AWS_PROFILE` / instance role.
4. Confirm the STS trust policy grants `sts:GetWebIdentityToken` for the audience(s) the proxy is requesting.
5. Confirm **outbound identity federation is enabled on the AWS account**. It is opt-in per account; if disabled, every `GetWebIdentityToken` call returns `AccessDenied` regardless of IAM policy. An account admin enables it via the IAM console (*Identity providers* → *Outbound federation* → **Enable**), or org-wide from the AWS Organizations management account. See the [AWS outbound identity federation docs](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_outbound.html).

## `502 Bad Gateway` without `token unavailable` body

**Cause:** token acquisition succeeded but the upstream call failed.

**Diagnose:** look at `http_client_request_duration_seconds` with `error_type` attribute set:

- `error_type="timeout"` - raise `--upstream-timeout` or investigate slow upstream.
- `error_type="connection_refused"` - upstream is not listening, or `--upstream-host` / header value is wrong.
- `error_type="unknown"` - check proxy logs for the underlying transport error.

## Upstream receives a request with the wrong audience

**Symptom:** the upstream rejects the JWT because the `aud` claim doesn't match what it expects.

**Possible causes:**

1. You're in [dynamic audience mode](dynamic-audience.md) and the derived audience (`<scheme>://<host>`) doesn't match what the upstream expects - set `--token-audience` explicitly to the audience the upstream validates against.
2. `--upstream-scheme` is wrong - dynamic audience derives from the scheme the proxy uses to forward, not from the inbound request.
3. The upstream resolves through a different hostname than what the proxy forwards to - again, use an explicit `--token-audience`.

## The inbound `Authorization` header is missing at the upstream

**Expected behavior.** The proxy **replaces** any inbound `Authorization` header with its own `Bearer <jwt>`. If you need to forward an inbound credential, put it in a different header.

## Clients hit the proxy but the upstream never sees the request

Check:

1. `--upstream-host` is correctly set, or the request includes the header named by `--host-header` (default `X-Upstream-Host`).
2. `--upstream-scheme` matches what the upstream expects.
3. If using `--host-header=Host`, the HTTP client actually sets the `Host` line - most clients derive it from the URL.

## Cardinality explosion in `token_cached_audiences`

**Symptom:** the gauge keeps climbing.

**Cause:** the proxy is running in dynamic audience mode and clients are supplying new upstream hosts continuously. The cache has no eviction.

**Fix:** bound the set of upstreams - either pin with `--upstream-host`, set an explicit `--token-audience`, or reject unknown hosts at a layer in front of the proxy.

## `--tls-cert` set but the proxy errors on startup

Both `--tls-cert` **and** `--tls-key` must be set together. Setting only one is a configuration error.

## Metrics endpoint not reachable

- Confirm `--metrics-enabled=true` (the default).
- Confirm `--metrics-listen-addr` (default `:9090`) is not blocked by a firewall or NetworkPolicy.
- The metrics listener is a separate port from `--listen-addr`; they must differ.
