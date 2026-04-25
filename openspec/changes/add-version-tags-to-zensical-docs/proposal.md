## Why

The docs site currently publishes a single rolling build from `main`, so users on a released binary can read docs describing features or flags that don't exist in their version — and older documentation is overwritten on every merge. As the project starts cutting tagged releases, readers need a way to land on the docs that match the binary they're running, and maintainers need old versions to stay reachable.

## What Changes

- Publish versioned copies of the docs under per-version URL prefixes (e.g., `/latest/`, `/v0.2/`, `/v0.1/`) instead of overwriting the site root on every push.
- Add a version selector to the site header driven by a `versions.json` manifest at the site root.
- Designate one version as the default (`latest`) so bare URLs redirect to it; keep a stable `/latest/` alias that tracks the newest release.
- Drive publication from git tags: pushing `vX.Y.Z` builds and deploys that version; pushes to `main` publish a `dev`/preview version without promoting `latest`.
- Update the GitHub Pages workflow to accumulate versions across deploys rather than replacing the `site` artifact each run.
- Document the release-docs flow (how to cut a version, how to retire one) in the repo.

## Capabilities

### New Capabilities
- `docs-versioning`: Publishing, listing, and navigating between multiple versions of the documentation site, including the version manifest, selector UI, default-version alias, and the tag-driven build/deploy workflow.

### Modified Capabilities
<!-- None — no existing spec governs docs publication today. -->

## Impact

- `.github/workflows/docs.yml`: replaced with a tag- and branch-aware workflow that preserves prior versions on gh-pages.
- `zensical.toml`: `site_url` and theme/extras adjusted to support a version selector and per-version base URLs.
- `docs/`: minor additions (version selector config, possibly a redirect stub at the root).
- New developer docs / release runbook describing how versioned docs are cut.
- gh-pages branch contents change shape: one subdirectory per version plus `versions.json` and a root redirect, instead of a flat site.
- Dependencies: may add a versioning helper (e.g., a Zensical-compatible equivalent of `mike`, or a small custom script) to the docs build.
