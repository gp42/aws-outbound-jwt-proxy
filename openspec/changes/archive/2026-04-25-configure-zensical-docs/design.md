## Context

`zensical.toml` was created via `zensical new` and still holds template defaults. The `docs/` folder contains the template's two demo pages. The repo already has a detailed `README.md` and seven spec capabilities under `openspec/specs/` (`proxy-configuration`, `token-acquisition`, `request-forwarding`, `outbound-auth-header`, `upstream-routing`, `dynamic-audience-resolution`, `metrics-observability`) that describe the proxy's behavior authoritatively. User documentation should reuse that material rather than re-describing it from scratch.

GitHub repo is `github.com/gp42/aws-outbound-jwt-proxy`. There is no published site URL yet; we'll use a GitHub Pages URL (`https://gp42.github.io/aws-outbound-jwt-proxy/`) as the canonical placeholder — easy to change when/if the site moves.

## Goals / Non-Goals

**Goals:**
- Produce a `zensical.toml` whose metadata is accurate and whose feature flags match the fact that a repository is now configured.
- Produce a documentation set that a first-time user can read top-to-bottom to install, configure, run, operate, and troubleshoot the proxy.
- Keep the docs as the single source of user-facing truth; `README.md` on GitHub stays as a concise entry point that links into the published site.

**Non-Goals:**
- Publishing the site (no CI workflow, no GitHub Pages setup) — that is a separate change.
- Customizing theme beyond enabling sensible stock features (no custom logo, fonts, palette beyond the two existing schemes, or `overrides/` directory).
- Writing developer/contributor docs or API reference — user-facing only.
- Changing any Go code or spec requirements for the proxy itself.

## Decisions

### Documentation structure

Pages (in navigation order):

1. `docs/index.md` — **Overview**: what the proxy is, the federation flow diagram, when to use it.
2. `docs/install.md` — **Install**: prebuilt binaries (from `make build-all` outputs), building from source, Docker (if applicable — note it's out of scope if no image exists).
3. `docs/quick-start.md` — **Quick start**: a minimal end-to-end example (run against a single upstream, verify a request is forwarded with a JWT).
4. `docs/configuration.md` — **Configuration reference**: one table of all CLI flags with their env-var equivalents, defaults, and descriptions. Sourced by reading `cmd/` flag definitions at implementation time.
5. `docs/dynamic-audience.md` — **Dynamic audience mode**: when and how to use it, cache/cardinality considerations, how audience is derived.
6. `docs/operations.md` — **Operations**: metrics exposed (from `metrics-observability` spec), failure modes (STS failure → 502), token caching behavior.
7. `docs/troubleshooting.md` — **Troubleshooting**: common errors (invalid audience, expired token, upstream unreachable, 502 from proxy) with diagnostic steps.

**Alternative considered**: a single long page. Rejected — Zensical's navigation features (sections, TOC, instant nav) are more useful with multiple shorter pages, and the current content naturally splits along these lines.

### Nav definition in `zensical.toml`

Define `nav` explicitly (as TOML inline tables, per the template comment) rather than relying on implicit directory-derived navigation. This gives us stable, human-readable titles and ordering independent of filenames.

### Repo integration

Set `repo_url` and `repo_name` at the `[project]` level. Turn on `content.action.edit` and `content.action.view` in the `features` list so the page header offers edit/view-source buttons. These features are already documented in the file and currently commented out.

### Social links

Add a single `[[project.extra.social]]` entry pointing at the GitHub repo with `icon = "fontawesome/brands/github"`.

### Reuse of existing material

- Overview page reuses the flow description from `README.md` ("How it works") almost verbatim — that text is already reviewed and accurate.
- Configuration reference pulls defaults and descriptions from `cmd/` flag registrations — the authoritative source. We do not hand-maintain a separate flag list anywhere else.
- Dynamic audience page paraphrases the existing `README.md#dynamic-audience` section plus the `dynamic-audience-resolution` spec.
- Operations/metrics content derives from the `metrics-observability` spec.

### What stays in README vs. goes into docs site

`README.md` keeps a short intro, one quick-start snippet, and a link to the published docs site. Everything else migrates to docs. This avoids two places drifting out of sync, while still giving GitHub visitors immediate orientation.

## Risks / Trade-offs

- **[Risk] Config docs drift from actual flags.** → Mitigation: generate/verify the configuration reference from `cmd/` flag registrations during implementation (read source, don't paraphrase from memory). Add a follow-up task to revisit if flags are added.
- **[Risk] `site_url` is wrong if we never publish to GitHub Pages.** → Mitigation: use `https://gp42.github.io/aws-outbound-jwt-proxy/` as the canonical URL; it's cheap to change later, and leaving it unset disables the canonical `<link>` tag and sitemap absolute URLs.
- **[Risk] README trimming loses useful context for GitHub readers.** → Mitigation: keep the "what it does" and "how it works" sections in README; only move deep configuration reference and operations content to docs.
- **[Trade-off] Explicit `nav` means adding a page requires editing `zensical.toml`.** Accepted — the site is small and navigation order is a deliberate curation choice.

## Migration Plan

No data or runtime migration. To deploy:
1. Merge the change.
2. Run `zensical build` (or existing build tooling) to regenerate `site/`.
3. If/when the site is published, the URLs for the two deleted template pages (`/markdown/`) will 404 — acceptable because no external link depends on them (fresh project).

Rollback: `git revert` the commit. No external state is affected.
