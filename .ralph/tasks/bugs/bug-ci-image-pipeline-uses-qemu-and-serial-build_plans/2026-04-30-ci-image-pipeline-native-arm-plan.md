# CI Image Pipeline Native ARM Plan

Plan file: `.ralph/tasks/bugs/bug-ci-image-pipeline-uses-qemu-and-serial-build_plans/2026-04-30-ci-image-pipeline-native-arm-plan.md`

## Scope

Fix `.github/workflows/publish-ghcr.yml` so the release pipeline no longer uses QEMU for `linux/arm64`, no longer serializes image work behind one combined validation job, and still publishes only after every prerequisite lane has passed.

This is a workflow task, so the `tdd` skill applies as a planning/quality mindset only. It is a TDD exception for test shape: verification must execute the real workflow and inspect the actual CI graph rather than adding brittle tests against YAML text.

This task should deliver:

- one workflow edit in `.github/workflows/publish-ghcr.yml`
- three prerequisite pipeline lanes that can run in parallel: `check`, `test`, and `build`
- a native ARM build leg on GitHub-hosted ARM instead of emulation
- a final `publish` job gated on all prerequisite lanes succeeding
- manual verification through authenticated GitHub run inspection plus local workflow validation

This task should not deliver:

- any QEMU setup step
- any second workflow file or helper script to orchestrate release logic
- any fake tests that assert workflow YAML strings
- any `make test-long` or e2e lane

## Public Interface

Keep the release contract explicit and small:

- workflow file: `.github/workflows/publish-ghcr.yml`
- workflow triggers remain `push` and `workflow_dispatch`
- published image remains `ghcr.io/${{ github.repository }}:${{ github.sha }}`
- per-platform builds continue to publish by digest and hand digests to the manifest step

No application code or runtime interface changes are planned.

## Boundary Decision

The `improve-code-boundaries` move here is to flatten one muddy CI boundary: the current workflow mixes repository validation into one serial gate that unnecessarily blocks image work, and it relies on QEMU to hide the architecture boundary instead of assigning ARM work to an ARM runner.

Planned responsibilities:

- `check` owns repository formatting and vetting via `make check`
- `test` owns repository test execution via `make test`
- `build` owns per-platform image builds and digest artifact handoff only
- `publish` owns final manifest assembly and canonical SHA tagging only

This keeps the workflow from regressing into shell-driven spaghetti:

- no combined validation-plus-build job
- no duplicated publish/tagging logic outside the final manifest job
- no architecture emulation layer inside the workflow
- no extra permanent architecture tags beyond digest pushes

## Planned Workflow Shape

Planned jobs:

- `check`
  - `runs-on: ubuntu-latest`
  - `actions/checkout`
  - `actions/setup-go`
  - run `make check`
- `test`
  - `runs-on: ubuntu-latest`
  - `actions/checkout`
  - `actions/setup-go`
  - run `make test`
- `build`
  - no `needs`; it must start in parallel with `check` and `test`
  - matrix include:
    - `linux/amd64` on `ubuntu-latest`
    - `linux/arm64` on `ubuntu-24.04-arm`
  - `actions/checkout`
  - `docker/setup-buildx-action`
  - `docker/login-action` against `ghcr.io`
  - `docker/build-push-action` with per-leg `platforms: ${{ matrix.platform }}`
  - upload per-leg digest artifacts
- `publish`
  - `needs: [check, test, build]`
  - download digest artifacts
  - `docker buildx imagetools create` to assign the canonical SHA tag
  - `docker buildx imagetools inspect` as the in-workflow proof the manifest exists

Explicit removals:

- delete `docker/setup-qemu-action`
- do not keep `needs: validate` on image build
- do not collapse `check` and `test` back into one prerequisite job

## Execution Slices

Because this is a workflow task, execute in real verification slices rather than unit-test slices:

- [x] Slice 1: refactor the workflow job graph from `validate -> build -> publish` into `check || test || build -> publish`
- [x] Slice 2: remove QEMU setup and assign the ARM matrix leg to `ubuntu-24.04-arm`
- [x] Slice 3: keep per-platform digest upload/download flow intact while updating the job dependencies
- [x] Slice 4: validate workflow syntax locally with `actionlint` if available
- [x] Slice 5: run `make check`, `make lint`, and `make test` locally as the required repository gates
- [x] Slice 6: push the branch and inspect the authenticated GitHub run to confirm the graph and runner usage are correct
- [x] Slice 7: confirm the run shows `check`, `test`, and `build` starting independently, the ARM leg running on a native ARM runner, and `publish` waiting for all three prerequisites
- [x] Slice 8: do one final `improve-code-boundaries` pass to ensure the workflow still has one clear owner for validation, build, and publish responsibilities

## Verification Strategy

Local validation during execution:

```bash
/home/joshazimullah.linux/go/bin/actionlint .github/workflows/publish-ghcr.yml
make check
make lint
make test
```

Remote validation during execution:

```bash
git push
/home/joshazimullah.linux/github-api-curl repos/<owner>/<repo>/actions/runs?event=push
/home/joshazimullah.linux/github-api-curl repos/<owner>/<repo>/actions/runs/<run-id>/jobs
```

What must be proven from the authenticated run:

- no QEMU setup step exists in the workflow graph or job logs
- the `build` matrix contains one x64 leg and one ARM leg
- the ARM leg runs on a native ARM runner label, specifically `ubuntu-24.04-arm`
- `check`, `test`, and `build` begin without dependency edges between them
- `publish` waits on all three prerequisites and does not start after only one or two of them

If GitHub run inspection or runner allocation reveals a real infrastructure blocker, do not hand-wave it away. Either fix the workflow honestly or create an `add-bug` task for the blocker before claiming completion.

## Implementation Notes

Expected file changes:

- update `.github/workflows/publish-ghcr.yml`
- update this bug task as boxes are completed

Avoid these muddy shortcuts:

- no helper scripts for workflow orchestration
- no arch-specific final tags
- no second manifest-assembly path
- no silent fallback back to emulation

If execution reveals that `ubuntu-24.04-arm` is unavailable to this repository at runtime, switch this plan back to `TO BE VERIFIED` immediately and capture the exact blocker rather than forcing QEMU back into the pipeline.

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long`
- local workflow validation with `actionlint`
- authenticated GitHub Actions run inspection proving the graph and native ARM runner behavior

NOW EXECUTE
