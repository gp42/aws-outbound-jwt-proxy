## Why

Zensical was initialized with boilerplate values (placeholder `site_author`, "A new project generated from the default template project." description, no `site_url`, no repository link, no social links) and the default template's demo content (`docs/index.md` is the Zensical "Get started" page, `docs/markdown.md` is a Markdown primer). The site currently says nothing about what this project actually is, which means it cannot be published as-is. We need real configuration values and a real user-facing documentation set for the AWS Outbound JWT Proxy so the site can be published.

## What Changes

- Replace placeholder values in `zensical.toml` (`site_description`, `site_author`, `site_url`, `copyright`) with values that reflect this project.
- Enable and configure the repository link (`repo_url`, `repo_name`) so the header and edit/view buttons point to `github.com/gp42/aws-outbound-jwt-proxy`.
- Turn on `content.action.edit` and `content.action.view` feature flags now that a repository is configured.
- Add a GitHub social link under `[[project.extra.social]]`.
- Define an explicit top-level navigation structure (`nav`) so pages appear in a deliberate order.
- Remove the template-demo pages (`docs/index.md`, `docs/markdown.md`) and replace them with user documentation for the proxy: a landing/overview page, installation, a quick-start walkthrough, configuration reference (all CLI flags / env vars), dynamic audience mode, deployment/operations guidance (metrics, failure modes), and a troubleshooting page.
- **BREAKING**: The rendered site's pages and URLs change. Any bookmarks to the default demo pages (`/markdown/`) will 404.

## Capabilities

### New Capabilities
- `user-documentation`: End-user-facing documentation set (content structure, required pages, and authoring conventions) published via Zensical for the AWS Outbound JWT Proxy.
- `site-publishing`: Zensical site configuration (project identity, repository integration, navigation, theme features) that makes the site publishable.

### Modified Capabilities
<!-- None. No existing spec covers zensical configuration or end-user documentation. -->

## Impact

- **Files**: `zensical.toml` (edits), `docs/index.md` (rewrite), `docs/markdown.md` (delete), new files under `docs/` for each documentation page.
- **Build output**: `site/` regenerates with different pages and URLs.
- **Dependencies**: None added; uses features Zensical already ships.
- **Code**: No Go code changes. The proxy binary and its behavior are unaffected.
