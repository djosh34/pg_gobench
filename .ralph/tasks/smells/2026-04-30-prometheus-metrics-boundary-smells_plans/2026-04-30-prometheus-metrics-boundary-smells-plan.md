# Prometheus Metrics Boundary Smell Plan

## Scope

Clean up the remaining benchmark metrics boundary duplication left after the `/metrics` feature landed.

Current smell:

- `internal/httpserver` is thin, which is good.
- `internal/benchmarkrun` still exposes two parallel public snapshot shapes for the same run sample: JSON-oriented `Stats` and Prometheus-oriented `MetricsSnapshot`.
- `internal/benchrunner` must produce both shapes separately through `Snapshot()` and `Metrics()`.
- `internal/benchmarkrun/coordinator.go` must cache both shapes separately.
- `metricsFromResults` is dead code, which is a symptom of the abandoned JSON-to-metrics bridge.

That leaves the benchmark boundary wider than necessary. One run sample should be produced once, then rendered into the JSON results view and the Prometheus exposition view inside `internal/benchmarkrun`.

## Boundary Move

Use `improve-code-boundaries` aggressively here:

- introduce one canonical benchmark-owned snapshot type in `internal/benchmarkrun` that contains the shared measured data needed by both public views
- keep Prometheus naming and rendering in `internal/benchmarkrun`
- keep JSON response shaping in `internal/benchmarkrun`
- make `internal/benchrunner` produce only that canonical snapshot
- make `internal/httpserver` continue to consume only already-shaped `Results()` and already-renderable metrics output

Planned shape:

- `benchmarkrun.Run` exposes one sample method instead of separate `Snapshot()` and `Metrics()`
- `benchmarkrun.Coordinator` stores one completed sample instead of separate `stats` and `metrics` caches
- `benchmarkrun.Results()` derives `Stats` from the canonical sample
- `benchmarkrun.Metrics()` derives `MetricsSnapshot` from the same canonical sample, with `RunActive` still owned by coordinator state
- remove `metricsFromResults`

This keeps transport thin and also flattens duplicate view-building out of the runner boundary.

## Public Behavior To Preserve

- `GET /benchmark/results` remains the JSON API shape that existing tests expect
- `GET /metrics` remains Prometheus text with the current content type and `pg_gobench_` names
- histogram buckets, count, and sum remain exposed in seconds
- low-cardinality rules remain unchanged
- latest error text stays available in JSON results but never appears in Prometheus output

## TDD Slices

Follow strict vertical slices. Do not write all tests up front.

- [x] Slice 1: add a failing `benchmarkrun` or `benchrunner` test proving one canonical run sample can still produce both the JSON stats view and the Prometheus histogram view without losing current behavior
- [x] Slice 2: make the runner expose only the canonical sample and get that test green with the smallest code change
- [x] Slice 3: add a failing coordinator test proving finished-run caching still preserves both `/benchmark/results` data and `/metrics` data after the run ends
- [x] Slice 4: refactor coordinator storage from parallel `stats` plus `metrics` caches into one cached sample and get green
- [x] Slice 5: add or tighten an HTTP test proving `/metrics` output is unchanged after the refactor
- [x] Slice 6: remove dead bridging code such as `metricsFromResults`, then run the full check/lint/test gates

## Expected Files

- `internal/benchmarkrun/coordinator.go`
- `internal/benchmarkrun/coordinator_test.go`
- `internal/benchmarkrun/results.go`
- `internal/benchmarkrun/metrics.go`
- `internal/benchrunner/runtime.go`
- `internal/benchrunner/runtime_test.go`
- `internal/benchrunner/stats.go`
- `internal/benchrunner/stats_test.go`

Possible new file only if it deepens the module cleanly:

- `internal/benchmarkrun/sample.go`

## Risks To Watch

- do not let the canonical sample become JSON-tagged transport soup
- do not move Prometheus naming or histogram formatting into `httpserver`
- do not duplicate latency conversion logic in both runner and benchmarkrun
- if the refactor reveals that one canonical sample cannot preserve both JSON and Prometheus semantics cleanly, switch this plan back to `TO BE VERIFIED` immediately instead of forcing it

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long`

NOW EXECUTE
