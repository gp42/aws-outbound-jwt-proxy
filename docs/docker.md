---
icon: lucide/container
---

# Container image

`aws-outbound-jwt-proxy` is published as a multi-architecture container image on every GitHub Release.

## Registries

The same image is pushed to two registries under identical tags:

- **Docker Hub:** `docker.io/gp42/aws-outbound-jwt-proxy`
- **GitHub Container Registry:** `ghcr.io/gp42/aws-outbound-jwt-proxy`

Either works. GHCR has no anonymous pull rate limits; Docker Hub is the default for most tooling.

## Quick start

```sh
docker pull gp42/aws-outbound-jwt-proxy:latest

docker run --rm \
  -p 8080:8080 \
  -e UPSTREAM_HOST=api.example.com \
  -e UPSTREAM_SCHEME=https \
  -e TOKEN_AUDIENCE=https://api.example.com \
  gp42/aws-outbound-jwt-proxy:latest
```

All proxy flags can also be set via environment variables — see the [Configuration reference](configuration.md).

## Supported platforms

Each tag resolves to a multi-arch manifest list covering:

- `linux/amd64`
- `linux/arm64`

`darwin` images are not published — use the [binary release](install.md) for macOS.

## Tag policy

Tags pushed for each release are derived from the release kind:

| Release kind                          | Tags pushed                              |
| ------------------------------------- | ---------------------------------------- |
| Stable release `vX.Y.Z`               | `vX.Y.Z`, `vX.Y`, `vX`, `latest`         |
| Prerelease `vX.Y.Z-rc.N`              | `vX.Y.Z-rc.N` only                       |

Prereleases never move floating tags (`latest`, `vX`, `vX.Y`). Pin to `vX.Y.Z` for reproducible deployments; pin to `vX.Y` to track patch updates only; `latest` always points at the most recent stable release on both registries.

## Image properties

- **Base image:** `gcr.io/distroless/static-debian13:nonroot` (pinned by digest in the [Dockerfile](https://github.com/gp42/aws-outbound-jwt-proxy/blob/main/Dockerfile)). No shell, no package manager, minimal attack surface.
- **User:** runs as UID `65532` (`nonroot`). The container does not require root and cannot escalate.
- **Entrypoint:** `/usr/local/bin/aws-outbound-jwt-proxy`.
- **Default port:** the proxy listens on the address configured via `LISTEN_ADDR` / `--listen-addr`. Map the host port accordingly.

## OCI labels

Each image carries the following `org.opencontainers.image.*` labels:

| Label                                | Value                                           |
| ------------------------------------ | ----------------------------------------------- |
| `org.opencontainers.image.title`     | `aws-outbound-jwt-proxy`                        |
| `org.opencontainers.image.version`   | the release tag, e.g. `v1.4.2`                  |
| `org.opencontainers.image.revision`  | the git commit SHA the image was built from     |
| `org.opencontainers.image.source`    | `https://github.com/gp42/aws-outbound-jwt-proxy`|
| `org.opencontainers.image.licenses`  | `MIT`                                           |

Inspect with:

```sh
docker inspect --format '{{json .Config.Labels}}' gp42/aws-outbound-jwt-proxy:vX.Y.Z | jq
```

## Source

- [Dockerfile](https://github.com/gp42/aws-outbound-jwt-proxy/blob/main/Dockerfile)
- [Release workflow](https://github.com/gp42/aws-outbound-jwt-proxy/blob/main/.github/workflows/release.yml) — the `docker` job builds and pushes images using the same binaries that are attached to the GitHub Release.
