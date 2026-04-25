## ADDED Requirements

### Requirement: Documentation set covers install-to-operate lifecycle

The published documentation SHALL contain, at minimum, pages covering: project overview, installation, a quick-start walkthrough, a configuration reference, dynamic audience mode, operations (metrics and failure modes), and troubleshooting. Each page SHALL be reachable from the site's top-level navigation.

#### Scenario: First-time user flow
- **WHEN** a user visits the site's home page with no prior context
- **THEN** the Overview page explains what the proxy does and links forward to Install
- **AND** Install, Quick start, Configuration, Dynamic audience, Operations, and Troubleshooting are each reachable in that order from the top-level navigation

#### Scenario: Configuration reference is complete
- **WHEN** the Configuration page is rendered
- **THEN** every CLI flag registered in `cmd/` appears in the reference with its env-var equivalent, default, and description

### Requirement: Documentation content is sourced from authoritative project material

Documentation pages SHALL derive their content from existing authoritative sources in the repository — `README.md` for overview/flow, `cmd/` for flag definitions, and `openspec/specs/` for behavioral details — rather than restating behavior from scratch.

#### Scenario: Dynamic audience page matches spec
- **WHEN** the Dynamic audience page is compared to `openspec/specs/dynamic-audience-resolution/spec.md`
- **THEN** the page does not contradict any requirement in that spec (normalization rules, caching behavior, fallback when `--token-audience` is set)

#### Scenario: Operations page matches metrics spec
- **WHEN** the Operations page lists exposed metrics
- **THEN** each metric named is also defined in `openspec/specs/metrics-observability/spec.md`

### Requirement: Template demo content is removed

The default Zensical template pages (`docs/markdown.md` and any template-provided content in `docs/index.md`) SHALL NOT remain in the documentation set.

#### Scenario: No template page survives
- **WHEN** the `docs/` directory is inspected after the change
- **THEN** `docs/markdown.md` does not exist
- **AND** `docs/index.md` does not contain the template's "For full documentation visit zensical.org" text

### Requirement: README links into the published documentation

`README.md` SHALL remain a concise entry point (project description, short quick-start, link to the docs site) and SHALL link to the published documentation rather than duplicating reference material.

#### Scenario: README keeps overview, links to docs
- **WHEN** `README.md` is viewed on GitHub
- **THEN** it contains a "what it does" overview and a link to the published docs site
- **AND** it does not contain the full flag-by-flag configuration reference (that lives only in the docs site)
