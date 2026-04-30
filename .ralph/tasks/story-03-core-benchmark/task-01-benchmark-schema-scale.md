## Task: 01 Create Benchmark Schema And Scale Data Setup <status>done</status> <passes>true</passes>

<blocked_by>.ralph/tasks/story-02-control-plane/task-03-http-json-api.md</blocked_by>

<description>
Must use tdd skill to complete

**Goal:** Implement the PostgreSQL benchmark schema and data setup system. All benchmark-owned database objects must live under a dedicated schema named `bench`. The service must not drop or mutate objects outside that schema.

The benchmark size/scale option must map to concrete row counts and table sizes. Document the mapping in code or docs. The setup path must be explicit and safe: initialize benchmark-owned tables and indexes, populate deterministic data needed by workloads, and optionally reset only the `bench` schema when requested by benchmark options.

This project is greenfield with no backwards compatibility requirement. Do not add legacy table names or migration compatibility paths.
</description>

<acceptance_criteria>
- [x] TDD red/green coverage exists for scale-to-row-count mapping.
- [x] TDD red/green coverage exists for generated schema SQL targeting only `bench`.
- [x] TDD red/green coverage or integration coverage exists for creating benchmark tables and indexes through `database/sql`.
- [x] Reset/destructive behavior is explicit and limited to the benchmark-owned `bench` schema.
- [x] Setup failures are returned and cause benchmark start to fail visibly.
- [x] No SQL is executed through pgx direct APIs; use `database/sql`.
- [x] `make check` — passes cleanly
- [x] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [x] `make lint` — passes cleanly
- [ ] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>

<plan>.ralph/tasks/story-03-core-benchmark/task-01-benchmark-schema-scale_plans/2026-04-30-benchmark-schema-scale-plan.md</plan>
