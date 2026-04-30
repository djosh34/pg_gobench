# GitHub Actions Multi-Arch GHCR Publish Plan

Plan file: `.ralph/tasks/story-08-release-docs/task-01-github-actions-multiarch-ghcr_plans/2026-04-30-github-actions-multiarch-ghcr-plan.md`

## Scope

Add one GitHub Actions workflow that validates the repository, builds the existing scratch container image for `linux/amd64` and `linux/arm64` in parallel, and publishes exactly one final multi-platform GHCR tag named from the commit SHA.

This turn is planning-only. Execution belongs to the next turn after the plan is reviewed and promoted to `NOW EXECUTE`.

This task should deliver:

- one workflow file under `.github/workflows/`
- repository validation before any publish step
- parallel per-platform image builds
- one final published image reference of the form `ghcr.io/<owner>/<repo>:<github-sha>`
- real workflow verification via `actionlint` and a pushed GitHub Actions run
- real manifest verification showing both `linux/amd64` and `linux/arm64`

This task should not deliver:

- any new registry credentials beyond `GITHUB_TOKEN`
- any final image tag ending in `-amd64` or `-arm64`
- helper scripts for tag construction or workflow orchestration
- fake tests that only assert workflow YAML text
- `make test-long` unless this later becomes a story-finishing validation turn

## Public Interface

Keep the CI contract explicit and minimal:

- workflow file:
  - `.github/workflows/publish-ghcr.yml`
- workflow trigger:
  - `push` so a normal branch push can exercise the workflow
  - `workflow_dispatch` for explicit reruns during manual verification
- validation commands before publish:
  - `make check`
  - `make test`
- published image name:
  - `ghcr.io/${{ github.repository }}:${{ github.sha }}`
- canonical registry repository:
  - `ghcr.io/${{ github.repository }}`

No application interface changes are planned. The workflow should package the existing `Dockerfile` contract exactly as-is.

## Boundary Decision

The `improve-code-boundaries` move for this task is to keep validation, per-platform build work, and final release tagging in separate workflow responsibilities instead of mixing them into one large shell-driven job.

Planned shape:

- `validate` owns repository checks only
- `build` owns per-platform container builds only
- `publish` owns manifest assembly and final tag publication only
- the image repository and SHA tag are derived once from GitHub context and reused everywhere

This avoids the main muddy workflow risks:

- no duplicated string-splitting logic for owner, repo, or tag naming
- no arch-suffixed final release tags
- no giant job that both tests, builds, tags, and publishes
- no parallel ad-hoc release paths outside `.github/workflows/`

One concrete boundary simplification is planned:

- build jobs should push by digest without inventing permanent per-arch tags
- the manifest job should be the only place that assigns the canonical `:${{ github.sha }}` tag

If execution shows that GitHub-hosted Actions cannot cleanly create the final manifest from per-platform digests with this shape, switch the plan back to `TO BE VERIFIED` immediately instead of forcing a stringly or duplicated release flow.

## Planned Workflow Shape

Planned jobs:

- `validate`
  - `actions/checkout`
  - `actions/setup-go`
  - run `make check`
  - run `make test`
- `build`
  - `needs: validate`
  - matrix over:
    - `linux/amd64`
    - `linux/arm64`
  - `actions/checkout`
  - `docker/setup-qemu-action`
  - `docker/setup-buildx-action`
  - `docker/login-action` against `ghcr.io` using `github.actor` and `secrets.GITHUB_TOKEN`
  - `docker/build-push-action` with:
    - `context: .`
    - `platforms: ${{ matrix.platform }}`
    - `push: true`
    - push-by-digest output so each matrix leg publishes immutable content without creating the final tag
  - persist each produced digest as a small artifact for the manifest job
- `publish`
  - `needs: build`
  - download the digest artifacts
  - log in to GHCR again with the workflow token
  - run `docker buildx imagetools create -t ghcr.io/${{ github.repository }}:${{ github.sha }} ...`
  - run `docker buildx imagetools inspect ghcr.io/${{ github.repository }}:${{ github.sha }}` as an in-workflow sanity check

Planned permissions:

- top-level or job-level `contents: read`
- top-level or job-level `packages: write`

No broader permissions are planned.

## Verification Strategy

This task is a TDD exception because it is a workflow task. Verification must execute the real workflow behavior rather than asserting YAML strings.

Planned execution slices:

- [ ] Slice 1: add the workflow file and validate local syntax with `actionlint`
- [ ] Slice 2: run local repo gates honestly with `make check`, `make lint`, and `make test`
- [ ] Slice 3: push the branch so GitHub Actions runs the workflow for a real commit SHA
- [ ] Slice 4: inspect the workflow run and confirm `validate` completes before any publish work starts
- [ ] Slice 5: confirm both platform build matrix legs execute and complete successfully
- [ ] Slice 6: confirm the final publish job creates exactly one canonical SHA tag without `-amd64` or `-arm64`
- [ ] Slice 7: inspect the published manifest and verify it reports both `linux/amd64` and `linux/arm64`
- [ ] Slice 8: do one final `improve-code-boundaries` pass to ensure no duplicate tagging logic or helper-script release path was introduced

Planned verification commands during execution:

```bash
/home/joshazimullah.linux/go/bin/actionlint .github/workflows/publish-ghcr.yml
make check
make lint
make test
git push
docker buildx imagetools inspect ghcr.io/djosh34/pg_gobench:<commit-sha>
```

Planned GitHub run inspection direction:

- use the pushed commit SHA to find the matching Actions run
- use `/home/joshazimullah.linux/github-api-curl` to inspect run status and job/log metadata if the web UI is not available
- confirm the run graph shows `validate` gating the downstream build/publish path

Planned manifest verification direction:

- prefer `docker buildx imagetools inspect ghcr.io/djosh34/pg_gobench:<commit-sha>`
- confirm both `linux/amd64` and `linux/arm64` are present in the manifest output
- if registry visibility or auth prevents honest platform verification, switch back to `TO BE VERIFIED` instead of hand-waving the acceptance criterion

## Implementation Notes

Expected file changes:

- add `.github/workflows/publish-ghcr.yml`
- no Go code changes are expected in the intended design
- no docs changes are expected unless execution reveals the workflow needs a tiny operator note

Avoid these muddy shortcuts:

- no shell script committed just to wrap buildx commands
- no duplicated `ghcr.io/<owner>/<repo>` construction in multiple ad-hoc places
- no permanent arch-specific release tags as the published interface
- no bypass of repository validation before publish

If execution reveals a real gap, fix the smallest honest layer:

- workflow orchestration issue -> fix in `.github/workflows/publish-ghcr.yml`
- image build issue -> fix in `Dockerfile`
- genuine application packaging problem -> only then consider Go or build input changes

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless this becomes the story-finishing turn or the task text changes
- `actionlint` validation is required for this workflow task
- one real pushed GitHub Actions run plus remote manifest inspection are required for this task

If execution shows that the chosen trigger, digest handoff, or GHCR verification path creates a muddier boundary than expected, switch this plan back to `TO BE VERIFIED` instead of forcing a second release path.

NOW EXECUTE
