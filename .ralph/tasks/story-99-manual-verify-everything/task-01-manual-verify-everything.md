## Task: 01 Manual Verify Everything End To End <status>in_progress</status> <passes>false</passes>

<blocked_by>.ralph/tasks/story-09-ui/task-01-standalone-html-control-page.md</blocked_by>

<description>
**Goal:** Perform the final manual verification pass for the whole project. Forget that automated tests exist for this task. Do not run tests for this task. This task exists to prove the shipped product actually works by using it like a user.

First, compile a complete list of all implemented features by reading every task file under `.ralph/tasks`. For each feature, add a `- [ ]` checklist entry directly in this task file before starting manual verification. Then manually test every listed feature against the real built service, real PostgreSQL, real scratch image, real Docker Compose example, real local Kubernetes deployment, real Prometheus metrics endpoint, real GitHub workflow behavior where possible, and the standalone HTML file.

If any problem is found, immediately create an add-bug task. Do not keep manually verifying unrelated features after finding a real bug if the bug should be fixed first. Run task switching much earlier so Ralph switches to the bug task. This manual verification task can become `<passes>true</passes>` only after the checklist is complete and all bug tasks created from this manual pass are fixed.

This is a non-code manual verification task. Do not use TDD. Do not run `make test`, `make check`, or `make lint` for this task.
</description>

<manual_feature_checklist>
- [ ] Bootstrap Go HTTP service entrypoint, explicit `-addr` bind flag, graceful shutdown, and health endpoint behavior.
- [ ] Strict YAML config loading from `-config` with `value`, `env-ref`, and `secret-file` username/password sources plus TLS path handling and strict rejection behavior.
- [ ] `database/sql` PostgreSQL connector and readiness/ping behavior built from validated YAML config rather than connection strings.
- [ ] Benchmark start option model, defaults, validation, supported profiles, and constrained alter-request model.
- [ ] Single-active benchmark run coordinator state machine including start, stop, alter, visible failures, and in-memory state/results only.
- [ ] JSON HTTP API for `/benchmark/start`, `/benchmark/alter`, `/benchmark/stop`, `/benchmark`, `/benchmark/results`, `/healthz`, and `/readyz`, including compact error JSON and single-active-run rejection.
- [ ] Benchmark schema setup under `pg_gobench`, explicit reset behavior, and scale-to-data initialization behavior.
- [ ] Core workloads for point reads, range reads, inserts, updates, mixed read/write, and multi-statement transactions with duration, warmup, clients, and TPS controls.
- [ ] In-memory stats aggregation including p95, p99, TPS, operation counts, client counts, elapsed time, and same-shape stats across workloads.
- [ ] Prometheus `/metrics` output with `pg_gobench_` metric names and low-cardinality benchmark metrics.
- [ ] Scratch Docker image build and runtime behavior with explicit flags and no shell/build tools in the final image.
- [ ] Docker Compose example with PostgreSQL plus `pg_gobench`, mounted YAML config, `env-ref` and `secret-file` secret handling, and live health/readiness behavior.
- [ ] Advanced workloads for join, aggregation, lock contention, and hot-row update contention, including surfaced contention errors.
- [ ] One-apply Kubernetes deployment, ConfigMap/Secret-backed config, Service exposure, and live benchmark control through the in-cluster deployment.
- [ ] GitHub Actions workflow validation-before-publish and multi-arch GHCR image publishing behavior where locally observable.
- [ ] Quick start documentation accuracy for config, scratch image, Docker Compose, curl control flows, metrics, and stats interpretation.
- [ ] Standalone raw HTML control page opened directly from disk and controlling a live API without server-side coupling.
</manual_feature_checklist>

<acceptance_criteria>
- [x] Compile a complete feature checklist from all task files under `.ralph/tasks` and add each feature as a `- [ ]` entry in this task file.
- [ ] Manually verify every feature checklist entry using the running product rather than automated tests.
- [ ] Manually verify config loading, username/password `value`, `env-ref`, `secret-file`, TLS path validation, and strict rejection behavior.
- [ ] Manually verify HTTP bind address, `/healthz`, `/readyz`, JSON benchmark start/alter/stop/state/results, error JSON, and single-active-run rejection.
- [ ] Manually verify benchmark schema setup, scale behavior, core workloads, advanced workloads, p95/p99/TPS stats, and same-shape stats across profiles.
- [ ] Manually verify `/metrics` Prometheus output and `pg_gobench_` metric names.
- [ ] Manually verify the scratch image, Docker Compose example, Kubernetes one-apply example, GHCR multi-arch workflow behavior where possible, quick start docs, and standalone HTML page.
- [x] For every failure, immediately create an add-bug task with enough detail to reproduce it.
- [x] If any bug task is created, stop this task and run task switching so the bug task is selected before continuing manual verification.
- [ ] Mark this task `<passes>true</passes>` only after every checklist item is checked and every bug created during this pass is fixed.
</acceptance_criteria>
