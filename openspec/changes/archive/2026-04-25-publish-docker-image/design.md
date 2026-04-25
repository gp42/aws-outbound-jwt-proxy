## Context

The release workflow (`.github/workflows/release.yml`) is the single source of truth for "what is a release": it computes the next semver, builds `linux/amd64`, `linux/arm64`, and `darwin/arm64` binaries on native runners, uploads them as named artifacts (`aws-outbound-jwt-proxy-<version>-<os>-<arch>`), creates an annotated git tag, and publishes a GitHub Release with the binaries and a `SHA256SUMS` file.

We want a Docker Hub image to ship for every published release, but we do **not** want a parallel build path that could drift from the release binaries — the published container must contain the exact same bytes that the GitHub Release ships. Re-compiling inside the Docker workflow would defeat that.

## Goals / Non-Goals

**Goals:**
- One published Docker Hub image per GitHub Release, multi-arch (`linux/amd64` + `linux/arm64`) under a single tag.
- The image contains the **same binary** that is attached to the GitHub Release (verified by digest before `COPY` into the image).
- The publish step is **driven by the release workflow** — maintainers do not run anything extra.
- Docker Hub tag policy mirrors the release kind: stable → `vX.Y.Z` + `X.Y` + `X` + `latest`; prerelease → only `vX.Y.Z-rc.N`.
- Minimal image surface: `FROM scratch` (or `gcr.io/distroless/static`), non-root, single static binary, no shell.

**Non-Goals:**
- `darwin/arm64` images (Docker images are Linux-only in practice).
- Building a development/CI image with Go toolchain inside.
- GHCR or other registries — Docker Hub only for now.
- Image signing (cosign) and SBOM attestation — out of scope; tracked as future work.
- Custom entrypoint scripts, healthcheck binaries, or shell wrappers.

## Decisions

### D1. Trigger model: chained job in the release workflow, not `workflow_run`
The Docker publish runs as an additional job in `.github/workflows/release.yml` named `docker`, with `needs: [version, build, publish]`. It runs only when the `publish` job succeeds.

**Why over `workflow_run`:**
- `workflow_run` fires on workflow completion regardless of job success/failure semantics that downstream depends on, and requires re-resolving the version / artifact names from the upstream run via the API — fragile.
- A chained job has direct access to `needs.version.outputs.version` and the already-uploaded artifacts via `actions/download-artifact` in the same run.
- Keeps "a release is one workflow run" as a single mental model; failure of the docker step shows up on the same Actions page as the release.
- The user requested "release flow should trigger docker build and deploy" — a downstream job in the same workflow is the most direct interpretation and avoids cross-workflow plumbing.

**Alternative considered:** separate `docker-publish.yml` triggered by `release: { types: [published] }`. Rejected because (a) it forces re-deriving the version from the release tag, (b) artifacts would have to be re-downloaded from the Release page rather than the run's artifact store, and (c) failures land in a separate workflow run, splitting the release story.

### D2. Reuse the release artifacts; do not rebuild
The `docker` job downloads the two linux artifacts from the same workflow run (`aws-outbound-jwt-proxy-<version>-linux-amd64` and `…-linux-arm64`) using `actions/download-artifact`, then verifies them against the staged `SHA256SUMS` produced by the `publish` job before they enter the image.

**Why:** guarantees the image contents are byte-identical to the GitHub Release asset. If the docker job rebuilt with `go build`, drift between "the binary on the Release page" and "the binary in the image" becomes possible (different toolchain, different ldflags, different commit at build time on `workflow_run`).

### D3. Multi-arch via `docker buildx` with per-platform `--build-arg`
The Dockerfile is platform-agnostic and accepts a `BINARY` build-arg pointing to the already-extracted binary. The workflow runs `docker buildx build --platform linux/amd64,linux/arm64` with a build context that contains both binaries laid out under `dist/linux/amd64/aws-outbound-jwt-proxy` and `dist/linux/arm64/aws-outbound-jwt-proxy`, and uses `TARGETARCH` inside the Dockerfile to pick the right one.

**Why over building per-arch on native runners and pushing a manifest list manually:** buildx handles the manifest list, provenance, and cache; doing it by hand with `docker manifest create` is more code for no benefit here since we are not cross-compiling — the binaries already exist per-arch.

**Why this works without QEMU emulation:** the Dockerfile only does `COPY` and metadata; no `RUN` steps execute the binary. So buildx can produce both platform images from a single amd64 builder runner without QEMU.

### D4. Base image: `gcr.io/distroless/static-debian13:nonroot`
- No shell, no package manager, no busybox — minimal attack surface.
- `:nonroot` variant ships a `nonroot` user (UID 65532) and `/etc/passwd` entry, so the proxy runs as a non-root identity by default with no `RUN useradd` needed.
- Bundles up-to-date CA certificates, which the proxy needs for outbound TLS to AWS STS.
- Statically linked Go binary (`CGO_ENABLED=0`) is the canonical fit for `static-*` distroless.
- Pinned by digest (`@sha256:…`) per the repo's pinning convention; same principle applies to base images.

**Alternatives considered:**
- `alpine:3.21` — smaller download, ships a shell for `docker exec` debugging. Rejected: requires a `RUN apk add ca-certificates && adduser …` step (and therefore QEMU during multi-arch builds), and trades attack surface for shell access we do not need at runtime.
- `FROM scratch` — even smaller, but requires bundling CA certs by hand and creating `/etc/passwd` entries for non-root. More moving parts than distroless without meaningful benefit.

**Trade-off accepted:** no shell to `docker exec` into. Operators who need to inspect process state must rely on logs, metrics, and `kubectl debug` / `docker run --pid` patterns. This matches industry practice for production-grade Go service images.

### D5. Tagging policy
Computed in the workflow from `needs.version.outputs.version` and `needs.version.outputs.is_prerelease`:
- **Stable** (`is_prerelease == 'false'`): push `vX.Y.Z`, `vX.Y`, `vX`, `latest`.
- **Prerelease**: push only `vX.Y.Z-rc.N`. No `latest`, no truncated majors — prereleases must never be reachable via a floating tag.

Implemented with `docker/metadata-action` using semver flavor.

### D6. Credentials
Two new repo secrets: `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` (a Docker Hub access token scoped to push on the target repo). The job fails fast with a clear error if either is missing. Login uses `docker/login-action` pinned by SHA.

### D7. Image repository name
`gp42/aws-outbound-jwt-proxy` on Docker Hub, mirroring the GitHub repo path. Set as a workflow env var so it can be changed in one place.

### D8. Dual-registry publish (Docker Hub + GHCR)
The same `docker buildx build --push` invocation pushes to both `docker.io/gp42/aws-outbound-jwt-proxy` and `ghcr.io/gp42/aws-outbound-jwt-proxy` under identical tags. `docker/login-action` runs twice (once per registry); `docker/metadata-action` is configured with both image refs so the resulting tag list applies to both.

**Auth:**
- Docker Hub: `DOCKERHUB_USERNAME` + `DOCKERHUB_TOKEN` repo secrets.
- GHCR: workflow-scoped `GITHUB_TOKEN` with `packages: write` permission on the `docker` job.

**Why both:** GHCR is free for public images, has no pull rate limits for authenticated users, and is the natural home for a GitHub-hosted project; Docker Hub remains the default registry most users `docker pull` from without thinking. Publishing to both costs one extra `login` step and gives users a choice plus a fallback if one registry has an outage.

### D9. Documentation strategy
- **`docs/docker.md`** is the canonical user-facing container guide. It is rendered as a page on the existing GitHub Pages site (the repo already has a `site/` flow and `docs.yml` workflow).
- **Docker Hub Overview** is synced from `docs/docker.md` on every successful publish using `peter-evans/dockerhub-description@<sha>` (pinned). The action is conditional on stable releases only — prereleases do not overwrite the Overview.
- **GHCR** auto-renders `README.md` once the package is linked to the repo; no sync action needed.
- **`README.md`** gains a minimal "Container image" section: one `docker pull` example plus a link to the published `docs/docker.md` page on GitHub Pages. Detailed tag policy, supported platforms, labels, non-root notes, and verification go in `docs/docker.md` only — not duplicated in README.

**Why split docs:** a project README has a different audience than a registry landing page; cramming container usage into README clutters it, but a Docker Hub Overview that just says "see GitHub" is unfriendly. `docs/docker.md` is a single source synced to one place (Hub) and rendered for two audiences (Pages site, GHCR via repo link).

### D10. Action pinning
Every third-party action used by the new job — `docker/setup-buildx-action`, `docker/login-action` (used twice, once per registry), `docker/metadata-action`, `docker/build-push-action`, `actions/download-artifact`, `peter-evans/dockerhub-description` — is pinned `@<sha> # <version>` per the repo's standing rule. `docker/setup-qemu-action` is omitted since the Dockerfile has no `RUN` steps (see D3).

## Risks / Trade-offs

- **Risk:** Docker Hub credentials leak or token over-scoped. → **Mitigation:** use a Docker Hub access token scoped to push-only on the single target repo; store as repo secret, never as environment-level secret; document rotation in `RELEASING.md`.
- **Risk:** Release succeeds but docker push fails (Docker Hub outage), leaving GitHub Release without a matching image. → **Mitigation:** docker job runs after `publish`, so the GitHub Release is already created; the failure is visible on the run, and a manual re-run job (`workflow_dispatch` re-run of failed jobs) can retry. Document this in `RELEASING.md`.
- **Risk:** Artifact name drift between release `build` job and docker job. → **Mitigation:** specs/release-automation gains a requirement that artifact names follow `aws-outbound-jwt-proxy-<version>-linux-<arch>`; docker job also verifies via `SHA256SUMS` before COPY.
- **Trade-off:** Tying docker publish into the same workflow couples two concerns. Accepted because the alternative (separate workflow + `workflow_run`) is more code and more failure modes for a project at this scale.
- **Trade-off:** No cosign / SBOM at this stage. Accepted; can be added later without breaking the tag contract.

## Migration Plan

1. Land Dockerfile + workflow changes; the docker job is **gated on a repo variable** (`vars.DOCKER_PUBLISH_ENABLED == 'true'`) so merging the change does not require secrets to be present yet.
2. Maintainer creates the Docker Hub repo and access token, adds `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN` secrets and sets `DOCKER_PUBLISH_ENABLED=true`.
3. Cut a prerelease (`-rc.1`) from a branch to validate end-to-end: only the `-rc.1` tag should appear on Docker Hub, no `latest` movement.
4. Cut a stable release from `main`: verify `vX.Y.Z`, `vX.Y`, `vX`, and `latest` are all updated and point at the same multi-arch manifest digest.
5. **Rollback:** flip `DOCKER_PUBLISH_ENABLED=false` to disable future pushes; existing tags can be deleted from Docker Hub manually if a bad image was pushed (immutable-tags is not enabled).

## Open Questions

- Image labels (`org.opencontainers.image.*`) — include source, revision, version, licenses now? **Yes**, populated by `docker/metadata-action` defaults; cheap and useful.
- Do we need a non-`:nonroot` variant for users who must remap UIDs? **No** for v1; revisit if requested.
- Should `dockerhub-description` push the rendered HTML or the raw Markdown? **Raw Markdown** — Docker Hub renders Markdown natively; pushing raw keeps the source-of-truth single.
