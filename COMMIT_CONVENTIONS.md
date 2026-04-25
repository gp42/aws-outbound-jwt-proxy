# Commit conventions

This repository uses [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/). The release workflow derives the next semver from these messages, so commits that don't conform are silently ignored when computing the version. To prevent that, a local `commit-msg` hook **rejects** non-conforming messages.

## Install the hook

One-time, per clone:

```sh
make install-hooks
```

This sets `core.hooksPath=hack/hooks` so every hook in [`hack/hooks/`](hack/hooks) becomes active. Uninstall with:

```sh
make uninstall-hooks
```

## Format

```
<type>(<optional-scope>)<!?>: <subject>

<optional body>

<optional footer(s)>
```

- **type** — required, lowercase, one of the allowed list below.
- **scope** — optional, lowercase letters/digits with `-`, `_`, `.`, `/`.
- **!** — optional breaking-change marker after type/scope.
- **subject** — required, at least one character on the same line.
- **`BREAKING CHANGE: …`** footer also signals a breaking change.

## Allowed types

`feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`.

To change the list, edit the regex at the top of [`hack/hooks/commit-msg`](hack/hooks/commit-msg) and update this doc.

## Examples

```
feat(proxy): add audience override
fix: handle empty STS response
docs(releasing): document rc workflow
refactor!: drop legacy header passthrough
chore(deps): bump aws-sdk-go-v2

feat(token): cache audiences per host

Adds an LRU keyed by upstream host so concurrent requests to the same
host share a single STS lookup.

BREAKING CHANGE: --token-cache-size now controls per-host buckets.
```

Merge commits (`Merge …`), reverts (`Revert "…"`), and `fixup!`/`squash!` autosquash messages are passed through unchanged.

## Bypassing

`git commit --no-verify` skips the hook. Avoid it: a non-conforming commit will not contribute to the next version bump, so a feature could ship as a patch release.

## How this drives releases

See [RELEASING.md](RELEASING.md) for the full release workflow. The short version:

| Commit shape since last tag | Resulting bump |
| --- | --- |
| any `feat:` (no breaking) | minor |
| only `fix:` / others | patch |
| any `…!:` or `BREAKING CHANGE:` footer | major |
