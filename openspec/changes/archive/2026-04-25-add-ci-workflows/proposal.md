## Why

The repository currently has only `release.yml` and `docs.yml` workflows. Pull requests and pushes to `main` are not exercised by automated checks, and dependency updates are not automated. This creates risk of regressions landing on `main` and lets third-party action SHAs and Go module versions drift unattended.

## What Changes

- Add a `ci.yml` GitHub Actions workflow that runs on pull requests and pushes to `main`, executing `go vet`, `go build`, and `go test ./...` against the Go toolchain pinned in `go.mod`.
- Run the test matrix on native-arch runners that mirror the release matrix (linux/amd64, linux/arm64, darwin/arm64) so test coverage matches what we ship.
- Add a Dependabot configuration (`.github/dependabot.yml`) that tracks updates for `gomod` and `github-actions` ecosystems on a weekly schedule, opening grouped PRs.
- Pin every new `uses:` reference to a commit SHA with a trailing `# vX.Y.Z` comment, consistent with the existing `release.yml` convention.

## Capabilities

### New Capabilities
- `ci-automation`: Continuous-integration workflow and dependency-update automation for the repository — what runs on which events, what must pass before merge, and how third-party updates are proposed.

### Modified Capabilities
<!-- None — these existing specs describe runtime proxy behavior, not CI. -->

## Impact

- New files: `.github/workflows/ci.yml`, `.github/dependabot.yml`.
- No changes to product code, `go.mod`, or the existing `release.yml` / `docs.yml` workflows.
- Future PRs will require the new CI checks to pass; Dependabot will begin opening update PRs once merged.
