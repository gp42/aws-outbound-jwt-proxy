## 1. Version plumbing in the binary

- [x] 1.1 Create `internal/version/version.go` with `var Version = "dev"` and a `String()` helper
- [x] 1.2 Add a `version` subcommand (or `--version` flag) in `cmd/root.go` that prints `internal/version.Version`
- [x] 1.3 Add `VERSION ?= dev` and an `LDFLAGS_VERSION` var to `Makefile`; wire it into `build`, `build-all`, and cross-compile targets via `-X github.com/gp42/aws-outbound-jwt-proxy/internal/version.Version=$(VERSION)`
- [x] 1.4 Verify locally: `make build && ./bin/aws-outbound-jwt-proxy version` prints `dev`; `make build VERSION=v0.0.0-test` prints `v0.0.0-test`

## 2. Commit-msg hook

- [x] 2.1 Write `hack/hooks/commit-msg` (POSIX shell) implementing the regex from design D6, with pass-through for `Merge ` and `Revert "` first lines
- [x] 2.2 `chmod +x hack/hooks/commit-msg`
- [x] 2.3 Add `install-hooks` target to `Makefile` that runs `git config core.hooksPath hack/hooks`
- [x] 2.4 Add `uninstall-hooks` target that runs `git config --unset core.hooksPath` (best-effort)
- [x] 2.5 Manual test: `make install-hooks`; try `git commit --allow-empty -m 'bad msg'` → rejected; `git commit --allow-empty -m 'chore: test'` → accepted
- [x] 2.6 Manual test: verify disallowed type (e.g. `wip: x`) is rejected and the error lists allowed types

## 3. Release workflow — version job

- [x] 3.1 Create `.github/workflows/release.yml` with `on: workflow_dispatch:` only (no inputs)
- [x] 3.2 Add a `version` job on `ubuntu-latest` that checks out with `fetch-depth: 0` and `fetch-tags: true`
- [x] 3.3 Add a step that fails early if `github.ref_type != 'branch'`
- [x] 3.4 Compute a boolean output `is_prerelease = github.ref_name != 'main'`
- [x] 3.5 Invoke a pinned `huggingface/semver-release-action@<pinned-version>` in dryRun mode with a per-branch `branches` config (synthesizes a prerelease channel for non-main); expose the computed version as a job output `version`. (Action does not have `pre_release`/`pre_release_suffix` inputs — uses semantic-release `branches` config instead; design.md D1/D2 updated.)
- [x] 3.6 Add a summary step that writes the computed version and release kind to `$GITHUB_STEP_SUMMARY`
- [x] 3.7 Fail the job if the action did not produce a bumped version

## 4. Release workflow — build matrix

- [x] 4.1 Add a `build` job with `needs: version` and `strategy.matrix` over `{os: linux, arch: amd64}`, `{os: linux, arch: arm64}`, `{os: darwin, arch: arm64}` — each shard runs on a native-arch runner (`ubuntu-latest`, `ubuntu-24.04-arm`, `macos-latest`)
- [x] 4.2 Use `actions/setup-go@<pinned>` with the Go version from `go.mod`
- [x] 4.3 Build with `CGO_ENABLED=0 GOOS=$matrix.os GOARCH=$matrix.arch go build -trimpath -ldflags "-s -w -X github.com/gp42/aws-outbound-jwt-proxy/internal/version.Version=${{ needs.version.outputs.version }}" -o aws-outbound-jwt-proxy-${{ needs.version.outputs.version }}-${{ matrix.os }}-${{ matrix.arch }} .`
- [x] 4.4 Upload each binary as a workflow artifact using `actions/upload-artifact`

## 5. Release workflow — publish job

- [x] 5.1 Add a `publish` job with `needs: [version, build]`, `permissions: contents: write`
- [x] 5.2 Download all build artifacts into a single directory
- [x] 5.3 Generate `SHA256SUMS` via `sha256sum aws-outbound-jwt-proxy-* > SHA256SUMS` (sort for determinism)
- [x] 5.4 Create and push an annotated tag equal to `needs.version.outputs.version` pointing at `GITHUB_SHA` (configure git user as `github-actions[bot]`)
- [x] 5.5 Use `softprops/action-gh-release@<pinned>` to create the release with `prerelease: ${{ needs.version.outputs.is_prerelease }}`, `make_latest: ${{ !needs.version.outputs.is_prerelease }}`, and all binaries + `SHA256SUMS` attached
- [x] 5.6 Confirm the release body contains the compare link `https://github.com/<owner>/<repo>/compare/<prev-tag>...<new-tag>`

## 6. Documentation

- [x] 6.1 Write `RELEASING.md` (top-level) describing: how to dispatch from main vs. a branch, what `-rc.N` means, how to verify `SHA256SUMS`
- [x] 6.2 Write `COMMIT_CONVENTIONS.md` (top-level) listing allowed types, examples, and how to install the hook
- [x] 6.3 Add top-level `CONTRIBUTING.md` (fork → branch → test → PR) linking to `COMMIT_CONVENTIONS.md` and `RELEASING.md`
- [x] 6.4 Link `CONTRIBUTING.md`, `COMMIT_CONVENTIONS.md`, and `RELEASING.md` from `README.md` (kept out of `docs/` because that directory is the published end-user site)

## 7. Validation

- [ ] 7.1 Dispatch the workflow from a throwaway branch; confirm it produces `vX.Y.Z-rc.1`, marks the release as prerelease, and does not move the `latest` pointer
- [ ] 7.2 Dispatch the workflow from `main`; confirm it produces `vX.Y.Z`, the release is not a prerelease, and `latest` now points at it
- [ ] 7.3 Download each binary on its target OS/arch, run `--version`, and verify it prints the expected semver
- [ ] 7.4 Verify `SHA256SUMS` matches the downloaded binaries
- [x] 7.5 Run `openspec validate add-release-automation --strict` and fix any issues
