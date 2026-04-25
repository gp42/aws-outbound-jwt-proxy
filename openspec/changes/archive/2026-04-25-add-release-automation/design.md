## Context

`aws-outbound-jwt-proxy` is a Go CLI built with `cobra`. A `Makefile` already cross-compiles to `linux/amd64`, `linux/arm64`, `darwin/{amd64,arm64}`, `windows/amd64` via pure Go (`CGO_ENABLED=0`). There is no versioning, no tagging discipline, no CI, and no published artifacts. Contributors commit with arbitrary messages.

We want:
1. Manual, maintainer-gated releases (no surprise publishes).
2. Version numbers derived from commit history, not typed by hand.
3. Safe "preview" releases cut from topic branches without contaminating the `latest` tag.
4. A local gate that keeps commit history clean enough for semver derivation to work.

## Goals / Non-Goals

**Goals:**
- One workflow file, one command for contributors (`make install-hooks`), reproducible binaries.
- Branch-derived release kind: `main` → stable, anything else → `-rc.N` prerelease.
- Checksums file for supply-chain verification.
- Version string reachable at runtime (`--version`).

**Non-Goals:**
- Docker image publishing (out of scope for this change; can be added later).
- Signing binaries with cosign/GPG (future work; SHA256 sums only for now).
- Homebrew tap, apt/rpm repositories, or GoReleaser config (keep surface minimal; revisit if demand appears).
- Automatic releases on merge to main (explicitly rejected — maintainers retain gate).
- Windows or `darwin/amd64` binaries (dropped per maintainer decision).

## Decisions

### D1: Use `huggingface/semver-release-action` for version bump
**Why:** The maintainer named this action explicitly. It wraps `semantic-release`, consumes Conventional Commits, and supports prereleases via semantic-release's `branches` config. We use it in **`dryRun: true` mode** purely as a "what's the next version" oracle — the action's own release-creation path is intentionally bypassed so the workflow can build binaries between version computation and release publication.

**Action interface (verified against `action.yml` at the pinned tag):**
- Inputs: `branches` (JSON array of semantic-release branch configs, default `'["main"]'`), `dryRun` (default `'false'`), `commitAnalyzerPluginOpts`.
- Outputs: `tag` (e.g. `v1.2.3` or `v1.2.3-rc.1`), `version`, `changelog`, `released`.
- Reads `GITHUB_TOKEN` from env.
- Docker action — Linux runners only (matches our `ubuntu-latest`).

**Alternatives considered:**
- `go-semantic-release/action` — heavier, opinionated release notes; we want control over the release body.
- Hand-rolled `svu` invocation — adds a binary dependency and reimplements prerelease numbering.

### D2: Branch determines release kind (no workflow input)
**Why:** Maintainers directly control the release kind by choosing the dispatch branch. This removes a class of "clicked the wrong option" mistakes and keeps the workflow inputs empty. The workflow reads `github.ref_name`:
- `main` → pass `branches: '[{"name":"main"}]'` to the action; output is a stable `vX.Y.Z`.
- any other branch `B` → pass `branches: '[{"name":"main"},{"name":"<B>","prerelease":"rc"}]'`; semantic-release recognizes the dispatched branch as a prerelease channel and emits `vX.Y.Z-rc.N`, incrementing `N` against any prior rc on that base version.

The arbitrary-branch prerelease handling sidesteps the limitation that semantic-release requires prerelease branches to be enumerated in config: we synthesize the config per-dispatch from `github.ref_name`.

**Alternatives considered:**
- Workflow input `release_type` with enum — more flexible, but invites mistakes and duplicates branch information.
- Separate workflows for stable and prerelease — duplicated YAML with drift risk.
- Restrict prereleases to a fixed pattern like `rc/*` — simpler config but forces a branch-naming convention contributors don't currently use.

### D3: Single job with matrix build, then release job
**Why:** A `build` job with `strategy.matrix.target` cross-compiles each target on `ubuntu-latest` (pure Go, no need for native runners). A dependent `release` job computes the version, creates the tag, generates `SHA256SUMS`, and publishes via `softprops/action-gh-release`. Ordering: `release` runs *after* builds so failures abort before any tag is pushed.
**Sequencing detail:** The version is computed once (in the `release` job) and passed to the `build` job via `needs.version.outputs.version`. Concretely: `version` job → `build` matrix (embeds version in ldflags) → `publish` job (creates tag, uploads assets). This avoids each matrix shard independently computing a different version if commits race.

### D4: Version injection via `-ldflags -X`
**Why:** Standard Go idiom. A new `internal/version` package exposes `var Version = "dev"` and `cmd/root.go` gains a `--version`/`version` subcommand printing it. CI passes `-X github.com/gp42/aws-outbound-jwt-proxy/internal/version.Version=$VERSION`. Local `make build` leaves the default `dev` value.

### D5: Hook lives under `hack/hooks/`, installed by `make install-hooks`
**Why:** Tracked in the repo so everyone uses the same version, but not active until opt-in. `make install-hooks` symlinks `.git/hooks/commit-msg → ../../hack/hooks/commit-msg` (relative symlink, survives `git clone` into a different absolute path). Fallback to copy when the filesystem lacks symlink support.
**Alternatives considered:**
- `core.hooksPath=hack/hooks` via `make install-hooks` — cleaner (no symlink), but any other hooks added later must also live in `hack/hooks/`. Acceptable; we'll go with this approach because it's simpler than symlinking and scales to future hooks. **Revised decision: use `git config core.hooksPath hack/hooks`.**
- Third-party managers (lefthook, pre-commit) — added dependency for a single hook.

### D6: Hook is POSIX shell, not Go or Node
**Why:** No toolchain assumption beyond `sh` + `grep -E`. Regex source of truth: `^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-z0-9._/-]+\))?!?: .+`. Merge and revert commits are passed through via a leading-line check before the regex.

### D7: Checksums
**Why:** `sha256sum aws-outbound-jwt-proxy-* > SHA256SUMS` in the publish job, attached as an asset. Cheap, verifiable by users, no infrastructure.

## Risks / Trade-offs

- **Risk:** A maintainer dispatches from `main` when they meant to cut an rc. → **Mitigation:** the workflow prints the computed release kind in a loud summary step before publishing; a failed dry-run is cheaper than a bad tag.
- **Risk:** A developer bypasses the hook with `git commit --no-verify`. → **Mitigation:** Accepted. The semver-release-action handles non-conforming commits (treats them as no-op); worst case is a patch bump when a minor was intended. We document that `--no-verify` defeats the gate.
- **Risk:** `huggingface/semver-release-action` could change behavior on a floating tag. → **Mitigation:** pin to a specific tag (e.g., `@v1.2.0`) — never `@main`.
- **Risk:** Prerelease numbering resets unexpectedly if the action interprets the history differently across branches. → **Mitigation:** documented scenario in specs and a CI smoke test that dry-runs the version calculator on a fixture repo can be added later.
- **Risk:** Two matrix shards compute slightly different versions if dispatched on a fast-moving branch. → **Mitigation:** D3 sequencing (version job first, pass via `needs`).
- **Trade-off:** No release notes generation in this change. The GitHub Release body will contain only the autogenerated "compare" link. Follow-up change can add Conventional Commit → changelog rendering.

## Migration Plan

1. Land this change with workflow, hook, docs.
2. Run `make install-hooks` in maintainer clones.
3. Dispatch the workflow once from a throwaway branch to produce `v0.1.0-rc.1` and verify assets/checksums.
4. Dispatch from `main` to cut `v0.1.0`.
5. Add a note in `README.md` pointing users at the Releases page for binaries.

No rollback needed: the workflow is additive, the hook is opt-in, and reverting the commit removes both cleanly.

## Open Questions

- Initial version: do we seed a `v0.0.0` tag so the action has a base, or let it default? (Probably let it default and let the first dispatch produce `v0.1.0`.)
- Should the `version` job fail when there are zero Conventional Commits since the last tag (no bump)? Current plan: fail with a clear message rather than silently produce the same version.
