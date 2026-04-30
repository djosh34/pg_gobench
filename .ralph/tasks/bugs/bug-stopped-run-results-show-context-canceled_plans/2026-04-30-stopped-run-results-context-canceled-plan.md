# Stopped Run Results Context-Canceled Plan

## Scope

Fix the bug where a normal user stop leaves `/benchmark/results` in `status: "stopped"` but still exposes `stats.latest_error: "context canceled"`.

This task should deliver:

- a stop-aware results snapshot that does not report control-plane cancellation as a workload error
- Red-Green coverage for the exact stopped-results behavior
- real manual verification against the documented Docker Compose stack after the code fix

This task should not deliver:

- a runner-wide error model rewrite
- changes to `/benchmark` lifecycle semantics
- long-lane or e2e coverage unless execution proves this bug changed those lanes

## Public Interface

Keep the public contract small:

- `POST /benchmark/stop` still returns a stopped lifecycle state
- `GET /benchmark/results` still returns the same `benchmarkrun.Results` shape
- only the final value of `stats.latest_error` changes for the specific case of a successful user stop

Expected observable behavior after the fix:

- when a run ends because the user requested stop, `results.status` is `stopped`
- `results.error` is empty
- `results.stats.latest_error` is empty if the only terminal error was the expected cancellation used to stop workers
- genuine workload failures still surface as failures rather than being erased

If RED shows that preserving real workload errors during a stopped run requires a richer terminal-signal shape than the current sample can express, switch this plan back to `TO BE VERIFIED` immediately instead of hiding all stopped-run errors unconditionally.

## Boundary Decision

The `improve-code-boundaries` move for this bug is to keep stop semantics in `internal/benchmarkrun`, not in `internal/benchrunner`.

Current smell:

- `benchrunner` records a raw latest error sample
- `benchmarkrun` already owns the distinction between `stopped` and `failed`
- `/benchmark/results` currently forwards the raw sample straight through, so control-plane cancellation leaks into product-facing results

Planned boundary cleanup:

- `internal/benchrunner` continues to own raw workload execution and raw sample collection
- `internal/benchmarkrun` owns stop-aware normalization when composing or retaining final results snapshots
- `internal/httpserver` remains a thin transport layer that serves whatever `benchmarkrun.Results` exposes

That keeps transport and runner concerns separate and avoids teaching the runner that some `context.Canceled` values are product-success signals.

## TDD Strategy

Use strict vertical slices. One failing test, the minimum implementation to pass it, then the next slice only if manual verification still reproduces the bug.

Planned first tracer bullet:

- [ ] Add one failing coordinator-level test proving that after a normal stop completes, `Results()` returns `StatusStopped` with an empty `Stats.LatestError` even if the underlying run snapshot carries compact `context canceled`

Planned green step:

- [ ] Add the smallest stop-aware normalization needed in `internal/benchmarkrun` so stopped results drop only the control-plane cancellation artifact

Manual verification loop after the first green:

- [ ] Reproduce the original flow against `examples/docker-compose-postgres/compose.yaml`
- [ ] Confirm `GET /benchmark/results` no longer shows `stats.latest_error: "context canceled"` after `POST /benchmark/stop`
- [ ] If the real stack still leaks a different stop-only error string or another path, add exactly one new failing test for that observed behavior before changing code again

## Implementation Notes

Start in `internal/benchmarkrun/coordinator_test.go` and keep the first RED at the public results boundary rather than the HTTP layer. The coordinator already owns:

- stop state transitions
- final sample retention after `Wait()`
- composition of lifecycle state plus sample into `Results`

Expected code shape:

- introduce one small helper in `internal/benchmarkrun` that normalizes a sample for results composition in the specific successful-stop case
- apply that helper where finished snapshots are retained or where `Results()` is composed, whichever keeps the stop rule local and easiest to reason about
- avoid widening `httpserver` or adding UI-specific conditionals

Avoid these muddy shortcuts:

- do not blank `latest_error` for every stopped result regardless of cause
- do not move control-plane stop knowledge into `internal/benchrunner`
- do not special-case JSON rendering in `httpserver`

## File Plan

Expected files:

- `internal/benchmarkrun/coordinator_test.go`
- `internal/benchmarkrun/coordinator.go`
- possibly `internal/benchmarkrun/results.go` if the normalization helper belongs with results composition

No new public packages or DTO layers should be needed for this bug.

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless execution proves this bug changed ultra-long selection or behavior

## Execution Rule

If execution shows that the only honest fix requires changing the sample/result type boundary materially, or that stopped runs can contain both expected stop cancellation and a preserved real workload error in the same snapshot, switch this plan back to `TO BE VERIFIED` immediately instead of forcing a muddy partial rule.

NOW EXECUTE
