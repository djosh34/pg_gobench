# Stats Aggregation Plan

## Scope

Implement the in-memory benchmark statistics pipeline for the existing benchmark runner and expose it cleanly through the control-plane/results boundary. This task should deliver:

- bounded-memory latency aggregation with p50/p90/p95/p99/min/max/avg
- measured TPS, elapsed seconds, total/success/failed operation counts, and compact latest error text
- active and configured client counts
- a fixed JSON stats shape for every profile and workload family
- `/benchmark/results` returning benchmark results rather than mirroring `/benchmark`

This task should not deliver:

- persistence or historical run storage
- Prometheus export yet
- per-profile result DTOs
- unbounded raw latency sample retention

## Public Interface

Keep the exported surface intentionally small and explicit:

- `benchmarkrun.State` stays the lifecycle/status contract for `GET /benchmark`
- add `benchmarkrun.Results` for `GET /benchmark/results`
- add `benchmarkrun.Stats` as the canonical stats payload inside `Results`
- extend `benchmarkrun.Run` with a snapshot method so the coordinator can compose lifecycle state with runner-owned stats

Planned results shape:

- `Results` contains the current `State` fields plus a `stats` object
- `Stats` always contains the same top-level fields regardless of profile:
  - `latency`
  - `tps`
  - `total_operations`
  - `successful_operations`
  - `failed_operations`
  - `active_clients`
  - `configured_clients`
  - `elapsed_seconds`
  - `operation_rates`
  - `latest_error`
- `operation_rates` must also have a fixed shape. Use a stable object with all known concrete workload kinds always present:
  - `point_read`
  - `range_read`
  - `history_insert`
  - `account_update`
  - `transaction`

Profiles that do not emit a given kind report zero for that rate. That avoids downstream special-casing.

If RED shows that `Results` must diverge materially from `State + stats`, switch this plan back to `TO BE VERIFIED` immediately instead of forcing another muddy transport layer.

## Boundary Decision

The `improve-code-boundaries` move for this task is to remove the current split where:

- `benchmarkrun` knows only lifecycle state
- `benchrunner` performs real work
- `/benchmark/results` just reuses lifecycle state because no stats boundary exists

Flatten that boundary like this:

- `internal/benchmarkrun` owns result composition and exposes the public `Results` contract
- `internal/benchrunner` owns stats collection, latency bucketing, workload-kind classification, warmup/measurement cutoff, and compact error capture
- `internal/httpserver` remains a thin transport layer that simply serves `State` on `/benchmark` and `Results` on `/benchmark/results`

Do not introduce a second config layer or stringly workload labels outside `internal/benchrunner`.

Inside `internal/benchrunner`, deepen the module with private types instead of widening external APIs:

- a private `operationKind` enum replaces inferred workload labels
- a private stats collector owns bounded latency distribution and counters
- `workloadPlan` returns the executed operation kind directly so the runtime can observe behavior without parsing SQL or knowing profile internals

That removes the wrong-module smell where higher layers would otherwise need workload-specific knowledge.

## Stats Model

Use constant-space aggregation:

- maintain a fixed latency histogram with deterministic buckets, plus count/sum/min/max
- compute quantiles from histogram counts rather than retaining raw durations
- keep counters for total/success/failed operations and per-operation-kind counts
- keep a single compact latest error string only

Planned measurement semantics:

- warmup remains internal to `benchrunner`
- operations completed before the warmup boundary do not contribute to measured TPS/latency/counts
- `elapsed_seconds` reflects measured time after warmup and is clamped to zero before measurement starts
- active clients reflect live worker count
- configured clients reflect the latest requested client count, including `Alter`

Compact error handling:

- keep only the latest error text
- normalize whitespace and truncate to a fixed small size so error storage stays bounded

## TDD Strategy

Follow strict vertical slices. One failing behavior test, the minimum code to pass it, then the next slice.

Planned slices:

- [x] Slice 1: failing `benchmarkrun` or `httpserver` test proving `/benchmark/results` no longer mirrors plain `State` and instead exposes a stable `Results` shape with a `stats` object
- [x] Slice 2: failing `benchrunner` stats test proving bounded histogram quantiles compute p95 and p99 from recorded durations without storing raw samples
- [x] Slice 3: failing `benchrunner` runtime test proving measured TPS and `elapsed_seconds` respect the warmup cutoff and run clock
- [x] Slice 4: failing `benchrunner` runtime test proving success/failure counts and active/configured client counts update through start, alter, and worker shutdown
- [x] Slice 5: failing `benchrunner` or `benchmarkrun` test proving every profile produces the same top-level stats shape and fixed `operation_rates` keys
- [x] Slice 6: failing error-path test proving latest error text appears compactly in results/state when a worker operation fails
- [x] Slice 7: refactor after green to keep operation classification and stats collection entirely private to `internal/benchrunner`

Tests must stay on behavior boundaries:

- stats math tests can target the private collector directly because it is the module boundary for bounded aggregation
- coordinator tests should assert public `Results` behavior, not internal collector fields
- HTTP tests should assert JSON shape and compact error exposure, not struct implementation details

## Implementation Notes

Expected internal changes:

- change private `workloadPlan` from `RunOnce(...) error` to `RunOnce(...) (operationKind, error)` or an equivalent private result type
- record operation start/end around each `RunOnce` call inside `activeRun`
- update worker lifecycle code to feed active-client counts into the collector
- capture a final snapshot when a run stops or fails so `/benchmark/results` still reports the last finished run

Avoid these muddy shortcuts:

- do not compute workload type from SQL strings
- do not add profile-specific optional JSON fields
- do not expose histogram internals through public packages
- do not store every latency sample or every error

## File Plan

Expected files:

- `internal/benchmarkrun/coordinator.go`
- `internal/benchmarkrun/coordinator_test.go`
- `internal/httpserver/server.go`
- `internal/httpserver/server_test.go`
- `internal/benchrunner/runtime.go`
- `internal/benchrunner/runtime_test.go`
- `internal/benchrunner/workload.go`

Likely new files if they keep the module deep and readable:

- `internal/benchmarkrun/results.go`
- `internal/benchrunner/stats.go`
- `internal/benchrunner/stats_test.go`

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless execution unexpectedly changes the ultra-long lane or task selection, which this task should not do

If execution shows that the public results contract needs more than one stats view, or that warmup cannot remain an internal runner concern, switch this plan back to `TO BE VERIFIED` immediately instead of widening the wrong boundary.

NOW EXECUTE
