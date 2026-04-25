---
icon: lucide/network
---

# Dynamic audience

When `--token-audience` is **not** configured (neither flag nor env var), the proxy selects the JWT audience per request from the outbound target URL - the scheme and host the request is actually being forwarded to, as determined by `--upstream-scheme` and either `--upstream-host` or the configured host header.

## Normalization

The scheme and host are lowercased and any port is stripped. For example, an HTTPS request being forwarded to `API.example.com:443` is minted with audience `https://api.example.com`.

## Intended deployment

This is intended for shared-proxy deployments where one proxy fronts many upstream hosts. The token cache keys on the normalized host, so there is one STS round-trip per distinct host followed by cached reuse.

## Caching and cardinality

The `token_cached_audiences` gauge reports the number of currently cached tokens - watch it for cardinality and growth in this mode, since the cache has no eviction. In deployments that forward to a bounded set of hosts this is a non-issue; in deployments where clients can supply arbitrary hosts, that cardinality is unbounded.

See [Operations](operations.md) for the full metrics catalog.

## IAM scoping

Under dynamic audience mode, the proxy will request tokens for every distinct upstream host it forwards to. Scope the IAM trust policy to the expected set of hosts rather than leaving it permissive - in shared-proxy deployments the target host is client-supplied, so a broad trust policy translates into broad token-minting capability.

## When `--token-audience` IS set

When `--token-audience` is set, the proxy always uses that static audience set regardless of target host. Dynamic derivation is skipped entirely.
