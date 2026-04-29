## Task: 01 Aggregate Benchmark Stats In Memory <status>not_started</status> <passes>false</passes>

<blocked_by>.ralph/tasks/story-03-core-benchmark/task-02-core-read-write-transaction-workloads.md</blocked_by>

<description>
Must use tdd skill to complete

**Goal:** Implement the in-memory benchmark statistics pipeline. Every workload must report the same stats shape no matter which profile is running, so API consumers and the later Prometheus endpoint do not need per-workload special cases.

The JSON stats must include at minimum p95 latency, p99 latency, TPS, total operations, successful operations, failed operations, active clients, configured clients, elapsed seconds, operation rate by workload type where available, and compact latest error text when present. Include additional useful latency values such as p50, p90, min, max, and average if they can be implemented cleanly.

Stats are in memory only. Do not add persistence. Use a design that avoids unbounded memory growth for long runs, such as histograms, bounded samples, rolling aggregation, or another tested approach.
</description>

<acceptance_criteria>
- [ ] TDD red/green coverage exists for p95 and p99 latency calculation.
- [ ] TDD red/green coverage exists for TPS and elapsed-time calculation.
- [ ] TDD red/green coverage exists for success/failure operation counts and active/configured clients.
- [ ] TDD red/green coverage exists proving every workload/profile reports the same top-level stats shape.
- [ ] Stats aggregation does not grow memory without bound for long runs.
- [ ] Error text is compactly included in JSON state/results when present.
- [ ] `make check` — passes cleanly
- [ ] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [ ] `make lint` — passes cleanly
- [ ] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>
