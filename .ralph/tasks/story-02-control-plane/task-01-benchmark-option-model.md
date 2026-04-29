## Task: 01 Define Benchmark Option Model And Profiles <status>done</status> <passes>true</passes>

<blocked_by>.ralph/tasks/story-01-foundation/task-03-database-sql-connector.md</blocked_by>

<description>
Must use tdd skill to complete

**Goal:** Define the JSON option model used by the HTTP API to start and alter a PostgreSQL benchmark. Before finalizing the option set, look online for inspiration from existing PostgreSQL/database benchmarking tools such as `pgbench`, HammerDB, sysbench, and PostgreSQL benchmarking documentation. Use that research to keep the API small but useful.

The start request must support differing benchmark levels, including at minimum: benchmark size/scale, number of clients, duration, warmup duration, workload/profile selection, read/write mix where applicable, transaction mix where applicable, and optional target TPS/rate limiting. Include validation rules and defaults. The accepted profiles should cover useful categories such as `read`, `write`, `transaction`, `join`, `lock`, and `mixed`.

Alter requests must be intentionally narrower than start requests. At minimum they must allow changing client count and target TPS/rate where safe. They must not allow changing schema scale, database connection settings, or destructive setup behavior while a benchmark is running.

This is a greenfield API. Keep request JSON ultra simple and do not add compatibility aliases.
</description>

<acceptance_criteria>
- [x] TDD red/green coverage exists for start option defaults and validation.
- [x] TDD red/green coverage exists for rejecting unknown JSON fields.
- [x] TDD red/green coverage exists for allowed and rejected alter-request fields.
- [x] TDD red/green coverage exists for benchmark scale and clients validation.
- [x] The task implementation records the online benchmarking inspiration in a concise code comment or docs note where future maintainers can see why the final option set exists.
- [x] The option model is simple JSON and does not require nested compatibility shapes.
- [x] `make check` — passes cleanly
- [x] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [x] `make lint` — passes cleanly
- [ ] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>

<plan>.ralph/tasks/story-02-control-plane/task-01-benchmark-option-model_plans/2026-04-30-benchmark-option-model-plan.md</plan>
