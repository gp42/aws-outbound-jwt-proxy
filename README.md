# aws-outbound-jwt-proxy

A proxy that implements [AWS outbound identity federation](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_outbound.html) by signing outbound HTTP requests from AWS workloads with short-lived JSON Web Tokens (JWTs) issued by AWS STS.

## Documentation

Full user documentation is published at **<https://gp42.github.io/aws-outbound-jwt-proxy/>**:

- [Overview](https://gp42.github.io/aws-outbound-jwt-proxy/)
- [Install](https://gp42.github.io/aws-outbound-jwt-proxy/install/)
- [Quick start](https://gp42.github.io/aws-outbound-jwt-proxy/quick-start/)
- [Configuration reference](https://gp42.github.io/aws-outbound-jwt-proxy/configuration/) - every flag and env var
- [Dynamic audience](https://gp42.github.io/aws-outbound-jwt-proxy/dynamic-audience/)
- [Operations](https://gp42.github.io/aws-outbound-jwt-proxy/operations/) - metrics, failure modes
- [Troubleshooting](https://gp42.github.io/aws-outbound-jwt-proxy/troubleshooting/)

The Markdown sources live under [`docs/`](./docs).

For contributors and maintainers:

- [Contributing](./CONTRIBUTING.md) — fork, branch, test, open a PR
- [Commit conventions](./COMMIT_CONVENTIONS.md) — Conventional Commits + commit-msg hook
- [Releasing](./RELEASING.md) — how the GitHub Actions release workflow works

## What it does

Instead of having applications manage long-term API keys or passwords for third-party services, the proxy obtains a web identity token from AWS STS on the workload's behalf and injects it into outgoing requests. The external service verifies the JWT against AWS's public OIDC discovery keys (signature, expiration, audience, and subject) before granting access.

The proxy handles token acquisition, caching, and renewal transparently, so application code does not need to integrate with AWS STS directly.

## How it works

1. An AWS workload sends an HTTP request through the proxy.
2. The proxy calls AWS STS to obtain a signed JWT identifying the workload, using either the audience set configured via `--token-audience` or, if that flag is unset, an audience derived from the outbound target host.
3. The proxy sets `Authorization: Bearer <jwt>` on the outbound request (replacing any inbound `Authorization` header) and forwards it.
4. The external service validates the JWT via AWS's OIDC discovery endpoint and authorizes the request.

Tokens are cached per audience set and reused until they near expiry. If token acquisition fails, the proxy returns `502 Bad Gateway` and does not forward the request upstream.

## Quick start

```sh
# build
make build

# run (pinned upstream, static audience)
./bin/aws-outbound-jwt-proxy \
  --upstream-host=api.example.com \
  --upstream-scheme=https \
  --token-audience=https://api.example.com
```

See the [Quick start guide](https://gp42.github.io/aws-outbound-jwt-proxy/quick-start/) for an end-to-end local walkthrough using a built-in echo upstream.

## Configuration

All flags can also be set via environment variables (`--upstream-host` ↔ `UPSTREAM_HOST`). See the [Configuration reference](https://gp42.github.io/aws-outbound-jwt-proxy/configuration/) for the full flag list, AWS credential handling, upstream-host resolution modes, and TLS setup.

## License

See [LICENSE](./LICENSE).
