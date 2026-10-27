# ElderlyGate/.github

The org's shared release pipeline. A merge to `main` releases by itself (V-7): nobody tags by hand.

## How it works

1. A pull request is squash-merged into `main` (G-GIT-5). Its title is the commit subject, so it is a Conventional Commit.
2. The product's `release.yml` runs on that push. It runs the product's own `ci.yml` first; a red test means no release.
3. It then calls `.github/workflows/release.yml` from this repository, which:
   - finds the highest `vX.Y.Z` tag behind `main` (rc and other tags are ignored);
   - reads the commit subjects since then, and bodies for `BREAKING CHANGE:`;
   - picks the next version by V-3 (table below), or stops when nothing is due;
   - checks the version file, when the product has one (V-6);
   - pushes one annotated tag `vX.Y.Z` (V-2);
   - builds and publishes the GitHub release in the same run, with the commit subjects grouped by type as notes.

A tag pushed with `GITHUB_TOKEN` starts no other workflow, so everything happens in this one run.

| Commit since the last tag | Release |
|---|---|
| `!` after the type or scope (`feat!:`, `fix(cli)!:`), or a `BREAKING CHANGE:` line in the body | MAJOR |
| `feat` | MINOR |
| `fix`, `perf`, `revert` | PATCH |
| `docs`, `chore`, `style`, `test`, `ci`, `build`, `refactor`, or not Conventional | none |

The highest one wins. No tag yet: the first release is `v0.1.0`.

- `perf` is a patch: users get a faster product, which is a change they can see.
- `refactor` is no release: by its meaning nothing changes for users. If it does, it is a `fix` or `feat`.
- `revert` is a patch: it takes back something users already have.
- A breaking change in `v0.x` goes to `v1.0.0`. V-3 has no exception for `v0.x`, and a breaking change should be seen.
- A Go module going to `v2.0.0` or above must also change its module path to `.../v2` in that pull request.

The script is `.github/scripts/next-version.sh`. Test it with `bash .github/scripts/next-version_test.sh`.

## Inputs

| Input | Default | What it does |
|---|---|---|
| `build` | `none` | `goreleaser` runs `goreleaser release --clean`; `script` runs `script`; `none` makes the tag and the release only |
| `script` | | The command for `build: script`. It runs after the tag is pushed, with `TAG` set, before the release is published |
| `version-file` | | A file that repeats the version (V-6). The release stops when it differs |
| `go-version-file` | `go.mod` | The Go version for goreleaser |
| `node-version` | | A Node version to set up before the build (kanban-app's goreleaser builds the UI) |
| `rc` | `false` | Make a release candidate `vX.Y.Z-rc.N` instead |

Output: `version`, the tag that was released, or empty.

## The version file (V-6)

The pipeline never commits to `main`, so it cannot write a new version into a file. The best way is to **drop the file**: the tag is the source of truth (V-1), the build stamps the tag into the binary (V-5), and with no file repeating the version, V-6 has nothing to check.

Until a product has dropped its file, set `version-file`. Then the pull request that brings a `feat` or `fix` also sets the file to the version it will release. If it does not, the release stops with a clear message: merge a pull request `chore: set VERSION to X.Y.Z`, and the release follows that merge, with the commits from both.

## Adopting it in a product

1. Copy the template that fits into `.github/workflows/release.yml`, replacing the tag-triggered one.
2. Give `ci.yml` the `workflow_call` trigger and remove its `push: main` trigger, since `release.yml` now runs it on `main`.
3. A goreleaser product sets `release: prerelease: auto` in `.goreleaser.yaml`, so an rc is a pre-release (V-8).

| Template | For |
|---|---|
| `templates/release-go.yml` | Go products with goreleaser |
| `templates/release-library.yml` | A library: tag and notes, no binaries |
| `templates/release-e-designer.yml` | e-designer: gate is `pnpm check`, `VERSION` checked |
| `templates/release-docs.yml` | Docs only, like the org constitution: no tests |
| `templates/ci-go.yml` | A Go product with no `ci.yml` yet |

## What the owner sets up

- Create the private repository `ElderlyGate/.github` and push this one.
- In this repository: Settings, Actions, General, Access: "Accessible from repositories in the 'ElderlyGate' organization". Without it, other repositories cannot call the workflow or its action.
- In each product, or for the org: Settings, Actions, General, Workflow permissions. The workflow asks for `contents: write` itself, so "Read repository contents" as default is fine, but the org must not forbid write.
- The tag ruleset must let GitHub Actions create `v*` tags. It already protects them from update and deletion.

## Release candidate

1. Merge the pull requests with `[skip release]` in the merge commit message (in the body, not the title), so `main` holds the changes without a release.
2. Actions, release, Run workflow on `main`, tick `rc`. It tags `vX.Y.Z-rc.1` and publishes a pre-release. Run it again after more merges for `rc.2`.
3. Run it without `rc`, or merge the next pull request, and `vX.Y.Z` is released.

## Skip a release

Put `[skip release]` in the merge commit message. The tests still run; the commits wait and are released with the next merge that is not skipped.

## A failed run

A run that stops after pushing the tag can be run again: it finds the tag on the same commit and goes on to publish the release. A release that already exists is never made twice.
