## ADDED Requirements

### Requirement: Dockerfile contract
The repository SHALL provide a `Dockerfile` at the repo root that produces a minimal runtime image from a pre-built linux binary supplied via the build context. The Dockerfile SHALL NOT compile Go code and SHALL NOT contain `RUN` steps that execute the application binary.

#### Scenario: Image is built from prebuilt binary
- **WHEN** the Dockerfile is built with a build context that contains `dist/linux/<arch>/aws-outbound-jwt-proxy`
- **THEN** the resulting image SHALL contain that exact binary at `/usr/local/bin/aws-outbound-jwt-proxy` and SHALL set it as the entrypoint

#### Scenario: Image runs as non-root
- **WHEN** a container is started from the image with no user override
- **THEN** the process SHALL run as a non-root UID (65532) and SHALL NOT have a shell available in the image

#### Scenario: Image carries OCI metadata
- **WHEN** the image is inspected
- **THEN** it SHALL carry `org.opencontainers.image.source`, `org.opencontainers.image.revision`, `org.opencontainers.image.version`, and `org.opencontainers.image.licenses` labels populated from the release context

### Requirement: Multi-arch image
For every published release, a single multi-arch manifest SHALL be pushed covering `linux/amd64` and `linux/arm64`. `darwin/*` images are explicitly out of scope.

#### Scenario: Single tag resolves on both architectures
- **WHEN** a user runs `docker pull <registry>/gp42/aws-outbound-jwt-proxy:vX.Y.Z` on either an amd64 or arm64 host
- **THEN** Docker SHALL select the correct platform-specific image from the same manifest list

### Requirement: Dual-registry publication
Each release SHALL publish identical multi-arch images, under identical tags, to both `docker.io/gp42/aws-outbound-jwt-proxy` and `ghcr.io/gp42/aws-outbound-jwt-proxy`.

#### Scenario: Image pushed to Docker Hub
- **WHEN** the docker job runs to completion for a release
- **THEN** all computed tags SHALL be present on `docker.io/gp42/aws-outbound-jwt-proxy`

#### Scenario: Image pushed to GHCR
- **WHEN** the docker job runs to completion for a release
- **THEN** all computed tags SHALL be present on `ghcr.io/gp42/aws-outbound-jwt-proxy`

#### Scenario: Same digest on both registries
- **WHEN** the same release is pulled from Docker Hub and GHCR for the same architecture
- **THEN** the resolved image digests SHALL be identical

### Requirement: Image binary matches the GitHub Release asset
The binary embedded in the published image SHALL be byte-identical to the corresponding `aws-outbound-jwt-proxy-<version>-linux-<arch>` asset attached to the GitHub Release for that version.

#### Scenario: Release artifacts are reused, not rebuilt
- **WHEN** the docker job stages the build context
- **THEN** it SHALL download the linux binaries from the same workflow run's artifact store (not recompile them) and SHALL verify each binary's SHA-256 against the `SHA256SUMS` file produced by the release `publish` job before they are copied into the image

#### Scenario: Checksum mismatch aborts the publish
- **WHEN** a downloaded binary's SHA-256 does not match `SHA256SUMS`
- **THEN** the docker job SHALL fail before any image is pushed to any registry

### Requirement: Tag policy mirrors release kind
The set of tags pushed for a release SHALL be derived from the computed semver and the release kind:
- Stable release (`is_prerelease == false`): `vX.Y.Z`, `vX.Y`, `vX`, `latest`.
- Prerelease (`is_prerelease == true`): `vX.Y.Z-rc.N` only.

The same set of tags SHALL be pushed to both registries.

#### Scenario: Stable release updates floating tags
- **WHEN** the docker job completes for a stable release `v1.4.2`
- **THEN** the tags `v1.4.2`, `v1.4`, `v1`, and `latest` SHALL all resolve to the new manifest digest on both registries

#### Scenario: Prerelease does not update floating tags
- **WHEN** the docker job completes for a prerelease `v1.5.0-rc.1`
- **THEN** only the tag `v1.5.0-rc.1` SHALL be pushed; `latest`, `v1.5`, and `v1` SHALL NOT be modified

### Requirement: Release-driven trigger
The container image publish SHALL run as a downstream job (`docker`) of the existing release workflow, gated on successful completion of the `version`, `build`, and `publish` jobs of the same workflow run. There SHALL NOT be a separate manually-dispatched workflow that publishes images.

#### Scenario: Docker job runs only after release publishes
- **WHEN** the release workflow's `publish` job succeeds
- **THEN** the `docker` job in the same run SHALL execute

#### Scenario: Docker job is skipped when release fails
- **WHEN** any of the `version`, `build`, or `publish` jobs fail
- **THEN** the `docker` job SHALL NOT run

### Requirement: Credentials and permissions
Docker Hub authentication SHALL use the repository secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`. GHCR authentication SHALL use the workflow-scoped `GITHUB_TOKEN`. The `docker` job SHALL declare `permissions: { contents: read, packages: write }`.

#### Scenario: Missing Docker Hub secret fails fast
- **WHEN** the `docker` job runs and `DOCKERHUB_TOKEN` is empty
- **THEN** the job SHALL fail with a clear error before attempting any push

#### Scenario: Publish gate variable disables pushes
- **WHEN** the repository variable `DOCKER_PUBLISH_ENABLED` is not `true`
- **THEN** the `docker` job SHALL be skipped, and the release SHALL still succeed

### Requirement: Action and base-image pinning
Every third-party action used by the docker job, and the runtime base image, SHALL be pinned by commit SHA (actions) or image digest (base image), with a trailing comment containing the human-readable version.

#### Scenario: Pinned references in workflow
- **WHEN** a reviewer reads `.github/workflows/release.yml`
- **THEN** every `uses:` for a third-party action under the `docker` job SHALL be of the form `<owner>/<repo>@<40-char-sha> # <version>`

#### Scenario: Pinned base image in Dockerfile
- **WHEN** a reviewer reads the `Dockerfile`
- **THEN** the `FROM` line SHALL reference the base image by `@sha256:<digest>` with a trailing `# <tag>` comment

### Requirement: User documentation for the container image
The repository SHALL provide a Docker-focused user guide at `docs/docker.md` covering: registries (`docker.io`, `ghcr.io`), supported tags and tag policy, supported platforms, the non-root UID, OCI labels exposed, and how to verify an image's binary against the corresponding GitHub Release `SHA256SUMS`. This document SHALL be the single source of truth for container documentation.

#### Scenario: docs/docker.md is the source of truth
- **WHEN** any container-related documentation needs to change
- **THEN** the change SHALL be made in `docs/docker.md` and SHALL NOT be duplicated in other files

#### Scenario: README delegates to docs/docker.md
- **WHEN** a reader views `README.md`
- **THEN** the "Container image" section SHALL contain at most a single `docker pull` example and a link to the published `docs/docker.md` page on the project's GitHub Pages site, with no duplicated tag policy, label, or platform detail

#### Scenario: docs/docker.md is published on the site
- **WHEN** the existing site/Pages workflow runs
- **THEN** `docs/docker.md` SHALL be rendered as a page on the project's GitHub Pages site

### Requirement: Docker Hub Overview synchronization
After a successful **stable** release publish, the Docker Hub repository's "Overview" SHALL be updated from `docs/docker.md`. Prerelease publishes SHALL NOT modify the Overview.

#### Scenario: Stable release syncs Overview
- **WHEN** the docker job completes for a stable release
- **THEN** the contents of `docs/docker.md` from that release's commit SHALL be pushed to the Docker Hub repository description

#### Scenario: Prerelease does not sync Overview
- **WHEN** the docker job completes for a prerelease
- **THEN** the Docker Hub Overview SHALL NOT be modified
