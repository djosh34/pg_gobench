## Task: 01 Add GitHub Actions Multi-Arch GHCR Publish Workflow <status>done</status> <passes>true</passes>

<blocked_by>.ralph/tasks/story-07-k8s/task-01-k8s-simple-deployment-configmap.md</blocked_by>

<description>
**Goal:** Add a simple GitHub Actions workflow that builds and publishes the scratch container image to GitHub Container Registry using the workflow-provided GitHub token.

The workflow must run repository validation before publishing. It must build `linux/amd64` and `linux/arm64` in parallel and publish one final multi-platform image under a single tag equal to the GitHub SHA. The final published tag must not end in `-amd64` or `-arm64`.

The expected image name is `ghcr.io/<owner>/<repo>:<github-sha>` using the current repository context. Do not add registry credentials beyond the standard GitHub token permissions needed for GHCR package publishing.

This is a non-code workflow task. Do not use TDD for this task. Verification must inspect or run the workflow behavior.
</description>

<acceptance_criteria>
- [x] Workflow runs validation before publish.
- [x] Workflow uses GitHub token permissions appropriate for GHCR publish.
- [x] Workflow builds amd64 and arm64 in parallel jobs or an equivalent parallel matrix.
- [x] Workflow combines architecture outputs into one multi-platform image tag named exactly from the GitHub SHA.
- [x] Final pushed tag does not include `-amd64` or `-arm64`.
- [x] Manual verification: workflow syntax is validated with an available local tool such as `actionlint`, or the workflow is pushed and authenticated logs are checked with `/home/joshazimullah.linux/github-api-curl`.
- [x] Manual verification: published package manifest is multi-platform and includes both amd64 and arm64.
</acceptance_criteria>

<plan>.ralph/tasks/story-08-release-docs/task-01-github-actions-multiarch-ghcr_plans/2026-04-30-github-actions-multiarch-ghcr-plan.md</plan>
