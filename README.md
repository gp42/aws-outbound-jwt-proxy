# aws-outbound-jwt-proxy

[![CI](https://github.com/gp42/aws-outbound-jwt-proxy/actions/workflows/ci.yml/badge.svg)](https://github.com/gp42/aws-outbound-jwt-proxy/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/gp42/aws-outbound-jwt-proxy)](https://github.com/gp42/aws-outbound-jwt-proxy/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/gp42/aws-outbound-jwt-proxy)](go.mod)
[![Docker Hub](https://img.shields.io/docker/pulls/gp42/aws-outbound-jwt-proxy?logo=docker&label=docker%20hub)](https://hub.docker.com/r/gp42/aws-outbound-jwt-proxy)
[![GHCR](https://img.shields.io/badge/ghcr.io-gp42%2Faws--outbound--jwt--proxy-2088FF?logo=github)](https://github.com/gp42/aws-outbound-jwt-proxy/pkgs/container/aws-outbound-jwt-proxy)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

A proxy for AWS workloads that call third-party APIs expecting a JWT instead of AWS SigV4. It implements [AWS outbound identity federation](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_outbound.html): outbound HTTP requests are signed with short-lived JSON Web Tokens issued by AWS STS, so applications do not need to manage long-term API keys.

## Documentation

Full user documentation is published at **<https://ops42.org/aws-outbound-jwt-proxy/>**:

- [Overview](https://ops42.org/aws-outbound-jwt-proxy/)
- [Install](https://ops42.org/aws-outbound-jwt-proxy/install/)
- [Quick start](https://ops42.org/aws-outbound-jwt-proxy/quick-start/)
- [Configuration reference](https://ops42.org/aws-outbound-jwt-proxy/configuration/) - every flag and env var
- [Dynamic audience](https://ops42.org/aws-outbound-jwt-proxy/dynamic-audience/)
- [Operations](https://ops42.org/aws-outbound-jwt-proxy/operations/) - metrics, failure modes
- [Troubleshooting](https://ops42.org/aws-outbound-jwt-proxy/troubleshooting/)

The Markdown sources live under [`docs/`](./docs).

For contributors and maintainers:

- [Contributing](./CONTRIBUTING.md) - fork, branch, test, open a PR
- [Commit conventions](./COMMIT_CONVENTIONS.md) - Conventional Commits + commit-msg hook
- [Releasing](./RELEASING.md) - how the GitHub Actions release workflow works

## What it does

Instead of having applications manage long-term API keys or passwords for third-party services, the proxy obtains a web identity token from AWS STS on the workload's behalf and injects it into outgoing requests. The external service verifies the JWT against AWS's public OIDC discovery keys (signature, expiration, audience, and subject) before granting access.

The proxy handles token acquisition, caching, and renewal transparently, so application code does not need to integrate with AWS STS directly.

## How it works

1. An AWS workload sends an HTTP request through the proxy.
2. The proxy calls AWS STS to obtain a signed JWT identifying the workload, using either the audience set configured via `--token-audience` or, if that flag is unset, an audience derived from the outbound target host.
3. The proxy sets `Authorization: Bearer <jwt>` on the outbound request (replacing any inbound `Authorization` header) and forwards it.
4. The external service validates the JWT via AWS's OIDC discovery endpoint and authorizes the request.

Tokens are cached per audience set and reused until they near expiry. If token acquisition fails, the proxy returns `502 Bad Gateway` and does not forward the request upstream.

```
  ┌─────────────┐    plain HTTP     ┌──────────┐    HTTPS + Bearer JWT    ┌──────────────┐
  │ AWS         │ ────────────────▶ │  proxy   │ ──────────────────────▶  │ third-party  │
  │ workload    │                   │          │                          │ API          │
  └─────────────┘                   └────┬─────┘                          └──────────────┘
                                         │ AssumeRoleWithWebIdentity
                                         ▼
                                    ┌──────────┐
                                    │ AWS STS  │
                                    └──────────┘
```

## What the token looks like

A standard JWT (`header.payload.signature`, base64url-encoded), signed by AWS STS and verifiable against AWS's public OIDC keys. Decoded payload from a real run:

```json
{
  "aud": "https://api.example.com",
  "sub": "arn:aws:iam::123456789012:role/aws-reserved/sso.amazonaws.com/AWSReservedSSO_ExampleRole_0000000000000000",
  "https://sts.amazonaws.com/": {
    "aws_account": "123456789012",
    "org_id": "o-xxxxxxxxxx",
    "source_region": "us-east-1",
    "principal_id": "arn:aws:iam::123456789012:role/aws-reserved/sso.amazonaws.com/AWSReservedSSO_ExampleRole_0000000000000000"
  },
  "iss": "https://00000000-0000-0000-0000-000000000000.tokens.sts.global.api.aws",
  "exp": 1777056976,
  "iat": 1777056076,
  "jti": "00000000-0000-0000-0000-000000000000"
}
```

See the [quick start](https://ops42.org/aws-outbound-jwt-proxy/quick-start/#what-the-token-looks-like) for the full payload including org-unit path and session metadata.

## Why not …?

- **IAM Roles Anywhere / SigV4** - works only when the third party speaks AWS SigV4. Most SaaS APIs want OIDC/JWT.
- **Direct OIDC from the app** - pushes STS calls, token caching, and refresh logic into every service. The proxy keeps that out of application code.
- **aws-vault / static API keys** - long-lived secrets to rotate and leak. STS-issued JWTs are short-lived and scoped per audience.

## Quick start

> **Prerequisite:** outbound identity federation must be enabled on the AWS account (IAM console → *Identity providers* → *Outbound federation* → **Enable**). It is opt-in per account; until enabled, `sts:GetWebIdentityToken` returns `AccessDenied`. See [AWS docs](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_outbound.html).

**Run with Docker:**

```sh
docker run --rm -p 8080:8080 \
  -e AWS_REGION=us-east-1 \
  -e UPSTREAM_HOST=api.example.com \
  -e UPSTREAM_SCHEME=https \
  -e TOKEN_AUDIENCE=https://api.example.com \
  gp42/aws-outbound-jwt-proxy:latest
```

**Or build from source:**

```sh
make build

./bin/aws-outbound-jwt-proxy \
  --upstream-host=api.example.com \
  --upstream-scheme=https \
  --token-audience=https://api.example.com
```

Pre-built binaries for Linux, macOS, and Windows on amd64/arm64 are also available on the [Releases](https://github.com/gp42/aws-outbound-jwt-proxy/releases/latest) page.

See the [Quick start guide](https://ops42.org/aws-outbound-jwt-proxy/quick-start/) for an end-to-end local walkthrough using a built-in echo upstream.

## Configuration

All flags can also be set via environment variables (`--upstream-host` ↔ `UPSTREAM_HOST`). See the [Configuration reference](https://ops42.org/aws-outbound-jwt-proxy/configuration/) for the full flag list, AWS credential handling, upstream-host resolution modes, and TLS setup.

## Container image

Multi-arch images (`linux/amd64`, `linux/arm64`) are published to Docker Hub and GHCR on every release.

```sh
docker run --rm -p 8080:8080 \
  -e UPSTREAM_HOST=api.example.com \
  -e UPSTREAM_SCHEME=https \
  -e TOKEN_AUDIENCE=https://api.example.com \
  gp42/aws-outbound-jwt-proxy:latest
```

See the [Container image guide](https://ops42.org/aws-outbound-jwt-proxy/docker/) for registries (Docker Hub + GHCR), supported tags, and platforms.

## License

See [LICENSE](./LICENSE).
