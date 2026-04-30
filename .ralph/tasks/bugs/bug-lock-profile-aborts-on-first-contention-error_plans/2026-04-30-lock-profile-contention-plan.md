# Lock Profile Contention Plan

## Scope

Fix the lock-profile bug where an expected PostgreSQL contention error aborts the entire benchmark immediately instead of being counted as a failed operation while the run continues.

This task should deliver:

- a `lock` workload that records lock-contention failures in stats without flipping the run to `failed` on the first expected contention error
- a realistic Red-Green test for the actual lock-conflict path (`FOR UPDATE NOWAIT` / PostgreSQL contention), not a weaker synthetic failure on setup SQL
- product-facing results that keep surfacing compact contention error text while the run remains alive until duration or explicit stop
- cleanup of stale tests and live task artifacts that still encode the wrong fail-fast contention contract

This task should not deliver:

- continue-on-error semantics for every workload failure in the system
- a public API or option change
- a benchmarkrun/httpserver lifecycle rewrite
- `make test-long` unless execution proves this bug affects the ultra-long lane

## Public Interface

Keep the public contract small and unchanged:

- `POST /benchmark/start` still accepts the existing `lock` profile
- `GET /benchmark/results` keeps the existing JSON shape
- `benchmarkrun.Status` values stay the same

Expected observable behavior after execution:

- when the `lock` profile hits row-lock contention, `stats.failed_operations` increases and `stats.latest_error` contains compact lock-related error text
- the run does not immediately become `status: "failed"` just because one contention operation failed
- the run remains `running` until its duration ends or the user stops it, and then settles as `stopped` unless a genuine terminal runner/setup error occurs
- `GET /benchmark/results` can show non-zero failed lock-related operations rather than an all-zero failed snapshot

If RED or manual verification shows that the correct contract is broader than lock-profile contention alone, switch this plan back to `TO BE VERIFIED` immediately instead of smuggling in a wider runner policy change without re-planning.

## Boundary Decision

The `improve-code-boundaries` move for this bug is to remove the overloaded meaning of a workload `error` inside `internal/benchrunner`.

Current smell:

- `workloadPlan.RunOnce` returns only `operationKind, error`
- `internal/benchrunner/runtime.go` records every operation error in stats and then immediately treats that same error as a terminal run failure
- `internal/benchrunner/runner_test.go` cannot model the real `FOR UPDATE NOWAIT` failure path cleanly, so it currently fakes contention by failing `SET LOCAL lock_timeout`

Planned cleanup:

- keep contention classification private to `internal/benchrunner`
- introduce one small private boundary that distinguishes counted non-terminal contention failures from terminal run failures
- keep `benchmarkrun` and `httpserver` unaware of SQLSTATE-specific contention rules
- improve the fake `database/sql` driver so tests can express query-path lock conflicts directly instead of abusing unrelated setup SQL

That keeps run-control semantics deep in the runner while letting lock-family workloads declare which failures are expected operation-level contention events.

## TDD Strategy

Follow strict vertical slices. One failing behavior test, the minimum implementation to pass it, then the next slice only if the real bug still reproduces.

Planned slices:

- [x] Slice 1: add one failing benchrunner/coordinator integration-style test that drives the real `lock` profile through the fake `database/sql` driver, injects a realistic row-lock conflict on the `FOR UPDATE NOWAIT` path, and proves the run records a failed lock operation plus latest error text without transitioning to `StatusFailed`
- [x] Slice 2: make that test green with the smallest private benchrunner change that treats classified lock contention as a non-terminal operation failure while keeping other workload/setup errors terminal
- [x] Slice 3: if manual verification still reproduces the bug through the hot-update half of the lock profile or another contention variant, add exactly one new failing test for that observed error class and make it green before changing more code
- [x] Slice 4: remove or rewrite stale tests and live task-plan text that still assert contention must stop the whole run
- [x] Slice 5: manually rerun the documented Docker Compose reproduction and confirm `/benchmark/results` now shows non-zero failed lock operations without immediate benchmark failure

The first RED should stay at the behavioral boundary:

- use the real `lock` profile selection and real runner/coordinator flow
- assert on state/results behavior, not helper names
- reproduce the real contention surface by failing the lock query itself, not `SET LOCAL lock_timeout`

## Implementation Notes

Start by improving the test harness only as much as needed to express the real bug:

- extend the fake SQL driver so query execution can fail on the `FOR UPDATE NOWAIT` path, either by allowing `QueryContext` to return an error or by returning rows that fail during scan
- stop using `SET LOCAL lock_timeout` as the proxy for lock contention in the red test

Then make the runtime change in `internal/benchrunner`:

- keep stats recording exactly once per operation
- preserve compact error text in `latest_error`
- continue the run only for the classified lock-contention failures that belong to the `lock` profile contract
- keep setup failures, pacing failures, begin/commit failures, and unrelated workload errors terminal

Possible private code shape:

- a small private error wrapper or outcome type that marks an operation failure as non-terminal
- lock-workload helpers that wrap expected contention errors with that private type
- runtime logic that records the failed operation and only calls `finish(err)` for terminal errors

Avoid these muddy shortcuts:

- do not hide contention errors from stats
- do not turn every workload error into continue-on-error behavior
- do not push SQLSTATE or contention classification into `benchmarkrun` or `httpserver`
- do not keep stale tests/docs that claim fail-fast is the intended lock-profile contract

## File Plan

Expected files:

- `internal/benchrunner/runner_test.go`
- `internal/benchrunner/runtime.go`
- `internal/benchrunner/workload_lock.go`
- `internal/benchrunner/workload_sql.go`
- possibly `internal/benchrunner/runtime_test.go` if a smaller runner-level behavior test makes the classification easier to cover
- `.ralph/tasks/story-06-advanced-workloads/task-01-join-lock-contention-workloads_plans/2026-04-30-join-lock-contention-workloads-plan.md` if it still states the stale fail-fast contention contract

No new exported packages, no new public DTOs, and no compatibility layer should be needed.

## Manual Verification

After the green code path is in place:

1. `docker compose -f examples/docker-compose-postgres/compose.yaml -p pg-gobench-manual up -d --build`
2. `curl -X POST http://127.0.0.1:8080/benchmark/start -H 'Content-Type: application/json' -d '{"scale":1,"clients":32,"duration_seconds":10,"warmup_seconds":1,"reset":false,"profile":"lock"}'`
3. `curl http://127.0.0.1:8080/benchmark/results`

Expected result:

- the run does not flip to `status: "failed"` immediately after the first contention
- `stats.failed_operations` becomes non-zero
- `stats.latest_error` still surfaces compact lock contention text

If manual verification shows a different contention class or failure path than the first test captured, write one new failing test for that exact observed behavior before changing code again.

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless execution proves this bug changed the ultra-long lane

## Execution Rule

If execution shows that the only honest fix requires a generalized runner outcome model across every workload, or that PostgreSQL contention cannot be classified narrowly enough inside the private lock workload boundary, switch this plan back to `TO BE VERIFIED` immediately instead of forcing a muddy partial policy.

NOW EXECUTE
