## Context

Today `.github/workflows/docs.yml` runs on every push to `main`: it does a clean Zensical build and uploads the resulting `site/` directory as the single GitHub Pages artifact. There is no history — any previously deployed content is replaced. `site_url` in `zensical.toml` points at the bare root `https://gp42.github.io/aws-outbound-jwt-proxy/`, and the site has no version selector.

The project uses Zensical, a Material-for-MkDocs-compatible generator. MkDocs' canonical versioning tool (`mike`) writes versioned subdirectories to the `gh-pages` branch and maintains a `versions.json` manifest that a theme-side version selector reads; Zensical advertises Material compatibility but is not a drop-in for `mike`. We need a versioning approach that works with Zensical's build output (a plain `site/` directory) and GitHub Pages as the host, and that is simple enough to live alongside a small Go project.

Stakeholders: end users running tagged binary releases (primary), maintainers cutting releases (secondary).

## Goals / Non-Goals

**Goals:**
- Serve multiple doc versions simultaneously under stable per-version URLs (`/vX.Y/`).
- Provide a `/latest/` alias that always points at the newest stable release, and make `/` redirect to `/latest/`.
- Render a version selector in the site header, populated from a `versions.json` manifest at the site root.
- Drive version publication from git tags (`vX.Y.Z`), not from `main` pushes. Pushes to `main` may publish a `dev` preview that does not promote `latest`.
- Keep old versions reachable and unmodified unless explicitly retired.

**Non-Goals:**
- Per-page "since version X" badges (a separate change, option 2 from the scoping question).
- Cross-version search (each version has its own search index).
- Backporting doc fixes into historical version branches — versions are treated as immutable snapshots unless a maintainer explicitly republishes.
- Translations or multi-language versioning.

## Decisions

### D1. Publish to `gh-pages` via `peaceiris/actions-gh-pages` instead of `actions/deploy-pages`

The current workflow uses GitHub's artifact-based Pages deploy, which replaces the whole site on every run. For accumulation we need a persistent branch we can merge into. We switch the Pages source to the `gh-pages` branch and use `peaceiris/actions-gh-pages@v4` with `keep_files: true` so existing version subdirectories survive new deploys.

Alternative considered: continue with `actions/deploy-pages` and rebuild the full multi-version tree on every run by checking out all tags and building each. Rejected — build time grows linearly with released versions, and reproducibility depends on old tags still building with the current Zensical version.

### D2. Layout on `gh-pages`

```
/
├── index.html          # meta-refresh → /latest/
├── versions.json       # [{version, title, aliases}, ...]
├── latest/             # copy of or symlink-style duplicate of newest release
├── v0.2/
├── v0.1/
└── dev/                # rebuilt on every push to main (optional)
```

`versions.json` schema matches `mike`'s so a Material-compatible version-selector partial can consume it unchanged:
```json
[
  {"version": "v0.2", "title": "v0.2 (latest)", "aliases": ["latest"]},
  {"version": "v0.1", "title": "v0.1", "aliases": []},
  {"version": "dev",  "title": "dev",  "aliases": []}
]
```

### D3. `latest` is a copy, not a redirect

Each release promotion copies the built `vX.Y/` tree to `latest/` (and rewrites `versions.json`). This keeps absolute asset URLs inside `latest/` self-consistent and avoids double-redirect hops. Cost: ~2× disk on `gh-pages`, which is negligible for a docs site.

Alternative: make `latest/` a directory of meta-refresh stubs that point into `vX.Y/`. Rejected — breaks relative links and fragments, and confuses crawlers.

### D4. Tag-driven workflow

Split `.github/workflows/docs.yml` into two triggers:
- `push: tags: ['v*.*.*']` → build, deploy to `/vX.Y/`, promote to `/latest/` if the tag is the highest semver seen (prerelease tags like `v1.0.0-rc.1` do NOT promote).
- `push: branches: [main]` → build, deploy to `/dev/` only.

Version derivation: strip the leading `v` and keep `MAJOR.MINOR` as the URL segment (`v0.2.1` → `/v0.2/`). Patch releases overwrite the minor directory. This keeps the selector list short; patch notes go in the changelog inside the version.

Open: whether to expose `v0.2.1` as its own directory as well. See Open Questions.

### D5. Version-selector integration

Zensical's Material-compatible theme supports template overrides via `[project.theme] custom_dir`. We add an `overrides/` directory with a minimal header partial that fetches `../versions.json` (relative to the current version's base URL) and renders a `<select>`. The partial is adapted from Material's `mkdocs-material/partials/nav.html` version block; we keep it small (<50 lines) and self-contained.

Alternative: adopt a Zensical-native plugin if/when one exists. Tracked as an open question.

### D6. `site_url` per version

Zensical bakes `site_url` into generated HTML (canonical tags, sitemap). The release workflow passes `--site-url https://gp42.github.io/aws-outbound-jwt-proxy/vX.Y/` (or the equivalent env/config override) at build time for each version, and `…/latest/` for the promoted copy. If Zensical has no CLI override, we render `zensical.toml` from a template with the version substituted.

## Risks / Trade-offs

- **Zensical-vs-mike compatibility drift** → Mitigation: keep the version-selector override tiny and under our control; pin the Zensical version in CI to avoid surprise template changes.
- **`gh-pages` branch grows unbounded as versions accumulate** → Mitigation: documented retirement procedure (delete directory, update `versions.json`, force-push). Acceptable at this project's release cadence.
- **Concurrent tag and main pushes racing on `gh-pages`** → Mitigation: workflow `concurrency: group: docs-deploy, cancel-in-progress: false` so deploys serialize.
- **Old tags may not rebuild with the current toolchain if ever re-run** → Mitigation: versions are treated as immutable once deployed; we don't attempt to rebuild them on every run (see D1 alternative rejection).
- **Canonical `site_url` is baked in per build, so a copy-to-`latest/` leaves canonical tags pointing at `/vX.Y/`** → That's intentional and desired (search engines should index the versioned URL as canonical; `/latest/` is an alias).
- **`/` root redirect and `/latest/` alias can diverge if the promotion step fails halfway** → Mitigation: promotion is a single atomic commit to `gh-pages` (update `latest/`, rewrite `index.html`, rewrite `versions.json` together).

## Migration Plan

1. Land the workflow change behind a new branch; do an initial manual run that seeds `gh-pages` with `v0.1/` (current content), `latest/` (copy), `versions.json`, and the root redirect. Verify the site loads.
2. Switch Pages source to "Deploy from branch: gh-pages / (root)" in repo settings.
3. Remove the `actions/deploy-pages` path from `docs.yml`.
4. Cut a real tag to validate the tag-driven flow end-to-end.

Rollback: revert the workflow commit and flip Pages source back to "GitHub Actions". The `gh-pages` branch can remain as a historical artifact; nothing depends on its shape once the source flips back.

## Open Questions

- Should patch releases (`v0.2.1`) also publish a dedicated `/v0.2.1/` directory, or only overwrite `/v0.2/`? (Current lean: overwrite only — patch-level URLs are rarely linked and double the directory count.)
- Is there a first-party Zensical versioning helper we should prefer over a hand-rolled `versions.json` + override? Check at implementation time; if one exists and is stable, D5/D2 may simplify.
- Should `dev/` be listed in the version selector, or hidden (reachable only by direct URL) to avoid confusing users? (Lean: listed, with a clearly labeled "dev" title.)
