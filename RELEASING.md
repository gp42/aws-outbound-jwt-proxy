# Releasing

Releases are produced exclusively by the [`release` GitHub Actions workflow](.github/workflows/release.yml). There is no automatic release on merge — a maintainer must dispatch the workflow.

## Branch determines release kind

| Dispatch from | Result |
| --- | --- |
| `main` | Stable release `vX.Y.Z`, marked **latest** on the Releases page |
| any other branch | Prerelease `vX.Y.Z-rc.N`, marked **prerelease**, does not move `latest` |

Release-candidate numbering increments automatically: dispatching the workflow twice from the same branch (without intervening commits that change the base version) produces `…-rc.1` then `…-rc.2`.

## How to cut a release

1. Make sure the branch you intend to release from is up to date and that all commits you want included have landed.
2. Open the repository on GitHub → **Actions** → **release** → **Run workflow**.
3. Choose the branch:
   - `main` for a stable release.
   - any topic branch for a prerelease.
4. The workflow's first job prints a **Release plan** summary. Cancel the run if the proposed version or kind is wrong.

## What the workflow produces

For every release:

- One binary per supported target. Current matrix:
  - `linux/amd64`
  - `linux/arm64`
  - `darwin/arm64`
- A `SHA256SUMS` file covering every binary asset.
- An annotated git tag at the built commit (`vX.Y.Z` or `vX.Y.Z-rc.N`).
- A GitHub Release whose body contains a `compare` link to the previous tag.

Each binary embeds its version. Verify with:

```sh
./aws-outbound-jwt-proxy-vX.Y.Z-linux-amd64 version
```

## Verifying downloads

```sh
sha256sum -c SHA256SUMS --ignore-missing
```

`--ignore-missing` lets you verify a single downloaded binary against the full sums file.

## Versioning

The next version is computed from [Conventional Commits](COMMIT_CONVENTIONS.md) since the previous tag, by [`huggingface/semver-release-action`](https://github.com/huggingface/semver-release-action) running in dry-run mode.

| Commit shape | Bump |
| --- | --- |
| `feat: …` | minor |
| `fix:` / `perf:` / others | patch |
| `…!:` or `BREAKING CHANGE:` footer | major |

If there are no Conventional Commits since the last tag, the workflow fails — there is nothing to release.

## Common situations

- **No commits since last release** → the workflow fails on the "Fail if no version bump" step. Add a meaningful commit and retry.
- **Wrong branch** → the workflow refuses to run on tag refs and the kind is fixed by branch name. Push your changes to the right branch and dispatch again.
- **Need to retry after a failed publish** → if a tag was already pushed before failure, delete it (`git tag -d vX.Y.Z && git push origin :refs/tags/vX.Y.Z`) before re-dispatching.
