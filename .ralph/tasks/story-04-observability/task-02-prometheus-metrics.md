## Task: 02 Expose Prometheus Metrics Endpoint <status>not_started</status> <passes>false</passes>

<blocked_by>.ralph/tasks/story-04-observability/task-01-stats-aggregation.md</blocked_by>

<description>
Must use tdd skill to complete

**Goal:** Add `/metrics` with Prometheus text exposition for the same in-memory benchmark stats available through the JSON API. Every metric name must start with `pg_gobench_`.

Expose conservative, low-cardinality metrics. Do not use arbitrary SQL text, database names, hostnames, benchmark IDs, or raw error messages as Prometheus labels. Include metrics equivalent to run active state, run duration, configured clients, active clients, operations total, operation errors total, TPS, and operation latency histogram. The JSON API may expose direct p95/p99 values; Prometheus should provide histogram buckets so users can calculate quantiles with `histogram_quantile`.
</description>

<acceptance_criteria>
- [ ] TDD red/green coverage exists for `/metrics` returning valid Prometheus text.
- [ ] TDD red/green coverage exists proving every metric starts with `pg_gobench_`.
- [ ] TDD red/green coverage exists for run state, duration, clients, operation totals, errors, TPS, and latency histogram output.
- [ ] Metrics labels are low cardinality and do not include SQL text, database name, host, benchmark ID, or raw error message.
- [ ] `/metrics` is unauthenticated and does not expose secrets.
- [ ] `make check` — passes cleanly
- [ ] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [ ] `make lint` — passes cleanly
- [ ] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>
