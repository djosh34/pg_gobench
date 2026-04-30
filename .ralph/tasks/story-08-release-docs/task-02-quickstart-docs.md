## Task: 02 Write Quick Start Documentation <status>done</status> <passes>true</passes>

<blocked_by>.ralph/tasks/story-08-release-docs/task-01-github-actions-multiarch-ghcr.md</blocked_by>

<description>
**Goal:** Add concise quick start documentation showing how to configure, run, control, and observe `pg_gobench`.

The docs must include a full YAML config example with `source.host`, `source.port`, `source.username`, `source.password`, `source.dbname`, and TLS path fields. Show both `env-ref` and `secret-file` examples for username/password. Make it clear that application config is file-based and that environment variables are only supported when explicitly referenced by username/password `env-ref`.

Include examples for Docker Compose startup, running the scratch image, starting a benchmark with curl, altering a benchmark with curl, stopping a benchmark with curl, viewing JSON state/results, viewing `/metrics`, and interpreting basic stats such as p95, p99, and TPS.

This is a non-code documentation task. Do not use TDD for this task. Verification must execute or dry-run the documented commands where possible.
</description>

<acceptance_criteria>
- [x] Docs include a complete YAML config example.
- [x] Docs explain `value`, `env-ref`, and `secret-file` for username/password.
- [x] Docs clearly state that connection strings and general env-var config are not supported.
- [x] Docs include Docker Compose quick start instructions.
- [x] Docs include scratch image run instructions.
- [x] Docs include curl examples for start, alter, stop, state/results, health/readiness, and metrics.
- [x] Docs describe p95, p99, TPS, errors, active clients, and operation counts.
- [x] Manual verification: documented local or Docker Compose quick start commands are executed successfully, or any environmental failure is recorded as a real bug task rather than ignored.
</acceptance_criteria>

<plan>.ralph/tasks/story-08-release-docs/task-02-quickstart-docs_plans/2026-04-30-quickstart-docs-plan.md</plan>
