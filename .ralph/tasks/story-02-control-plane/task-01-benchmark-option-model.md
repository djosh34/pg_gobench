## Task: 01 Define Benchmark Option Model And Profiles <status>not_started</status> <passes>false</passes>

<blocked_by>.ralph/tasks/story-01-foundation/task-03-database-sql-connector.md</blocked_by>

<description>
Must use tdd skill to complete

**Goal:** Define the JSON option model used by the HTTP API to start and alter a PostgreSQL benchmark. Before finalizing the option set, look online for inspiration from existing PostgreSQL/database benchmarking tools such as `pgbench`, HammerDB, sysbench, and PostgreSQL benchmarking documentation. Use that research to keep the API small but useful.

The start request must support differing benchmark levels, including at minimum: benchmark size/scale, number of clients, duration, warmup duration, workload/profile selection, read/write mix where applicable, transaction mix where applicable, and optional target TPS/rate limiting. Include validation rules and defaults. The accepted profiles should cover useful categories such as `read`, `write`, `transaction`, `join`, `lock`, and `mixed`.

Alter requests must be intentionally narrower than start requests. At minimum they must allow changing client count and target TPS/rate where safe. They must not allow changing schema scale, database connection settings, or destructive setup behavior while a benchmark is running.

This is a greenfield API. Keep request JSON ultra simple and do not add compatibility aliases.
</description>

<acceptance_criteria>
- [ ] TDD red/green coverage exists for start option defaults and validation.
- [ ] TDD red/green coverage exists for rejecting unknown JSON fields.
- [ ] TDD red/green coverage exists for allowed and rejected alter-request fields.
- [ ] TDD red/green coverage exists for benchmark scale and clients validation.
- [ ] The task implementation records the online benchmarking inspiration in a concise code comment or docs note where future maintainers can see why the final option set exists.
- [ ] The option model is simple JSON and does not require nested compatibility shapes.
- [ ] `make check` — passes cleanly
- [ ] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [ ] `make lint` — passes cleanly
- [ ] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>
