---
icon: lucide/shield-check
---

# AWS Outbound JWT Proxy

A proxy that implements [AWS outbound identity federation](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_outbound.html) by signing outbound HTTP requests from AWS workloads with short-lived JSON Web Tokens (JWTs) issued by AWS STS.

## What it does

Instead of having applications manage long-term API keys or passwords for third-party services, the proxy obtains a web identity token from AWS STS on the workload's behalf and injects it into outgoing requests. The external service verifies the JWT against AWS's public OIDC discovery keys (signature, expiration, audience, and subject) before granting access.

The proxy handles token acquisition, caching, and renewal transparently, so application code does not need to integrate with AWS STS directly.

## How it works

1. An AWS workload sends an HTTP request through the proxy.
2. The proxy calls AWS STS to obtain a signed JWT identifying the workload, using either the audience set configured via `--token-audience` or, if that flag is unset, an audience derived from the outbound target host (see [Dynamic audience](dynamic-audience.md)).
3. The proxy sets `Authorization: Bearer <jwt>` on the outbound request (replacing any inbound `Authorization` header) and forwards it.
4. The external service validates the JWT via AWS's OIDC discovery endpoint and authorizes the request.

Tokens are cached per audience set and reused until they near expiry, so the first request per unique audience set incurs a single STS round-trip while subsequent requests are served from memory. If token acquisition fails, the proxy returns `502 Bad Gateway` and does not forward the request upstream.

## Use cases

- Authenticating AWS workloads to SaaS platforms and external APIs.
- Accessing resources in other cloud providers without static credentials.
- Hybrid or multi-cloud scenarios requiring federated identity.
- Calling on-premises services from AWS-hosted applications.

## Where to go next

- [Install](install.md) - get the binary.
- [Quick start](quick-start.md) - run the proxy end-to-end against a local echo upstream.
- [Configuration](configuration.md) - every flag and environment variable.
- [Dynamic audience](dynamic-audience.md) - per-host audience derivation for shared-proxy deployments.
- [Operations](operations.md) - metrics, failure modes, token cache behavior.
- [Troubleshooting](troubleshooting.md) - common errors and how to diagnose them.
