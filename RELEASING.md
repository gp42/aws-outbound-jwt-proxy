# Releasing

Releases are produced exclusively by the [`release` GitHub Actions workflow](.github/workflows/release.yml). There is no automatic release on merge - a maintainer must dispatch the workflow.

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
- A multi-arch container image pushed to **Docker Hub** (`docker.io/gp42/aws-outbound-jwt-proxy`) and **GHCR** (`ghcr.io/gp42/aws-outbound-jwt-proxy`) under identical tags. Tag policy mirrors the release kind: stable releases push `vX.Y.Z`, `vX.Y`, `vX`, and `latest`; prereleases push only `vX.Y.Z-rc.N`. See [Container image](https://gp42.github.io/aws-outbound-jwt-proxy/docker/) for details.

## Container image publishing

The `docker` job runs at the end of the release workflow and reuses the linux binaries that were already built and uploaded by the `build` matrix - it does not recompile. Each binary is verified against the `SHA256SUMS` file attached to the GitHub Release before it enters the image, so the bytes shipped to the registries are guaranteed to match the bytes on the Release page.

### One-time setup

1. Create the Docker Hub repository `gp42/aws-outbound-jwt-proxy` and a push-scoped access token.
2. Add repository secrets:
   - `DOCKERHUB_USERNAME` - Docker Hub username.
   - `DOCKERHUB_TOKEN` - the push-scoped access token created above (used for image pushes).
   - `DOCKERHUB_PASSWORD` - the Docker Hub account password. Required by the "Sync Docker Hub Overview" step: the Docker Hub API rejects access tokens for description edits and only accepts the account password.
3. Set the repository variable `DOCKER_PUBLISH_ENABLED=true` to enable the `docker` job. (When unset, the release workflow runs as before and the `docker` job is skipped.)
4. After the first publish, link the auto-created GHCR package to this repository and set its visibility to public.

The GHCR push uses the workflow's `GITHUB_TOKEN` with `packages: write` - no extra secret is required.

### If the docker job fails

A failure in the `docker` job does **not** invalidate the GitHub Release: the git tag and Release assets are already in place by the time the `docker` job starts. To recover:

- Re-run only the failed jobs from the workflow run page.
- If a bad tag was pushed to Docker Hub or GHCR, remove it manually:
  - Docker Hub: web UI → repository → Tags → delete.
  - GHCR: `gh api -X DELETE /user/packages/container/aws-outbound-jwt-proxy/versions/<id>` (or the repo-level equivalent).

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

If there are no Conventional Commits since the last tag, the workflow fails - there is nothing to release.

## Common situations

- **No commits since last release** → the workflow fails on the "Fail if no version bump" step. Add a meaningful commit and retry.
- **Wrong branch** → the workflow refuses to run on tag refs and the kind is fixed by branch name. Push your changes to the right branch and dispatch again.
- **Need to retry after a failed publish** → if a tag was already pushed before failure, delete it (`git tag -d vX.Y.Z && git push origin :refs/tags/vX.Y.Z`) before re-dispatching.
