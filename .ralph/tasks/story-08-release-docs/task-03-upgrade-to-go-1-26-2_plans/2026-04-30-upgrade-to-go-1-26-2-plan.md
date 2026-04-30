# Go 1.26.2 Toolchain Upgrade Plan

Plan file: `.ralph/tasks/story-08-release-docs/task-03-upgrade-to-go-1-26-2_plans/2026-04-30-upgrade-to-go-1-26-2-plan.md`

## Scope

Upgrade the repository-owned Go toolchain from `1.25.5` to `1.26.2` and prove the release path still works all the way through a pushed GHCR publish for the upgraded commit.

This turn is planning-only. Execution belongs to the next turn after this plan is promoted to `NOW EXECUTE`.

This task should deliver:

- one canonical Go version in `go.mod`
- one matching Go builder image version in `Dockerfile`
- no stale `1.25.5` references in repository-owned build, CI, release, or operator-facing docs that affect the active workflow
- honest local validation with `make check`, `make lint`, `make test`, and a real Docker build
- one pushed GitHub commit that runs the GHCR publish workflow for the upgraded commit
- authenticated verification that the workflow succeeded and that the SHA-tagged image was published to GHCR

This task should not deliver:

- a second toolchain-version file that duplicates `go.mod`
- fake tests that only assert version strings in config files
- `make test-long` unless this later becomes the story-finishing turn
- hand-waved remote verification based only on local Docker success

## Public Interface

No application behavior or API contract changes are planned.

The public build/release interfaces that matter are:

- `go.mod` as the canonical Go toolchain source for local development and CI
- `Dockerfile` as the canonical container build toolchain source
- `.github/workflows/publish-ghcr.yml` as the canonical remote release path
- the published container tag `ghcr.io/<owner>/<repo>:<github-sha>`

Current observed version boundaries:

- `go.mod` currently pins `go 1.25.5`
- `Dockerfile` currently uses `golang:1.25.5-bookworm`
- GitHub Actions already derives its Go version from `go.mod` through `actions/setup-go` with `go-version-file: go.mod`

That CI shape is already the correct boundary and should be preserved rather than widened.

## Boundary Decision

The `improve-code-boundaries` move for this task is to keep the toolchain version owned in the minimum number of places and to avoid inventing new glue for CI or Docker builds.

Planned boundary shape:

- `go.mod` remains the single source of truth for the Go toolchain used by local commands and the validation job in CI
- `Dockerfile` keeps its one explicit builder-image version because the container build cannot consume `go.mod` directly
- `.github/workflows/publish-ghcr.yml` continues to consume `go.mod` instead of gaining a second hard-coded Go version

This avoids the main smells:

- no duplicate CI version pin beside `go.mod`
- no shell wrapper or helper file that exists only to pass a version string around
- no split between local/CI Go version and Docker build Go version
- no release-path verification outside the existing GHCR workflow

One concrete boundary simplification is planned:

- if execution finds any independent Go version pin in workflow or tooling files, collapse it into the existing `go.mod`-driven CI boundary instead of keeping multiple owned sources

If execution shows that Go `1.26.2` requires a wider packaging or dependency boundary change than this task anticipates, switch this plan back to `TO BE VERIFIED` immediately instead of forcing a muddy partial upgrade.

## Planned Change Shape

Planned repository inspection and edit order:

- [x] Slice 1: re-scan the repository for all owned `1.25.5` or `1.26.2` references before editing
- [x] Slice 2: update `go.mod` from `go 1.25.5` to `go 1.26.2`
- [x] Slice 3: update `Dockerfile` from `golang:1.25.5-bookworm` to `golang:1.26.2-bookworm`
- [x] Slice 4: update any additional owned build or release pin discovered during execution, but only if it is a real active boundary
- [x] Slice 5: remove or rewrite any stale active docs/build references that still claim `1.25.5`
- [x] Slice 6: do one final boundary pass to confirm CI still derives Go from `go.mod` and no duplicate version ownership was introduced

Expected file changes:

- `go.mod`
- `Dockerfile`
- possibly one small workflow or doc edit only if execution uncovers a real stale active reference

Files that should remain untouched unless execution proves otherwise:

- `.github/workflows/publish-ghcr.yml` because it already uses `go-version-file: go.mod`
- application Go source, unless the new toolchain exposes a real compile or vet issue that must be fixed honestly

## Verification Strategy

This task is a TDD exception because it is toolchain, Dockerfile, and release plumbing work. Verification must execute the real build and release paths rather than add brittle tests for version strings.

Planned execution slices:

- [x] Slice 1: confirm the repository search finds the full set of active version references before edits
- [x] Slice 2: make the minimal version-pin edits
- [x] Slice 3: run `make check`
- [x] Slice 4: run `make lint`
- [x] Slice 5: run `make test`
- [x] Slice 6: run a real Docker build against the upgraded `Dockerfile`
- [x] Slice 7: push the upgraded commit to GitHub so the publish workflow runs for the exact commit SHA
- [x] Slice 8: use authenticated GitHub inspection to confirm the workflow run for that commit completed successfully
- [x] Slice 9: verify the GHCR image `ghcr.io/<owner>/<repo>:<github-sha>` exists for the pushed commit
- [x] Slice 10: re-check the boundary shape and revert this plan to `TO BE VERIFIED` if the upgrade required a wider redesign

Planned verification commands during execution:

```bash
rg -n "1\\.25\\.5|1\\.26\\.2|go1\\.25|go1\\.26|golang:|setup-go|toolchain go|^go [0-9]" -S .
make check
make lint
make test
docker build -t pg_gobench:go-1.26.2 .
git push
/home/joshazimullah.linux/github-api-curl /repos/<owner>/<repo>/actions/runs?head_sha=<commit-sha>
docker buildx imagetools inspect ghcr.io/<owner>/<repo>:<commit-sha>
```

Planned GitHub verification direction:

- identify the pushed commit SHA with `git rev-parse HEAD`
- inspect the matching Actions run for that exact SHA
- confirm the `Publish GHCR Image` workflow completed successfully for the upgraded commit
- if needed, inspect workflow jobs or logs through authenticated GitHub API calls rather than guessing from local state

Planned GHCR verification direction:

- inspect `ghcr.io/<owner>/<repo>:<commit-sha>` after the workflow succeeds
- confirm the image tag exists for the pushed SHA
- if manifest inspection requires auth, authenticate honestly rather than treating a missing inspect result as success

If remote workflow or package verification is blocked by a real infrastructure issue, create an `add-bug` task before claiming completion. Do not wave it away.

## Implementation Notes

Avoid these muddy shortcuts:

- do not add a repo-local version constant, script, or Makefile variable just to echo the Go version
- do not hard-code `1.26.2` into the workflow if `go.mod` can remain the authoritative CI source
- do not skip local checks because the task is "just" a version upgrade
- do not treat a local Docker build as proof that GHCR publication worked

If execution reveals a real issue, fix the smallest honest layer:

- stale version pin -> update the owning file
- Go toolchain behavior change -> fix the real compile, vet, or test failure
- Docker build failure -> fix the Docker/build boundary
- workflow or GHCR failure -> fix the release workflow or create an `add-bug` task if blocked externally

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless this becomes the story-finishing turn or the task text changes
- one real Docker build for the upgraded image
- one real pushed GitHub Actions run for the upgraded commit
- authenticated GHCR/image verification for the pushed commit SHA

If execution shows the chosen version-ownership boundary is wrong, especially if CI or Docker needs a second independent toolchain source to stay honest, switch this plan back to `TO BE VERIFIED` immediately instead of forcing duplicated ownership.

NOW EXECUTE
