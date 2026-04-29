## Task: 01 Create Benchmark Schema And Scale Data Setup <status>not_started</status> <passes>false</passes>

<blocked_by>.ralph/tasks/story-02-control-plane/task-03-http-json-api.md</blocked_by>

<description>
Must use tdd skill to complete

**Goal:** Implement the PostgreSQL benchmark schema and data setup system. All benchmark-owned database objects must live under a dedicated schema named `pg_gobench`. The service must not drop or mutate objects outside that schema.

The benchmark size/scale option must map to concrete row counts and table sizes. Document the mapping in code or docs. The setup path must be explicit and safe: initialize benchmark-owned tables and indexes, populate deterministic data needed by workloads, and optionally reset only the `pg_gobench` schema when requested by benchmark options.

This project is greenfield with no backwards compatibility requirement. Do not add legacy table names or migration compatibility paths.
</description>

<acceptance_criteria>
- [ ] TDD red/green coverage exists for scale-to-row-count mapping.
- [ ] TDD red/green coverage exists for generated schema SQL targeting only `pg_gobench`.
- [ ] TDD red/green coverage or integration coverage exists for creating benchmark tables and indexes through `database/sql`.
- [ ] Reset/destructive behavior is explicit and limited to the benchmark-owned `pg_gobench` schema.
- [ ] Setup failures are returned and cause benchmark start to fail visibly.
- [ ] No SQL is executed through pgx direct APIs; use `database/sql`.
- [ ] `make check` — passes cleanly
- [ ] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [ ] `make lint` — passes cleanly
- [ ] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>
