## ADDED Requirements

### Requirement: Project identity is accurate

`zensical.toml` SHALL set `site_name`, `site_description`, `site_author`, `site_url`, and `copyright` to values that reflect the AWS Outbound JWT Proxy project, with no remaining template placeholders (e.g., "A new project generated from the default template project.", "<your name here>").

#### Scenario: No template placeholders remain
- **WHEN** `zensical.toml` is searched for the strings `"<your name here>"` and `"generated from the default template project"`
- **THEN** no matches are found

#### Scenario: site_url is set
- **WHEN** `zensical.toml` is parsed
- **THEN** `site_url` is present and non-empty

### Requirement: Repository integration is enabled

`zensical.toml` SHALL configure repository integration by setting `repo_url` and `repo_name` and SHALL enable the `content.action.edit` and `content.action.view` features so rendered pages expose edit/view-source buttons.

#### Scenario: Repo settings present
- **WHEN** `zensical.toml` is parsed
- **THEN** `repo_url` points to `https://github.com/gp42/aws-outbound-jwt-proxy`
- **AND** `repo_name` is set
- **AND** the `features` list contains `"content.action.edit"` and `"content.action.view"`

### Requirement: GitHub social link is present

`zensical.toml` SHALL declare at least one `[[project.extra.social]]` entry linking to the project's GitHub repository.

#### Scenario: GitHub social entry
- **WHEN** `zensical.toml` is parsed
- **THEN** there is a `[[project.extra.social]]` entry whose `link` points to `https://github.com/gp42/aws-outbound-jwt-proxy`

### Requirement: Navigation is explicitly defined

`zensical.toml` SHALL define an explicit `nav` structure at `[project]` level listing each user-documentation page with a human-readable title, in a deliberate order (Overview, Install, Quick start, Configuration, Dynamic audience, Operations, Troubleshooting).

#### Scenario: Nav contains all user-doc pages
- **WHEN** `zensical.toml` is parsed
- **THEN** `project.nav` is defined
- **AND** each user-documentation page from the user-documentation capability has a corresponding entry in `nav`

### Requirement: Site builds without errors

Running the project's build tooling SHALL produce a `site/` directory without warnings about missing files, dangling nav references, or template-placeholder values.

#### Scenario: Clean build
- **WHEN** the build command is run on a clean checkout
- **THEN** the command exits with status 0
- **AND** `site/index.html` exists and renders the Overview page
