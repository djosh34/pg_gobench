## Task: 01 Add Join Lock And Contention Workloads <status>not_started</status> <passes>false</passes>

<blocked_by>.ralph/tasks/story-05-delivery/task-02-docker-compose-postgres-example.md</blocked_by>

<description>
Must use tdd skill to complete

**Goal:** Extend the benchmark engine with workloads that expose PostgreSQL behavior beyond simple reads and writes. The service must support join performance, aggregation/grouping behavior, explicit lock contention, and hot-row update contention.

Include at minimum:

- join workload across related benchmark tables
- aggregation/group-by workload
- lock contention workload using benchmark-owned tables
- hot-row update contention workload

These workloads must integrate with the same option model, run coordinator, stats pipeline, and error handling as the core workloads. If lock timeouts or serialization/conflict errors occur, record them as benchmark operation errors with clear Go error text. Do not swallow expected contention errors silently.
</description>

<acceptance_criteria>
- [ ] TDD red/green coverage exists for selecting `join` and `lock` profiles.
- [ ] TDD red/green coverage or integration coverage exists for join and aggregation SQL against benchmark-owned schema.
- [ ] TDD red/green coverage or integration coverage exists for lock contention and hot-row contention behavior.
- [ ] Contention-related SQL errors are counted and surfaced instead of ignored.
- [ ] These workloads report the same stats shape as every other workload.
- [ ] No direct pgx query APIs are used for benchmark execution.
- [ ] `make check` — passes cleanly
- [ ] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [ ] `make lint` — passes cleanly
- [ ] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>
