---
icon: lucide/rocket
---

# Quick start

This walkthrough runs the proxy against a local echo upstream so you can see the injected `Authorization: Bearer <jwt>` end-to-end, and inspect a real token minted by AWS STS.

## 1. Set up AWS credentials

The proxy uses the [default AWS SDK credentials chain](https://docs.aws.amazon.com/sdkref/latest/guide/standardized-credentials.html) (env vars, `~/.aws/config`/`credentials`, instance/pod role, etc.). For this walkthrough, any credentials that can call `sts:GetWebIdentityToken` work - e.g. an IAM user with a short-lived access key, or an assumed role.

Pick **one** of the following:

=== "Environment variables"

    ```sh
    export AWS_ACCESS_KEY_ID=AKIA...
    export AWS_SECRET_ACCESS_KEY=...
    export AWS_REGION=us-east-1
    ```

=== "Named profile"

    ```sh
    export AWS_PROFILE=my-dev-profile
    export AWS_REGION=us-east-1
    ```

=== "Already on AWS"

    Running on EC2, ECS, or EKS with an instance/task/pod role? The SDK picks it up automatically - just make sure `AWS_REGION` is set.

Sanity-check the chain resolves and the principal can mint tokens:

```sh
aws sts get-caller-identity
aws sts get-web-identity-token --audience https://example.test
```

The second call should return a JSON document containing a `Token` field. If it returns `AccessDenied`, update the principal's IAM policy to allow `sts:GetWebIdentityToken` for the audiences you plan to use; see [Troubleshooting](troubleshooting.md#502-bad-gateway-with-body-token-unavailable) for the full failure playbook.

## 2. Start the echo upstream

`hack/echo-upstream` is a tiny server that logs each inbound request's `Host` and `Authorization` headers and echoes all request headers back in the response body. We'll use it to see exactly what the proxy sends.

```sh
# terminal 1
go run ./hack/echo-upstream     # listens on :8081
```

## 3. Mint and inspect a token

Start the proxy pinned at the echo upstream with a static audience, then send a request through it:

```sh
# terminal 2
go run . \
  --upstream-host=127.0.0.1:8081 \
  --upstream-scheme=http \
  --token-audience=https://example.test
```

```sh
# terminal 3
curl -s http://127.0.0.1:8080/anything
```

The echo server's stdout should now include an `Authorization="Bearer …"` line, and curl's response body repeats the `Authorization` header as the upstream saw it.

### What the token looks like

A web identity token is a standard JWT (`header.payload.signature`, base64url-encoded). Decoded payload from a real run:

```json
{
  "aud": "http://127.0.0.1",
  "sub": "arn:aws:iam::123456789012:role/aws-reserved/sso.amazonaws.com/AWSReservedSSO_ExampleRole_0000000000000000",
  "https://sts.amazonaws.com/": {
    "org_id": "o-xxxxxxxxxx",
    "aws_account": "123456789012",
    "ou_path": [
      "o-xxxxxxxxxx/r-xxxx/ou-xxxx-xxxxxxxx/ou-xxxx-xxxxxxxx/"
    ],
    "original_session_exp": "2026-04-25T02:41:15Z",
    "source_region": "us-east-1",
    "principal_id": "arn:aws:iam::123456789012:role/aws-reserved/sso.amazonaws.com/AWSReservedSSO_ExampleRole_0000000000000000",
    "identity_store_user_id": "00000000-0000-0000-0000-000000000000"
  },
  "iss": "https://00000000-0000-0000-0000-000000000000.tokens.sts.global.api.aws",
  "exp": 1777056976,
  "iat": 1777056076,
  "jti": "00000000-0000-0000-0000-000000000000"
}
```

Key claims to verify upstream:

- `iss` - an STS tokens endpoint under `*.tokens.sts.global.api.aws`. Fetch signing keys from `<iss>/.well-known/openid-configuration`.
- `aud` - matches `--token-audience` (or the derived `<scheme>://<host>` under [dynamic audience](dynamic-audience.md)). Repeat `--token-audience` to mint a multi-audience token.
- `sub` / `principal_id` - ARN of the calling principal, useful for authorization rules on the receiving side.
- `https://sts.amazonaws.com/` - AWS-specific claims block with account, org, OU path, and source region for fine-grained policy on the verifier.
- `exp` - `iat + --token-duration` (default 1h). The proxy refreshes automatically `--token-refresh-skew` (default 5m) ahead of expiry.
- `jti` - unique token id; upstreams can use it for replay protection.

To decode the live token yourself, pipe the echo response through a JWT decoder:

```sh
curl -s http://127.0.0.1:8080/anything \
  | grep -Eo 'Bearer [^"]+' | cut -d' ' -f2 \
  | cut -d. -f2 | base64 -d 2>/dev/null | jq .
```

!!! note "Inbound Authorization is replaced"
    Any `Authorization` header on the inbound request is stripped before the proxy injects its own bearer token.

### Failure path

To see what happens when token acquisition fails, restart the proxy with bogus credentials:

```sh
AWS_ACCESS_KEY_ID=bogus \
AWS_SECRET_ACCESS_KEY=bogus \
AWS_REGION=us-east-1 \
go run . \
  --upstream-host=127.0.0.1:8081 \
  --upstream-scheme=http \
  --token-audience=https://example.test
```

The same `curl` now returns `502 Bad Gateway` with body `token unavailable`. The upstream is **not** called when token acquisition fails.

## 4. Check metrics

The proxy exposes OpenTelemetry/Prometheus metrics on a separate listener (default `:9090`):

```sh
curl -s http://127.0.0.1:9090/metrics | head
```

See [Operations](operations.md) for the full metric catalog.

## Next steps

- [Configuration](configuration.md) - all flags and environment variables.
- [Dynamic audience](dynamic-audience.md) - run without `--token-audience` to derive audiences per upstream host.
- [Troubleshooting](troubleshooting.md) - common errors and how to diagnose them.
