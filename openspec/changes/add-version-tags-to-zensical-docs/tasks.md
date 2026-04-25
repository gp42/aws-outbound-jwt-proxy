## 1. Build tooling

- [ ] 1.1 Add a `hack/docs-release.sh` (or equivalent) helper that takes a version string and renders `zensical.toml` with `site_url` set to `<base>/<version>/`, then runs `zensical build --clean` into a version-specific output directory.
- [ ] 1.2 Add a `hack/docs-manifest.py` (or shell/jq) helper that updates `versions.json` — adding/updating an entry, moving the `"latest"` alias, and sorting by semver.
- [ ] 1.3 Add a root `index.html` template (meta-refresh → `./latest/`) used by the publish step.

## 2. Theme override for version selector

- [ ] 2.1 Create `overrides/` directory and wire it into `zensical.toml` via `[project.theme] custom_dir = "overrides"`.
- [ ] 2.2 Add a minimal header partial that fetches `../versions.json` at runtime and renders a `<select>` linking to each version's root, marking the current version active.
- [ ] 2.3 Style the selector to match the existing Zensical header; verify it renders in both `default` and `slate` palettes.

## 3. Release workflow

- [ ] 3.1 Split `.github/workflows/docs.yml` into two jobs/triggers: `on.push.tags: ['v*.*.*']` for releases and `on.push.branches: [main]` for `dev`.
- [ ] 3.2 Release job: derive `MAJOR.MINOR` from the tag, detect prerelease (`-rc`, `-beta`, etc.), build with per-version `site_url`, and place output under a staging directory `vX.Y/`.
- [ ] 3.3 Release job: if not a prerelease and the tag is the highest seen stable semver, copy `vX.Y/` to `latest/` and update the `"latest"` alias in `versions.json`; otherwise leave `latest/` untouched.
- [ ] 3.4 Dev job: build with `site_url = <base>/dev/` and stage output under `dev/`.
- [ ] 3.5 Add `concurrency: group: docs-deploy, cancel-in-progress: false` to serialize deploys.
- [ ] 3.6 Replace `actions/deploy-pages` path with `peaceiris/actions-gh-pages@v4` publishing to the `gh-pages` branch with `keep_files: true`.
- [ ] 3.7 Ensure `versions.json` and root `index.html` are rewritten in the same commit as the version directory update.

## 4. Repository/Pages configuration

- [ ] 4.1 Seed `gh-pages` manually from the current `main` build as `v0.1/` + `latest/` + `versions.json` + root redirect (first-time bootstrap only).
- [ ] 4.2 Switch the GitHub Pages source from "GitHub Actions" to "Deploy from branch: gh-pages / (root)" (document in README/runbook; this is a repo-settings change, not code).
- [ ] 4.3 Update `site_url` comment in `zensical.toml` to note that the effective value is injected per-build by the workflow.

## 5. Validation

- [ ] 5.1 Smoke-test locally: run the release helper for a fake `v0.9.0`, build, serve `site/` with a static server, confirm version selector lists all versions and navigates correctly.
- [ ] 5.2 Verify `<link rel="canonical">` on a `/latest/` page points at the corresponding `/vX.Y/` URL.
- [ ] 5.3 Verify `/` redirects to `/latest/` in a clean browser.
- [ ] 5.4 Push a throwaway tag `v0.0.1-test` to a fork or branch to dry-run the tag workflow; then delete the tag and the resulting `gh-pages` entries.
- [ ] 5.5 Cut the first real tag once validation passes.

## 6. Documentation

- [ ] 6.1 Add a "Release docs" runbook under `docs/` (or `README.md`) covering: how to cut a new version, how to retire an old one (delete directory + update `versions.json` + force-push `gh-pages`), and the dev-preview behavior.
- [ ] 6.2 Note the versioning scheme (MAJOR.MINOR directories, patch overwrites) in the runbook.

## 7. Resolve open questions

- [ ] 7.1 Decide whether patch-level prefixes (`/vX.Y.Z/`) are published; update spec + implementation if the answer is yes.
- [ ] 7.2 Check for a first-party Zensical versioning helper at implementation time; if available and stable, prefer it over the hand-rolled override and revise task group 2 accordingly.
- [ ] 7.3 Decide whether `dev/` appears in the version selector or is hidden; reflect the choice in the selector partial.
