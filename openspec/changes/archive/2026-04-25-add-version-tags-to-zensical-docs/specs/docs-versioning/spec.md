## ADDED Requirements

### Requirement: Per-version URL prefix
The documentation site SHALL serve each published version under its own stable URL prefix of the form `/vMAJOR.MINOR/` (e.g., `/v0.2/`). Deploying a new version MUST NOT modify or remove the content of previously published version prefixes.

#### Scenario: Version-prefixed page is reachable
- **WHEN** a reader requests `https://<pages-host>/aws-outbound-jwt-proxy/v0.2/configuration/`
- **THEN** the site responds with the `configuration` page built from the `v0.2.*` tag

#### Scenario: Publishing a new version preserves prior versions
- **WHEN** the workflow deploys a new version `v0.3`
- **THEN** `/v0.2/` and all earlier version prefixes remain byte-identical to their last deploy

### Requirement: Patch releases overwrite the minor prefix
When a tag of the form `vX.Y.Z` is pushed, the workflow SHALL deploy the build to `/vX.Y/`, overwriting any existing content at that prefix. The workflow MUST NOT create a separate `/vX.Y.Z/` prefix.

#### Scenario: Patch release updates minor directory
- **WHEN** tag `v0.2.1` is pushed and an existing `/v0.2/` directory contains the `v0.2.0` build
- **THEN** `/v0.2/` is replaced with the `v0.2.1` build and no `/v0.2.1/` directory is created

### Requirement: Latest alias
The site SHALL expose a `/latest/` prefix that is a full copy of the newest non-prerelease published version. Each release promotion MUST update `/latest/` atomically with the rest of the deploy (version directory, manifest, and root redirect in the same commit to the publishing branch).

#### Scenario: Stable release promotes latest
- **WHEN** tag `v0.3.0` is pushed
- **THEN** `/latest/` is overwritten with the `v0.3.0` build and `versions.json` lists `v0.3` with `aliases: ["latest"]`

#### Scenario: Prerelease does not promote latest
- **WHEN** tag `v1.0.0-rc.1` is pushed
- **THEN** `/v1.0/` is deployed but `/latest/` and the `aliases: ["latest"]` entry remain pointing at the previously promoted stable version

### Requirement: Root redirects to latest
The site root (`/`) SHALL redirect the browser to `/latest/`. The redirect MUST be a client-side meta-refresh or equivalent that works on static GitHub Pages hosting without server-side rewrites.

#### Scenario: Bare URL redirects
- **WHEN** a reader requests `https://<pages-host>/aws-outbound-jwt-proxy/`
- **THEN** the browser is redirected to `https://<pages-host>/aws-outbound-jwt-proxy/latest/`

### Requirement: Version manifest
The site SHALL publish a `versions.json` file at the site root enumerating all currently published versions. Each entry MUST contain `version` (URL segment), `title` (display label), and `aliases` (array, possibly empty). The manifest MUST be updated atomically with every deploy.

#### Scenario: Manifest lists published versions
- **WHEN** `/v0.2/`, `/v0.1/`, and `/dev/` are currently published and `v0.2` is promoted to latest
- **THEN** `versions.json` at the site root contains entries for each with `v0.2` carrying `aliases: ["latest"]`

### Requirement: Version selector in header
Every documentation page SHALL render a version selector in the site header that lists the entries from `versions.json` and navigates to the selected version at the root of that version's prefix.

#### Scenario: Selecting another version navigates there
- **WHEN** a reader on `/v0.2/configuration/` opens the selector and picks `v0.1`
- **THEN** the browser navigates to `/v0.1/` (the root of the selected version)

#### Scenario: Current version is marked
- **WHEN** a reader is viewing any page under `/v0.2/`
- **THEN** the selector indicates `v0.2` as the active entry

### Requirement: Tag-driven release build
The docs deployment workflow SHALL trigger on pushes of tags matching `v*.*.*` and produce a build for the corresponding `/vMAJOR.MINOR/` prefix. The workflow MUST serialize concurrent deploys so that parallel tag/branch pushes cannot corrupt the publishing branch.

#### Scenario: Tag push deploys the matching version
- **WHEN** tag `v0.3.0` is pushed to the repository
- **THEN** the workflow builds the docs from that tag's tree and publishes them to `/v0.3/`

#### Scenario: Concurrent deploys serialize
- **WHEN** two deploy-triggering pushes occur within the same minute
- **THEN** the second workflow run waits for the first to finish before modifying the publishing branch

### Requirement: Main-branch dev preview
Pushes to the `main` branch SHALL publish a preview build to `/dev/`. This deploy MUST NOT alter any `/vX.Y/` directory, MUST NOT update `/latest/`, and MUST NOT change the `aliases` of any manifest entry other than `dev`.

#### Scenario: Main push updates only dev
- **WHEN** a commit is pushed to `main`
- **THEN** `/dev/` is rebuilt and `versions.json` still lists the same release entries with unchanged `aliases`

### Requirement: Per-version canonical URL
Each version's generated HTML SHALL declare a canonical URL under its own versioned prefix (e.g., pages under `/v0.2/` point canonical at `https://<pages-host>/aws-outbound-jwt-proxy/v0.2/...`). The `/latest/` copy MUST retain the canonical of the version it was promoted from.

#### Scenario: Latest copy keeps versioned canonical
- **WHEN** a reader views `/latest/configuration/` and the current latest is `v0.2`
- **THEN** the page's `<link rel="canonical">` points to `/v0.2/configuration/`
