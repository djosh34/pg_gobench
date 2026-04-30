## Task: 03 Upgrade Project Toolchain To Go 1.26.2 <status>done</status> <passes>true</passes>

<blocked_by>.ralph/tasks/story-08-release-docs/task-01-github-actions-multiarch-ghcr.md</blocked_by>

<description>
**Goal:** Upgrade the project from Go 1.25.5 to the latest stable Go release, Go 1.26.2, and prove the release pipeline still works end to end.

The latest Go version was checked against the official Go release history on 2026-04-30: https://go.dev/doc/devel/release lists `go1.26.2` as released on 2026-04-07. Update every repository-owned Go toolchain reference that controls local builds, Docker builds, or CI builds so the project consistently uses Go 1.26.2. Known current references include `go.mod` and `Dockerfile`; inspect the repository for any additional Go version pins before changing anything.

This task is not complete merely because version strings were changed. It is complete only after local validation passes and after a GitHub push has successfully run the GHCR publish workflow for the upgraded commit. Verify the GHCR push using authenticated GitHub workflow logs and/or package/image metadata; do not assume success from local Docker builds alone.

This is a toolchain, Dockerfile, and workflow verification task. Do not use the TDD skill for this task because the requested work is version/configuration/release plumbing rather than application behavior. Do not skip tests or lint. Do not ignore any errors; any unrelated blocking error must be reported with an add-bug task before marking this task complete.
</description>

<acceptance_criteria>
- [x] Repository-owned Go toolchain references are updated from Go 1.25.5 to Go 1.26.2, including `go.mod`, `Dockerfile`, and any CI/tooling version pins found during inspection.
- [x] The upgrade does not leave stale Go 1.25.5 references in build, CI, release, or documentation paths unless they are historical examples explicitly irrelevant to runtime/build behavior.
- [x] `make check` — passes cleanly.
- [x] `make test` — passes cleanly.
- [x] `make lint` — passes cleanly.
- [x] Docker build verification for the upgraded `Dockerfile` passes cleanly.
- [x] A commit containing the Go 1.26.2 upgrade is pushed to GitHub and the GHCR publish workflow runs for that pushed commit.
- [x] Authenticated GitHub verification using `/home/joshazimullah.linux/github-api-curl` or equivalent confirms the GHCR workflow completed successfully for the upgraded commit.
- [x] GHCR/package verification confirms the upgraded commit's image tag was pushed successfully to `ghcr.io/<owner>/<repo>:<github-sha>`.
</acceptance_criteria>

<plan>.ralph/tasks/story-08-release-docs/task-03-upgrade-to-go-1-26-2_plans/2026-04-30-upgrade-to-go-1-26-2-plan.md</plan>
