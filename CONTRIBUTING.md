# Contributing

Thanks for your interest in `aws-outbound-jwt-proxy`! This guide covers everything you need to send a pull request.

## TL;DR

```sh
# 1. Fork on GitHub, then:
git clone git@github.com:<you>/aws-outbound-jwt-proxy.git
cd aws-outbound-jwt-proxy
git remote add upstream git@github.com:gp42/aws-outbound-jwt-proxy.git

# 2. Install the commit-msg hook (one-time, per clone)
make install-hooks

# 3. Branch, code, test
git checkout -b feat/my-thing
make build test vet

# 4. Commit using Conventional Commits and push
git commit -m "feat(proxy): add my thing"
git push -u origin feat/my-thing

# 5. Open a pull request against gp42/aws-outbound-jwt-proxy:main on GitHub
```

## Prerequisites

- Go matching the version in [`go.mod`](go.mod) (`go 1.25`).
- `make`.
- A POSIX shell for git hooks.

## Workflow

### 1. Fork and clone

Fork `gp42/aws-outbound-jwt-proxy` on GitHub, then clone your fork. Add the canonical repo as `upstream` so you can pull updates:

```sh
git remote add upstream git@github.com:gp42/aws-outbound-jwt-proxy.git
git fetch upstream
```

### 2. Install hooks

```sh
make install-hooks
```

This sets `core.hooksPath=hack/hooks` and activates the `commit-msg` hook that enforces [Conventional Commits](COMMIT_CONVENTIONS.md). Without this, your commits may be silently ignored when the next version is computed.

### 3. Create a topic branch

Branch off the latest `main`:

```sh
git fetch upstream
git checkout -b feat/<short-description> upstream/main
```

Branch names are not enforced, but keeping a `<type>/<short-description>` shape (e.g. `fix/empty-sts-response`) helps reviewers.

### 4. Make your changes

- Keep the change focused on one concern. Several small PRs beat one large one.
- Match existing code style. Run:

  ```sh
  make fmt vet test
  ```
- Add or update tests for the behavior you're changing. Run them with `make test`.
- Update docs/help text when behavior or flags change. End-user docs live under [`docs/`](docs); maintainer docs (this file, [RELEASING.md](RELEASING.md), [COMMIT_CONVENTIONS.md](COMMIT_CONVENTIONS.md)) live at the repo root.

### 5. Commit

Every commit message must follow [Conventional Commits](COMMIT_CONVENTIONS.md). The hook will reject anything that doesn't, and bypassing it with `--no-verify` will hurt the next version bump - so don't.

Examples:

```
feat(proxy): add audience override
fix: handle empty STS response
docs(contributing): clarify branch naming
```

Squashing your local fixup commits before pushing is appreciated but not required - the maintainers can handle this at merge time.

### 6. Push and open a pull request

```sh
git push -u origin feat/my-thing
```

Then open a PR against `gp42/aws-outbound-jwt-proxy:main`. In the description:

- Explain **what** changed and **why**. Link any related issue.
- Note any breaking change explicitly (it should also be in your commit message as `…!:` or a `BREAKING CHANGE:` footer).
- Mention how you tested. If the change is user-visible, include before/after output or a config example.

## Reviewing and merging

- A maintainer will review and may request changes. Push additional commits to the same branch - do not force-push unless asked.
- CI must be green before merge.
- Merges land via squash-merge. The squash commit message is what shows up in `git log`, so it will be edited to be a clean Conventional Commit.

## Reporting issues

Open a GitHub issue. Useful information:

- Version (`aws-outbound-jwt-proxy version`).
- Relevant flags / env vars (redact secrets).
- Logs around the failure, ideally at `--log-level=debug`.
- Minimum repro steps.

## Security issues

Do **not** open a public issue for security reports. Email the maintainer (see the GitHub profile of `gp42`) so the issue can be triaged privately before disclosure.

## Releases

Cutting releases is documented in [RELEASING.md](RELEASING.md). Contributors don't need to do anything special - your merged commits become part of the next dispatched release.

## License

By contributing, you agree that your contributions are licensed under the project's [LICENSE](LICENSE).
