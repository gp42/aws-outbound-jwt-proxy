## Context

The repo ships a Go binary built and released by `.github/workflows/release.yml`, which already pins all third-party actions by SHA and runs the build matrix on native-arch runners (`ubuntu-latest`, `ubuntu-24.04-arm`, `macos-latest`). There is no PR-time CI: nothing currently enforces that `go vet`, `go build`, or `go test ./...` pass before merge. Dependency updates (Go modules and GitHub Actions) are handled manually.

Repo conventions worth preserving:
- All `uses:` references are pinned to a 40-char commit SHA followed by a `# vX.Y.Z` comment (see release.yml).
- Per-arch builds use a matching native runner — never cross-compile from amd64.
- Go toolchain is read from `go.mod` via `actions/setup-go` with `go-version-file: go.mod` and `check-latest: true`.

## Goals / Non-Goals

**Goals:**
- PR and `main`-push runs that fail merges when vet/build/test fail.
- Test on the same OS/arch combinations we publish binaries for, so CI signal matches release reality.
- Weekly automated update PRs for `gomod` and `github-actions` ecosystems, grouped to reduce noise.
- Maintain SHA-pinning and native-runner conventions established in `release.yml`.

**Non-Goals:**
- Linting beyond `go vet` (no golangci-lint in this change).
- Coverage gates, race-detector matrices, or fuzz-test scheduling.
- Container image builds, SBOMs, or supply-chain signing.
- Auto-merge of Dependabot PRs.
- Required-status-check / branch-protection configuration (set in repo settings, not code).

## Decisions

### Decision 1: Single `ci.yml` workflow with a build-and-test matrix

Run vet/build/test in one matrix job rather than splitting into separate workflows. Same matrix shape as `release.yml`:

| os     | arch  | runner            |
| ------ | ----- | ----------------- |
| linux  | amd64 | ubuntu-latest     |
| linux  | arm64 | ubuntu-24.04-arm  |
| darwin | arm64 | macos-latest      |

**Why:** Mirrors release matrix so PR signal == release signal. `fail-fast: false` so a failure on one arch doesn't mask others. Native runners (no cross-compile) per repo convention.

**Alternative considered:** A single ubuntu-latest job for speed. Rejected because release builds on three runners; tests should too — otherwise an arm64- or darwin-only regression escapes until release time.

### Decision 2: Triggers — pull_request + push to main, no schedule

`on: pull_request` (default activity types) and `on: push: branches: [main]`. No nightly cron.

**Why:** Pre-merge check on PRs; post-merge check on main captures any merge-skew regressions. A scheduled run adds noise without catching anything the PR run wouldn't, since dependencies are tracked via Dependabot.

### Decision 3: Use `go test ./...` directly, not Make targets

Call `go vet ./...`, `go build ./...`, `go test ./...` directly in workflow steps rather than `make test`.

**Why:** Workflow stays self-describing; doesn't depend on Makefile targets staying stable. Matches `release.yml` style (uses `go build` directly, not `make build`).

### Decision 4: Dependabot config with two ecosystems, weekly, grouped

`.github/dependabot.yml` v2 with:
- `gomod` at `/`, schedule weekly, group all minor+patch into one PR (`go-deps`).
- `github-actions` at `/`, schedule weekly, group all minor+patch into one PR (`actions`).
- Major bumps left ungrouped (separate PRs for review).

**Why:** Weekly cadence matches a low-traffic single-binary repo. Grouping avoids PR-storm churn while keeping majors visible. SHA-pinned actions still get updated by Dependabot — it understands the `# vX.Y.Z` comment convention and updates both SHA and comment.

### Decision 5: Pin every new action by SHA

Every `uses:` in `ci.yml` uses `<sha> # vX.Y.Z`. Reuse the same SHAs already vetted in `release.yml` for `actions/checkout` and `actions/setup-go` to minimize new third-party trust footprint.

**Why:** Project-wide rule; Dependabot will keep them current.

## Risks / Trade-offs

- **arm64 / macOS runner availability or queue time** → Mitigation: matrix uses `fail-fast: false` so transient runner issues on one shard don't block the others; reruns are cheap.
- **Dependabot PR volume on a quiet repo** → Mitigation: weekly cadence + minor/patch grouping. Re-evaluate cadence if signal-to-noise drops.
- **CI minutes cost from 3-runner matrix on every PR** → Acceptable: the project is small, test suite is short, and release-parity outweighs the marginal cost.
- **Required-status-check names are not enforced by this change** → Out of scope; documented as a follow-up to configure in branch protection once job names stabilize.

## Migration Plan

1. Land `.github/workflows/ci.yml` and `.github/dependabot.yml` together in one PR.
2. Verify the PR itself triggers the new `ci.yml` and all matrix shards pass.
3. After merge, configure branch protection on `main` to require the new CI job(s) (manual repo-settings step, outside this change).
4. Rollback: revert the PR; no runtime or release-pipeline state is touched.

## Open Questions

- Should darwin/amd64 be added to the test matrix even though release.yml omits it? Current answer: no — keep CI matrix == release matrix; revisit if user reports surface darwin/amd64 issues.
