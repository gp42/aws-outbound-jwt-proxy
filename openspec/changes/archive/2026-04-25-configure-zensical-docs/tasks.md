## 1. Update zensical.toml metadata

- [x] 1.1 Set `site_description` to a one-sentence description of the AWS Outbound JWT Proxy
- [x] 1.2 Set `site_author` to the project owner name
- [x] 1.3 Uncomment and set `site_url` to `https://gp42.github.io/aws-outbound-jwt-proxy/`
- [x] 1.4 Update `copyright` to name the project (replace generic "The authors" if desired)
- [x] 1.5 Add `repo_url = "https://github.com/gp42/aws-outbound-jwt-proxy"` and `repo_name = "gp42/aws-outbound-jwt-proxy"` under `[project]`
- [x] 1.6 Uncomment `"content.action.edit"` and `"content.action.view"` in the `features` list
- [x] 1.7 Add a `[[project.extra.social]]` entry with `icon = "fontawesome/brands/github"` and `link` pointing at the repo

## 2. Define navigation

- [x] 2.1 Uncomment and define `nav` under `[project]` with entries for Overview, Install, Quick start, Configuration, Dynamic audience, Operations, Troubleshooting — mapped to the doc files created in section 3

## 3. Author user documentation

- [x] 3.1 Delete `docs/markdown.md`
- [x] 3.2 Rewrite `docs/index.md` as the Overview page (what the proxy does, flow summary, when to use it) — source text from `README.md` "What it does" / "How it works"
- [x] 3.3 Create `docs/install.md` — prebuilt binaries via `make build-all`, building from source (`go build`), minimum Go version from `go.mod`
- [x] 3.4 Create `docs/quick-start.md` — minimal end-to-end example: set required env vars / flags, run binary, make a request through it, observe the Authorization header injection
- [x] 3.5 Create `docs/configuration.md` — read flag registrations from `cmd/` and produce a reference table (flag, env var, default, description) covering every flag
- [x] 3.6 Create `docs/dynamic-audience.md` — derive content from `README.md#dynamic-audience` and `openspec/specs/dynamic-audience-resolution/spec.md`; cover normalization, per-host caching, cardinality caveat, behavior when `--token-audience` is set
- [x] 3.7 Create `docs/operations.md` — metrics (names and types from `openspec/specs/metrics-observability/spec.md`), failure modes (STS failure → 502), token cache behavior
- [x] 3.8 Create `docs/troubleshooting.md` — common errors and how to diagnose: invalid audience, upstream unreachable, 502 from proxy, token fetch failures

## 4. Trim README

- [x] 4.1 Keep intro, "What it does", short "How it works" summary in `README.md`
- [x] 4.2 Remove or condense the deep Configuration section; replace with a link to the published docs site
- [x] 4.3 Add a "Documentation" section near the top of `README.md` linking to the published site

## 5. Verify

- [x] 5.1 Run the project's Zensical build command; confirm exit code 0 and no warnings about missing nav targets or template placeholders
- [x] 5.2 Open `site/index.html` locally and confirm it renders the Overview page (not the template demo)
- [x] 5.3 Grep `zensical.toml` for `"<your name here>"` and `"generated from the default template project"` — confirm no matches
- [x] 5.4 Confirm every page listed in `nav` has a corresponding file in `docs/`
