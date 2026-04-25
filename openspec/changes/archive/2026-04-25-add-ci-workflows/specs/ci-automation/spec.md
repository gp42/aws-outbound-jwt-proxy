## ADDED Requirements

### Requirement: CI workflow runs on pull requests and main pushes

The repository SHALL provide a GitHub Actions workflow at `.github/workflows/ci.yml` that triggers on `pull_request` events targeting any branch and on `push` events to the `main` branch. The workflow SHALL NOT run on tag pushes or on a schedule.

#### Scenario: Pull request opened
- **WHEN** a pull request is opened or updated against any branch
- **THEN** the CI workflow runs and reports a status check on the pull request

#### Scenario: Push to main
- **WHEN** a commit is pushed to `main` (including via merge)
- **THEN** the CI workflow runs against the resulting commit

#### Scenario: Tag push
- **WHEN** a tag (e.g. `v1.2.3`) is pushed
- **THEN** the CI workflow does NOT run

### Requirement: CI workflow runs vet, build, and test on a release-parity matrix

The CI workflow SHALL execute `go vet ./...`, `go build ./...`, and `go test ./...` for each of the following `(os, arch, runner)` combinations on a native-arch runner that matches the architecture, with `fail-fast: false`:

| os     | arch  | runner            |
| ------ | ----- | ----------------- |
| linux  | amd64 | ubuntu-latest     |
| linux  | arm64 | ubuntu-24.04-arm  |
| darwin | arm64 | macos-latest      |

The workflow SHALL use `actions/setup-go` with `go-version-file: go.mod` and `check-latest: true`. The workflow MUST NOT cross-compile (no `GOOS`/`GOARCH` overrides that differ from the runner's native arch).

#### Scenario: All matrix shards succeed
- **WHEN** vet, build, and test pass on every matrix shard
- **THEN** the workflow concludes successfully and the status check is green

#### Scenario: One matrix shard fails
- **WHEN** `go test ./...` fails on `linux/arm64` while the other shards pass
- **THEN** the other shards still complete (fail-fast disabled) and the overall workflow is reported as failed

#### Scenario: Go toolchain version
- **WHEN** the workflow installs Go
- **THEN** it reads the toolchain version from `go.mod` (not a hardcoded version) and uses `check-latest: true`

### Requirement: Third-party actions are pinned by commit SHA

Every `uses:` reference in `.github/workflows/ci.yml` SHALL be pinned to a 40-character commit SHA, followed by a trailing comment of the form `# vX.Y.Z` indicating the human-readable release tag. First-party `actions/*` SHAs SHOULD match those already vetted in `release.yml` where the same major version is used.

#### Scenario: Action reference format
- **WHEN** any step uses a third-party action
- **THEN** the `uses:` value matches `<owner>/<repo>@<40-char-sha> # vX.Y.Z`

#### Scenario: Float-tag reference rejected
- **WHEN** a `uses:` reference uses a moving tag (e.g. `actions/checkout@v6` or `@main`) without a SHA
- **THEN** the workflow does NOT meet this requirement

### Requirement: Dependabot configuration tracks gomod and github-actions

The repository SHALL provide `.github/dependabot.yml` (config version 2) with update entries for:

- `package-ecosystem: gomod`, `directory: "/"`, `schedule.interval: weekly`, with minor and patch updates grouped into a single PR group named `go-deps`.
- `package-ecosystem: github-actions`, `directory: "/"`, `schedule.interval: weekly`, with minor and patch updates grouped into a single PR group named `actions`.

Major-version bumps SHALL NOT be grouped (Dependabot opens them as individual PRs).

#### Scenario: Weekly Go module updates
- **WHEN** a new minor or patch version of any direct or indirect Go module dependency is released
- **THEN** Dependabot opens at most one combined PR per week in the `go-deps` group

#### Scenario: Weekly action updates
- **WHEN** new minor or patch versions of pinned GitHub Actions are released
- **THEN** Dependabot opens at most one combined PR per week in the `actions` group, updating both the SHA and the trailing `# vX.Y.Z` comment

#### Scenario: Major version bump
- **WHEN** a new major version of a tracked dependency is released
- **THEN** Dependabot opens a separate, ungrouped PR for that major bump
