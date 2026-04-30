# Prometheus Metrics Plan

## Scope

Add an unauthenticated `/metrics` endpoint that exposes the current in-memory benchmark state in Prometheus text format with a strict `pg_gobench_` prefix. This task should deliver:

- Prometheus text exposition with the correct content type
- low-cardinality metrics for run state, run duration, configured clients, active clients, total operations, operation errors, TPS, and latency histogram
- histogram buckets so Prometheus can calculate quantiles itself
- tests that prove naming, transport behavior, and label/cardinality constraints through public interfaces

This task should not deliver:

- arbitrary labels derived from SQL, database name, hostname, benchmark ID, or raw error text
- reuse of the JSON `/benchmark/results` payload as the Prometheus source of truth
- a third-party Prometheus registry or client package unless RED proves a local encoder is insufficient

## Public Interface

Keep the transport boundary small:

- add `GET /metrics` in `internal/httpserver`
- extend the benchmark control interface with a metrics snapshot method
- expose a typed metrics snapshot from `internal/benchmarkrun` rather than making `httpserver` inspect `Results` or `Stats`
- render Prometheus text from a typed metrics snapshot via a writer-style boundary, not ad hoc string soup in the handler

Planned exported metrics:

- `pg_gobench_run_active` gauge, `0` or `1`
- `pg_gobench_run_duration_seconds` gauge
- `pg_gobench_configured_clients` gauge
- `pg_gobench_active_clients` gauge
- `pg_gobench_operations_total` counter
- `pg_gobench_operation_errors_total` counter
- `pg_gobench_tps` gauge
- `pg_gobench_operation_latency_seconds` histogram with `_bucket`, `_count`, and `_sum`

Deliberate non-goals for the public metrics contract:

- no latest-error metric
- no profile label
- no operation-kind label
- no benchmark options echoed back into labels

That keeps the entire metrics surface low-cardinality by construction. The only label should be the Prometheus-required histogram `le` bucket label.

## Boundary Decision

The `improve-code-boundaries` move for this task is to flatten the mismatch between the current JSON-oriented stats boundary and the new Prometheus needs.

Current risky shape:

- `internal/benchrunner` owns the real latency histogram counts
- `internal/benchmarkrun` exposes only JSON-facing summary stats
- `internal/httpserver` is the only obvious place to add `/metrics`

If `/metrics` is implemented directly in `httpserver` by scraping `Results` or by inventing metric names beside the handler, the transport layer would own benchmark semantics, histogram policy, and low-cardinality rules. That is wrong-place knowledge.

Planned shape:

- `internal/benchrunner` remains the owner of latency bucket accounting and converts private runtime data into one typed metrics snapshot
- `internal/benchmarkrun` owns the public metrics contract and exposition rendering boundary
- `internal/httpserver` only sets the content type, status code, and writes the already-renderable metrics snapshot

If RED shows that the existing `Stats` type is too summary-oriented for this, introduce a deeper canonical runner snapshot that can feed both JSON results and Prometheus exposition cleanly. Do not widen `httpserver` instead.

## Metrics Model

Prometheus should reflect the same in-memory run, but not necessarily the same JSON shape.

Planned model:

- reuse the existing measured counters for operations, errors, TPS, active clients, configured clients, and elapsed time
- add a typed latency histogram snapshot that exposes bucket upper bounds, cumulative counts, total count, and sum in seconds
- derive `run_active` from coordinator state, not from transport heuristics
- keep `run_duration_seconds` based on measured runtime state, clamped to zero before measurement starts

Important translation rules:

- Prometheus histogram units should be seconds
- JSON quantiles stay available in `/benchmark/results`
- Prometheus must expose buckets, count, and sum instead of p95 or p99-only values
- raw error text must never cross the metrics boundary

## TDD Strategy

Follow strict vertical slices. One failing behavior test, minimum code to pass, then the next slice.

Planned slices:

- [ ] Slice 1: failing `httpserver` test proving `GET /metrics` returns Prometheus text with the Prometheus content type and no authentication requirement
- [ ] Slice 2: failing metrics-rendering test proving every emitted metric name starts with `pg_gobench_`
- [ ] Slice 3: failing runner or benchmarkrun metrics-snapshot test proving latency histogram buckets, count, and sum are exported from the in-memory histogram in seconds
- [ ] Slice 4: failing endpoint test proving the exposition includes run active state, duration, configured clients, active clients, operations total, operation errors total, and TPS
- [ ] Slice 5: failing endpoint or renderer test proving the exposition contains no forbidden labels or raw error text, and only histogram bucket labels are present
- [ ] Slice 6: refactor after green to keep metric naming, histogram rendering, and label policy out of `httpserver`

Tests must stay on behavior boundaries:

- HTTP tests assert content type, endpoint behavior, and exposition text properties
- benchmarkrun tests assert the public metrics contract and rendered names
- benchrunner tests assert histogram snapshot behavior, not private mutex fields or formatting details

## Implementation Notes

Expected internal changes:

- extend the control-plane interface used by `internal/httpserver` with a metrics snapshot method
- add a new benchmarkrun metrics type plus a writer or renderer for Prometheus text
- extend the runner snapshot path so Prometheus can access histogram bucket data without parsing the JSON summary struct
- keep `/benchmark/results` unchanged as the JSON API for humans and API consumers

Avoid these muddy shortcuts:

- do not render Prometheus text by marshaling JSON first
- do not add user-controlled labels
- do not expose SQL strings, database names, hostnames, benchmark IDs, or latest error text
- do not add a second histogram implementation just for Prometheus

## File Plan

Expected files:

- `internal/httpserver/server.go`
- `internal/httpserver/server_test.go`
- `internal/benchmarkrun/coordinator.go`
- `internal/benchmarkrun/coordinator_test.go`
- `internal/benchrunner/stats.go`
- `internal/benchrunner/stats_test.go`

Likely new files if they keep the boundary deep and readable:

- `internal/benchmarkrun/metrics.go`
- `internal/benchmarkrun/metrics_test.go`

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless execution unexpectedly changes the ultra-long lane or the task text is updated to require it

If execution shows that a typed metrics snapshot cannot be introduced cleanly without reworking the stats boundary, switch this plan back to `TO BE VERIFIED` immediately rather than forcing Prometheus concerns into `httpserver`.

NOW EXECUTE
