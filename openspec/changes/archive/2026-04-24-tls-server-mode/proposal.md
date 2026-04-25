## Why

Deployments that terminate TLS at the proxy itself (rather than behind a sidecar, Envoy, or load balancer) need the proxy to listen on HTTPS directly. This is common for standalone VMs and simple Kubernetes setups without a service mesh. The `initial-proxy-architecture` change already accounted for this; this change implements it.

## What Changes

- Add two CLI flags (with env-var fallback per existing config convention): `--tls-cert` and `--tls-key`, each pointing to a PEM file path.
- When **both** flags are set, the server SHALL listen on HTTPS (`ListenAndServeTLS`). When **neither** is set, the server listens on plain HTTP (current behavior).
- When **only one** of the two is set, the proxy SHALL refuse to start with a clear error — partial TLS config is always a misconfiguration.
- No changes to routing, header handling, or response behavior. This only affects the transport between client and proxy.

## Capabilities

### New Capabilities
<!-- None. TLS is a delta on an existing capability. -->

### Modified Capabilities
- `proxy-configuration`: Adds `--tls-cert` / `--tls-key` flags and the rule that both-or-neither must be set.

## Impact

- Code: `internal/config` (new fields, validation), `internal/server` (branch on whether TLS is configured). `cmd/root.go` unchanged structurally.
- No new runtime dependencies (stdlib `net/http` handles TLS natively).
- No change to client-facing request/response semantics; only the listener changes.
- Out of scope: client-authenticated TLS (mTLS), certificate reloading/rotation, ACME/Let's Encrypt integration. Can be separate changes later if needed.
