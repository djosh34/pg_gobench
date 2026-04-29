## Task: 02 Implement Core Read Write And Transaction Workloads <status>not_started</status> <passes>false</passes>

<blocked_by>.ralph/tasks/story-03-core-benchmark/task-01-benchmark-schema-scale.md</blocked_by>

<description>
Must use tdd skill to complete

**Goal:** Implement the first executable PostgreSQL benchmark workloads through Go's `database/sql` interface. These workloads must be useful for evaluating common database behavior and must feed the coordinator and stats aggregator.

Include at minimum:

- point reads by primary key
- indexed/range reads
- insert-heavy writes
- update-heavy writes
- mixed read/write workload
- multi-statement transaction workload

The workload runner must honor benchmark options for clients, duration, warmup, profile, read/write mix where applicable, transaction mix where applicable, and optional target TPS/rate limiting. Worker errors must stop or fail the run according to coordinator rules; do not swallow SQL errors.
</description>

<acceptance_criteria>
- [ ] TDD red/green coverage exists for workload selection by profile.
- [ ] TDD red/green coverage exists for client count and duration behavior using deterministic clocks or controllable workers where practical.
- [ ] TDD red/green coverage exists for target TPS/rate limiting behavior.
- [ ] TDD red/green coverage or integration coverage exists for point read, range read, insert, update, mixed, and transaction workloads using `database/sql`.
- [ ] Workload errors are returned to the coordinator and appear in failed run state.
- [ ] No direct pgx query APIs are used for benchmark execution.
- [ ] `make check` — passes cleanly
- [ ] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [ ] `make lint` — passes cleanly
- [ ] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>
