## 1. Dockerfile

- [x] 1.1 Resolve a current `gcr.io/distroless/static-debian13:nonroot` digest and record both digest and tag in a comment in the Dockerfile
- [x] 1.2 Write `Dockerfile` at repo root: pinned `FROM ...@sha256:...`, `ARG TARGETARCH`, `COPY dist/linux/${TARGETARCH}/aws-outbound-jwt-proxy /usr/local/bin/aws-outbound-jwt-proxy`, `USER 65532:65532`, `ENTRYPOINT ["/usr/local/bin/aws-outbound-jwt-proxy"]`, no `RUN` steps
- [x] 1.3 Add `.dockerignore` so only the staged `dist/` tree (and nothing else) enters the build context
- [x] 1.4 Verify locally: `docker buildx build --platform linux/amd64,linux/arm64` succeeds with a hand-staged `dist/` tree built from `go build`

## 2. Release workflow — `docker` job

- [x] 2.1 Add the `docker` job to `.github/workflows/release.yml` with `needs: [version, build, publish]`, `runs-on: ubuntu-latest`, and `permissions: { contents: read, packages: write }`
- [x] 2.2 Gate the job with `if: vars.DOCKER_PUBLISH_ENABLED == 'true'`
- [x] 2.3 Checkout step (pinned `actions/checkout` SHA already used elsewhere in the file)
- [x] 2.4 Download both linux artifacts via `actions/download-artifact` (pinned SHA), naming each one explicitly so amd64 and arm64 land in separate paths
- [x] 2.5 Stage `dist/linux/amd64/aws-outbound-jwt-proxy` and `dist/linux/arm64/aws-outbound-jwt-proxy` from the downloaded artifacts
- [x] 2.6 Re-download `SHA256SUMS` (re-stage it from the `publish` job's flow, or recompute it in this job from the downloaded artifacts and assert equality with what `publish` staged) and verify both linux binaries against it; fail the job on mismatch
- [x] 2.7 Set up Buildx via `docker/setup-buildx-action` (pinned SHA + version comment)
- [x] 2.8 Log into Docker Hub via `docker/login-action` (pinned) using `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN`; fail fast if either secret is empty
- [x] 2.9 Log into GHCR via `docker/login-action` (pinned) using `${{ github.actor }}` and `${{ secrets.GITHUB_TOKEN }}`
- [x] 2.10 Compute tag list with `docker/metadata-action` (pinned) configured with both image refs (`docker.io/gp42/aws-outbound-jwt-proxy`, `ghcr.io/gp42/aws-outbound-jwt-proxy`) and semver flavor; ensure prereleases produce only the exact `vX.Y.Z-rc.N` tag and stable releases produce `vX.Y.Z`, `vX.Y`, `vX`, `latest`
- [x] 2.11 Push the multi-arch image with `docker/build-push-action` (pinned), `platforms: linux/amd64,linux/arm64`, `push: true`, `provenance: false` (until provenance is explicitly designed), `tags:` and `labels:` from the metadata step
- [x] 2.12 After successful push, log a summary table to `$GITHUB_STEP_SUMMARY` listing every pushed tag on each registry and the final manifest digest
- [x] 2.13 Sync Docker Hub Overview from `docs/docker.md` via `peter-evans/dockerhub-description` (pinned), gated on `needs.version.outputs.is_prerelease == 'false'`

## 3. Documentation

- [x] 3.1 Write `docs/docker.md`: registries, supported tags + tag policy, supported platforms, non-root UID, OCI labels, verification recipe (`docker pull`, `docker inspect`, comparing digest/binary against the GitHub Release `SHA256SUMS`)
- [x] 3.2 Confirm the existing site/Pages workflow renders `docs/docker.md` without changes; if it doesn't pick up new files automatically, add it to the site nav/index in the minimum way the existing site flow expects
- [x] 3.3 Add a minimal "Container image" section to `README.md`: one-line `docker pull` example and a single link to the published `docs/docker.md` page on GitHub Pages — no tag policy, label, or platform detail
- [x] 3.4 Update `RELEASING.md`: required Docker Hub secrets, GHCR `packages: write` permission, the `DOCKER_PUBLISH_ENABLED` repo variable, how to roll back a bad tag on each registry, and the note that GitHub Releases are not invalidated by docker-job failure

## 4. First publish & validation — **deferred to maintainer (out-of-band)**

- [ ] 4.1 Maintainer creates the Docker Hub repo `gp42/aws-outbound-jwt-proxy` and a push-scoped access token; adds `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN` secrets and sets `vars.DOCKER_PUBLISH_ENABLED=true`
- [ ] 4.2 Cut a prerelease (`-rc.1`) from a non-`main` branch; verify only the exact `-rc.N` tag is pushed to both registries, no `latest` movement, and Docker Hub Overview is unchanged
- [ ] 4.3 Cut a stable release from `main`; verify `vX.Y.Z`, `vX.Y`, `vX`, and `latest` all resolve to the same multi-arch manifest digest on both registries, and the Docker Hub Overview matches `docs/docker.md`
- [ ] 4.4 Verify on each registry: `docker pull` works on amd64 and arm64 hosts, the binary inside the image matches the corresponding GitHub Release asset by SHA-256, the container starts and prints the version, and the process runs as UID 65532
- [ ] 4.5 Link the GHCR package to the repository and set its visibility to public

## 5. Spec sync — **run after merge**

- [ ] 5.1 After landing the change, run `openspec apply` (or the project's archive command) so the change's deltas are merged into `openspec/specs/container-image-publishing/spec.md` and `openspec/specs/release-automation/spec.md`
