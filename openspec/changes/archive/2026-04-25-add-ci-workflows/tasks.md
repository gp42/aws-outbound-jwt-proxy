## 1. Resolve action SHAs

- [x] 1.1 Reuse `actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2` from `release.yml`
- [x] 1.2 Reuse `actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v6.4.0` from `release.yml`
- [x] 1.3 Confirm both SHAs still resolve to the tagged release on the actions' GitHub repos before committing

## 2. Add CI workflow

- [x] 2.1 Create `.github/workflows/ci.yml` with `name: CI` and `on:` set to `pull_request` (default activity types) and `push` to `branches: [main]`
- [x] 2.2 Set top-level `permissions: contents: read`
- [x] 2.3 Define a single job `build-test` with `runs-on: ${{ matrix.runner }}` and `strategy.fail-fast: false`
- [x] 2.4 Populate the matrix with `(linux, amd64, ubuntu-latest)`, `(linux, arm64, ubuntu-24.04-arm)`, `(darwin, arm64, macos-latest)` — no cross-compile env overrides
- [x] 2.5 Step: `actions/checkout` pinned per task 1.1
- [x] 2.6 Step: `actions/setup-go` pinned per task 1.2 with `go-version-file: go.mod` and `check-latest: true`
- [x] 2.7 Step: `go vet ./...`
- [x] 2.8 Step: `go build ./...`
- [x] 2.9 Step: `go test ./...`

## 3. Add Dependabot configuration

- [x] 3.1 Create `.github/dependabot.yml` with `version: 2`
- [x] 3.2 Add `gomod` updater at `directory: "/"`, `schedule.interval: weekly`, with a `groups.go-deps` block matching `update-types: ["minor", "patch"]` and `patterns: ["*"]`
- [x] 3.3 Add `github-actions` updater at `directory: "/"`, `schedule.interval: weekly`, with a `groups.actions` block matching `update-types: ["minor", "patch"]` and `patterns: ["*"]`
- [x] 3.4 Leave major bumps ungrouped (do not add them to the groups)

## 4. Validate locally and in CI

- [x] 4.1 Run `actionlint` (or equivalent YAML lint) on the new workflow file if available locally; otherwise visually verify YAML structure
- [ ] 4.2 Open the implementation PR and confirm `ci.yml` triggers and all three matrix shards complete
- [ ] 4.3 Confirm Dependabot picks up the config (Insights → Dependency graph → Dependabot tab) once the PR is merged

## 5. Follow-ups (out of this change)

- [ ] 5.1 After job names stabilize, configure branch protection on `main` to require the new CI checks (manual repo-settings step)
