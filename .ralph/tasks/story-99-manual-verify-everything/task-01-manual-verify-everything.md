## Task: 01 Manual Verify Everything End To End <status>not_started</status> <passes>false</passes>

<blocked_by>.ralph/tasks/story-09-ui/task-01-standalone-html-control-page.md</blocked_by>

<description>
**Goal:** Perform the final manual verification pass for the whole project. Forget that automated tests exist for this task. Do not run tests for this task. This task exists to prove the shipped product actually works by using it like a user.

First, compile a complete list of all implemented features by reading every task file under `.ralph/tasks`. For each feature, add a `- [ ]` checklist entry directly in this task file before starting manual verification. Then manually test every listed feature against the real built service, real PostgreSQL, real scratch image, real Docker Compose example, real local Kubernetes deployment, real Prometheus metrics endpoint, real GitHub workflow behavior where possible, and the standalone HTML file.

If any problem is found, immediately create an add-bug task. Do not keep manually verifying unrelated features after finding a real bug if the bug should be fixed first. Run task switching much earlier so Ralph switches to the bug task. This manual verification task can become `<passes>true</passes>` only after the checklist is complete and all bug tasks created from this manual pass are fixed.

This is a non-code manual verification task. Do not use TDD. Do not run `make test`, `make check`, or `make lint` for this task.
</description>

<manual_feature_checklist>
Populate this section at the start of the task by reading every task file under `.ralph/tasks`. Add one `- [ ]` entry for each feature that must be manually verified. Do not mark this task done until every generated feature entry is checked and every bug found during this pass is fixed.
</manual_feature_checklist>

<acceptance_criteria>
- [ ] Compile a complete feature checklist from all task files under `.ralph/tasks` and add each feature as a `- [ ]` entry in this task file.
- [ ] Manually verify every feature checklist entry using the running product rather than automated tests.
- [ ] Manually verify config loading, username/password `value`, `env-ref`, `secret-file`, TLS path validation, and strict rejection behavior.
- [ ] Manually verify HTTP bind address, `/healthz`, `/readyz`, JSON benchmark start/alter/stop/state/results, error JSON, and single-active-run rejection.
- [ ] Manually verify benchmark schema setup, scale behavior, core workloads, advanced workloads, p95/p99/TPS stats, and same-shape stats across profiles.
- [ ] Manually verify `/metrics` Prometheus output and `pg_gobench_` metric names.
- [ ] Manually verify the scratch image, Docker Compose example, Kubernetes one-apply example, GHCR multi-arch workflow behavior where possible, quick start docs, and standalone HTML page.
- [ ] For every failure, immediately create an add-bug task with enough detail to reproduce it.
- [ ] If any bug task is created, stop this task and run task switching so the bug task is selected before continuing manual verification.
- [ ] Mark this task `<passes>true</passes>` only after every checklist item is checked and every bug created during this pass is fixed.
</acceptance_criteria>
