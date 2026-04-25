## Why

The project has no automated release pipeline: binaries must be built locally via `make build-all`, and there is no enforced commit message format to drive versioning. We need reproducible, tagged releases with published multi-platform binaries, and a commit convention that can feed automated semver bumps.

## What Changes

- Add a GitHub Actions workflow (`workflow_dispatch`) that computes the next version using [`huggingface/semver-release-action`](https://github.com/huggingface/semver-release-action) (Conventional Commits → semver), builds binaries for `linux/amd64`, `linux/arm64`, `darwin/arm64`, and publishes them as assets on a GitHub Release.
- Support release-candidate flows by branch: `workflow_dispatch` from `main` produces a stable release; `workflow_dispatch` from any other branch produces a `-rc.N` prerelease, marked as a GitHub prerelease and not updating the `latest` tag.
- Add a local git `commit-msg` hook that validates Conventional Commits format and **rejects** non-conforming messages, plus a `make install-hooks` target and repo-tracked hook script so contributors opt in with a single command.
- Document the release process and commit convention in `docs/`.

## Capabilities

### New Capabilities
- `release-automation`: Automated building, versioning, and publishing of multi-platform binaries via GitHub Actions, gated on Conventional Commits.
- `commit-conventions`: Local enforcement of Conventional Commits via a git `commit-msg` hook.

### Modified Capabilities
<!-- None: existing specs cover runtime proxy behavior only -->

## Impact

- **New files**: `.github/workflows/release.yml`, `hack/hooks/commit-msg`, `RELEASING.md`, `COMMIT_CONVENTIONS.md`, `CONTRIBUTING.md` (top-level — `docs/` is reserved for the published end-user site).
- **Modified files**: `Makefile` (add `install-hooks`, `release-build` targets injecting version ldflags), `README.md` (link to new docs), `cmd/root.go` or a new `internal/version` package to expose build-time version.
- **Dependencies**: no new Go deps. CI consumes `actions/checkout`, `actions/setup-go`, `huggingface/semver-release-action`, `softprops/action-gh-release`.
- **Developer workflow**: contributors must run `make install-hooks` once; all commits must follow Conventional Commits or be rejected locally.
- **No runtime behavior change** for the proxy itself.
