## Why

The proxy currently ships only as raw OS binaries on GitHub Releases. Most consumers run it inside containers (ECS, EKS, k8s, docker-compose), so every user re-implements the same `FROM scratch` / `COPY binary` recipe and has to wire their own image build into CI. Publishing an official multi-arch image to Docker Hub on each release removes that friction, gives users a stable supply-chain reference (digest + tag), and lets the project ship one canonical container artifact alongside the existing binaries.

## What Changes

- Add a `Dockerfile` at the repo root that produces a minimal runtime image from a pre-built linux binary (no in-image `go build`).
- Add a release-driven GitHub Actions job that publishes a multi-arch (`linux/amd64`, `linux/arm64`) image to **both** Docker Hub (`docker.io/gp42/aws-outbound-jwt-proxy`) and GitHub Container Registry (`ghcr.io/gp42/aws-outbound-jwt-proxy`) under identical tags.
- The publish step is **triggered by the existing release workflow** as a downstream job and **consumes the linux binary artifacts** already produced by the release `build` matrix — it does NOT recompile.
- Tagging on both registries mirrors the GitHub Release: stable releases get `vX.Y.Z`, `X.Y`, `X`, and `latest`; prereleases get only the exact `vX.Y.Z-rc.N` tag (no `latest`, no floating majors).
- Add Docker Hub credentials (`DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`) as required repository secrets and document them in `RELEASING.md`. GHCR auth uses the workflow-scoped `GITHUB_TOKEN` with `packages: write` — no extra secret.

## Capabilities

### New Capabilities
- `container-image-publishing`: Defines the Dockerfile contract, multi-arch image layout, dual-registry (Docker Hub + GHCR) tagging policy, and the release-triggered publish workflow.

### Modified Capabilities
- `release-automation`: The release workflow gains a downstream contract — its uploaded `linux/amd64` and `linux/arm64` binary artifacts MUST remain in their current naming scheme and retention so the docker-publish workflow can consume them, and a successful release run SHALL trigger the docker-publish workflow.

## Impact

- **New files**: `Dockerfile`, `docs/docker.md` (Docker-focused user guide, also published via the existing site/Pages flow and synced to Docker Hub Overview), `openspec/specs/container-image-publishing/spec.md`.
- **Modified files**: `.github/workflows/release.yml` (adds chained `docker` job), `RELEASING.md` (document Docker Hub secrets, GHCR linking, image rollback), `README.md` (minimal "Container image" section: one-line `docker pull` + link to the published `docs/docker.md` page on GitHub Pages — no detailed usage in README).
- **Secrets**: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN` must be added at the repo level before the first run. GHCR uses the built-in `GITHUB_TOKEN` (no extra secret).
- **Permissions**: the release workflow gains `packages: write` on the docker job so it can push to GHCR.
- **External**: A Docker Hub repository (`gp42/aws-outbound-jwt-proxy`) must exist with a push-scoped access token. The GHCR package is auto-created on first push and must be linked to the repo (visibility set to public) post-first-push.
- **No code changes** to the Go application.
