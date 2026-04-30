## Bug: CI image pipeline uses QEMU and serializes image build behind validation <status>done</status> <passes>true</passes> <priority>high</priority>

<description>
The CI image pipeline currently uses QEMU for ARM builds and only starts the image build after validation has completed.

This is broken because the project should use a native ARM runner instead of QEMU emulation, and validation, image build, and the third required pipeline task should all run in parallel. Export/publish must remain gated until all three prerequisite jobs have completed successfully, so artifacts are only exported when every required task is non-failing.
</description>

<mandatory_manual_verification>
This is a workflow/pipeline configuration task. Do not use TDD for this bug.

Verify the fixed workflow manually by inspecting the CI graph and checking an authenticated workflow run with `github-api-curl` or an equivalent GitHub workflow log command. The verification must prove that QEMU setup is gone, ARM image work runs on a native ARM runner, the three prerequisite jobs run in parallel, and export/publish waits for all three jobs to succeed.
</mandatory_manual_verification>

<acceptance_criteria>
- [x] QEMU setup/emulation is removed from the image pipeline.
- [x] ARM image build uses a native ARM runner.
- [x] Validation, image build, and the third required pipeline task run in parallel.
- [x] Export/publish runs only after all three prerequisite tasks are complete and non-failing.
- [x] GitHub workflow logs or CI graph were manually verified with authenticated access.
- [x] `make check` — passes cleanly.
- [x] `make lint` — passes cleanly.
</acceptance_criteria>

<plan>.ralph/tasks/bugs/bug-ci-image-pipeline-uses-qemu-and-serial-build_plans/2026-04-30-ci-image-pipeline-native-arm-plan.md</plan>
