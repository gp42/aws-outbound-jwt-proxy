# release-automation Specification

## Purpose

Manual-dispatch release workflow for the AWS Outbound JWT Proxy: how releases are triggered, how versions are computed from Conventional Commits, which platforms are built, and how artifacts and tags are published.

## Requirements

### Requirement: Release workflow trigger
The repository SHALL provide a GitHub Actions workflow that can be triggered manually via `workflow_dispatch` from any branch. The workflow SHALL NOT run automatically on push or pull request events.

#### Scenario: Dispatched from main produces stable release
- **WHEN** a maintainer triggers the release workflow with `ref=main`
- **THEN** the workflow computes the next stable semver, creates a GitHub Release (not marked prerelease), and updates the `latest` release pointer

#### Scenario: Dispatched from non-main branch produces prerelease
- **WHEN** a maintainer triggers the release workflow with `ref=<any-branch-other-than-main>`
- **THEN** the workflow computes the next version with an `-rc.N` suffix, creates a GitHub Release marked as prerelease, and does not update the `latest` release pointer

#### Scenario: Dispatched on a non-branch ref
- **WHEN** the workflow is dispatched on a ref that is not a branch (for example, a tag)
- **THEN** the workflow SHALL fail early with a clear error message

### Requirement: Semver computation via Conventional Commits
The workflow SHALL compute the next version using [`huggingface/semver-release-action`](https://github.com/huggingface/semver-release-action), which derives the bump from Conventional Commit messages since the last release tag.

#### Scenario: Feature commit bumps minor
- **WHEN** commits since the previous tag include at least one `feat:` commit and no breaking changes
- **THEN** the computed version bumps the minor component (e.g., `v1.2.3` → `v1.3.0`)

#### Scenario: Fix-only commits bump patch
- **WHEN** commits since the previous tag contain only `fix:`/non-feat conventional commits and no breaking changes
- **THEN** the computed version bumps the patch component (e.g., `v1.2.3` → `v1.2.4`)

#### Scenario: Breaking change bumps major
- **WHEN** any commit since the previous tag contains `BREAKING CHANGE:` in the footer or `!` after the type
- **THEN** the computed version bumps the major component (e.g., `v1.2.3` → `v2.0.0`)

#### Scenario: Prerelease candidate numbering
- **WHEN** the workflow is dispatched from a non-main branch and a prior `-rc.N` exists for the same base version
- **THEN** the next version increments the rc counter (e.g., `v1.3.0-rc.1` → `v1.3.0-rc.2`)

### Requirement: Multi-platform binary build
The workflow SHALL produce statically linked binaries for the following targets: `linux/amd64`, `linux/arm64`, `darwin/arm64`.

#### Scenario: All targets built per release
- **WHEN** the workflow runs to completion
- **THEN** one binary artifact per target triple SHALL be produced, each with `CGO_ENABLED=0` and stripped debug symbols

#### Scenario: Version embedded in binary
- **WHEN** a binary is built
- **THEN** the computed semver SHALL be injected via `-ldflags "-X"` into a build-time version variable, and `aws-outbound-jwt-proxy --version` SHALL print that version

### Requirement: Release asset publication
The workflow SHALL publish all built binaries as assets on the created GitHub Release, along with a `SHA256SUMS` checksums file covering every asset.

#### Scenario: Assets attached with deterministic names
- **WHEN** the release is published
- **THEN** each asset SHALL be named `aws-outbound-jwt-proxy-<version>-<os>-<arch>` (no platform-specific extension is required for the current target set) and SHALL appear on the Release page

#### Scenario: Checksums file present
- **WHEN** the release is published
- **THEN** a `SHA256SUMS` file listing sha256 of every other asset SHALL be attached to the release

### Requirement: Tag creation
The workflow SHALL create and push an annotated git tag matching the computed version (e.g., `v1.3.0` or `v1.3.0-rc.2`) pointing at the commit that was built.

#### Scenario: Tag points at built commit
- **WHEN** the workflow completes successfully
- **THEN** the tag exists on the remote and its target SHA equals the `GITHUB_SHA` used for the build

### Requirement: Release artifact contract for downstream consumers
The release workflow's per-target binary artifacts SHALL be uploaded with names of the form `aws-outbound-jwt-proxy-<version>-<os>-<arch>` and SHALL remain available within the same workflow run for downstream jobs to download. The `linux/amd64` and `linux/arm64` artifacts in particular constitute a stable contract consumed by the container image publish job.

#### Scenario: Linux artifacts available to docker job
- **WHEN** the release `build` matrix completes
- **THEN** the artifacts `aws-outbound-jwt-proxy-<version>-linux-amd64` and `aws-outbound-jwt-proxy-<version>-linux-arm64` SHALL be downloadable by other jobs in the same run via `actions/download-artifact`

#### Scenario: Artifact rename is a breaking change
- **WHEN** the artifact name pattern changes
- **THEN** the change SHALL be treated as breaking for the container image publish workflow and the `container-image-publishing` spec SHALL be updated in the same change

### Requirement: Release workflow triggers container image publish
A successful release run SHALL trigger publication of the corresponding container image. This SHALL be implemented as a downstream `docker` job within the release workflow itself, with `needs: [version, build, publish]`, so that one workflow run represents one complete release (binaries + git tag + GitHub Release + container image).

#### Scenario: Docker job runs as part of the release workflow
- **WHEN** a maintainer dispatches the release workflow and all upstream jobs succeed
- **THEN** the `docker` job in the same workflow run SHALL execute and publish the container image

#### Scenario: Docker failure does not invalidate the GitHub Release
- **WHEN** the `docker` job fails after the `publish` job has already created the GitHub Release and git tag
- **THEN** the GitHub Release and tag SHALL remain in place and the failure SHALL be visible on the workflow run for re-run
