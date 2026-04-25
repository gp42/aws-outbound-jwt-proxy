---
icon: lucide/settings
---

# Configuration

All flags can also be set via environment variables. The env-var name is the uppercased flag name with dashes replaced by underscores (`--upstream-host` ↔ `UPSTREAM_HOST`). CLI flags take precedence when both are set.

## Flag reference

| Flag | Env var | Default | Purpose |
|---|---|---|---|
| `--listen-addr` | `LISTEN_ADDR` | `:8080` | host:port the server listens on |
| `--upstream-host` | `UPSTREAM_HOST` | (empty) | pinned upstream host; when set, the host header is ignored |
| `--upstream-scheme` | `UPSTREAM_SCHEME` | `https` | scheme used when forwarding (`http` or `https`) |
| `--host-header` | `HOST_HEADER` | `X-Upstream-Host` | request header read for the upstream host when unpinned |
| `--upstream-timeout` | `UPSTREAM_TIMEOUT` | `30s` | max time to wait for the upstream to send response headers |
| `--token-audience` | `TOKEN_AUDIENCE` | (empty; enables host-derived audience) | audience for the STS-issued JWT; repeat the flag for a multi-audience JWT, or set the env var to a comma-separated list. When omitted, the proxy derives a single-element audience per request from the outbound target host (see [Dynamic audience](dynamic-audience.md)) |
| `--token-signing-algorithm` | `TOKEN_SIGNING_ALGORITHM` | `RS256` | signing algorithm for STS-issued JWTs (`RS256` or `ES384`) |
| `--token-duration` | `TOKEN_DURATION` | `1h` | requested JWT lifetime (must be between `60s` and `1h`) |
| `--token-refresh-skew` | `TOKEN_REFRESH_SKEW` | `5m` | refresh cached tokens this far before their expiration (must be `> 0` and `< --token-duration`) |
| `--log-level` | `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |
| `--log-format` | `LOG_FORMAT` | `json` | `json` or `text` |
| `--tls-cert` | `TLS_CERT` | (empty) | path to PEM cert; enables HTTPS when set with `--tls-key` |
| `--tls-key` | `TLS_KEY` | (empty) | path to PEM key; enables HTTPS when set with `--tls-cert` |
| `--metrics-enabled` | `METRICS_ENABLED` | `true` | enable OpenTelemetry metrics + Prometheus scrape endpoint |
| `--metrics-listen-addr` | `METRICS_LISTEN_ADDR` | `:9090` | host:port for the metrics listener (must differ from `--listen-addr`) |
| `--metrics-path` | `METRICS_PATH` | `/metrics` | URL path where Prometheus metrics are served |

## AWS credentials and IAM

The proxy uses the standard AWS SDK default credentials chain (env vars, shared config, EC2/ECS/EKS role, etc.) and requires `sts:GetWebIdentityToken` for each audience it will request. There are no AWS-specific flags - set `AWS_REGION` / `AWS_PROFILE` / instance role as you normally would.

Under [dynamic audience mode](dynamic-audience.md) (`--token-audience` unset), the proxy will request tokens for every distinct upstream host it forwards to. Scope the IAM trust policy to the expected set of hosts rather than leaving it permissive - in shared-proxy deployments the target host is client-supplied, so a broad trust policy translates into broad token-minting capability.

## Choosing the upstream host

The proxy resolves the target host for each request in one of two modes.

### 1. Pinned upstream (sidecar pattern)

Set `--upstream-host` (and optionally `--upstream-scheme`). Every request is forwarded to that host regardless of any header on the inbound request.

```sh
aws-outbound-jwt-proxy \
  --upstream-host=api.example.com \
  --upstream-scheme=https
```

Best for one proxy per upstream. Clients cannot redirect traffic elsewhere.

### 2. Header-driven upstream (shared proxy pattern)

Leave `--upstream-host` empty. Each inbound request declares its target in a header (default `X-Upstream-Host`).

```sh
aws-outbound-jwt-proxy        # defaults: listens on :8080, header X-Upstream-Host

# client
curl -H "X-Upstream-Host: api.example.com" http://proxy:8080/v1/items
```

Customize the header name:

```sh
aws-outbound-jwt-proxy --host-header=X-Target
```

If `--host-header=Host` is set, the proxy reads the upstream from the standard HTTP `Host` line. Most HTTP clients derive `Host` from the URL and won't set it explicitly, so this only works with clients that support it (e.g. `curl --resolve`, Go's `req.Host`, etc.).

!!! tip "Pinned wins"
    If `--upstream-host` is set, the header is never consulted.

## TLS

To terminate TLS at the proxy, provide both `--tls-cert` and `--tls-key` (PEM-encoded file paths). Setting only one is an error.

```sh
aws-outbound-jwt-proxy \
  --listen-addr=:8443 \
  --tls-cert=/etc/proxy/cert.pem \
  --tls-key=/etc/proxy/key.pem \
  --upstream-host=api.example.com
```

Omit both flags to listen on plain HTTP.

For local testing, a self-signed cert is enough:

```sh
openssl req -x509 -newkey rsa:2048 -sha256 -days 1 -nodes \
  -keyout key.pem -out cert.pem -subj "/CN=localhost"
```
