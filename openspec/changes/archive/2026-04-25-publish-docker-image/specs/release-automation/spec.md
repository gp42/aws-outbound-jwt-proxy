## ADDED Requirements

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
