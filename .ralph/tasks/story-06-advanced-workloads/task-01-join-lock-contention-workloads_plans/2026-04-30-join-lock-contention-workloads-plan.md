# Join/Lock/Contention Workloads Plan

## Scope

Implement the advanced `database/sql` workload families behind the existing benchmark runner. This task should deliver:

- a `join` profile that executes both a multi-table join workload and an aggregation/group-by workload against `pg_gobench` tables
- a `lock` profile that executes both explicit lock-contention SQL and hot-row update contention SQL against `pg_gobench` tables
- visible contention failures recorded through the existing run/error/stats pipeline with clear Go error text
- the same top-level JSON stats shape and fixed `operation_rates` object for old and new profiles

This task should not deliver:

- new public profiles beyond `join` and `lock`
- direct pgx query execution
- a second benchmark options/config model
- `make test-long` unless execution proves this task changed the ultra-long lane

## Public Interface

Keep the exported surface intentionally small:

- `internal/benchmark` keeps the existing `StartOptions`, `AlterOptions`, and public `Profile` enum
- `internal/benchmarkrun` keeps the existing coordinator/results APIs
- `internal/benchrunner` continues to export only `func New(db *sql.DB) benchmarkrun.Runner`

Do not add public `aggregation` or `hot-row` profiles. Keep those as private operation families under the existing public profiles:

- `profile == "join"` selects a join-family workload that alternates between:
  - a relational join across benchmark-owned tables
  - an aggregation/group-by query across benchmark-owned tables
- `profile == "lock"` selects a contention-family workload that alternates between:
  - an explicit lock-contention operation
  - a hot-row update contention operation

If RED shows that aggregation or hot-row contention truly requires new public profile values instead of private composition under `join` and `lock`, switch this plan back to `TO BE VERIFIED` immediately.

## Boundary Decision

The `improve-code-boundaries` move for this task is to flatten the growing workload boundary inside `internal/benchrunner` before adding more behavior.

Right now `internal/benchrunner/workload.go` mixes:

- public-profile selection
- concrete SQL for unrelated workload families
- helper functions for account/branch/teller targeting
- two execution shapes (`*sql.DB` and `*sql.Tx`) that know nearly the same query/exec boundary

Execution should simplify that shape like this:

- keep profile selection in one small private selector
- split concrete workload families into focused private files rather than extending one monolith
- introduce one private SQL executor boundary that both `*sql.DB` and `*sql.Tx` satisfy for `QueryRowContext`, `QueryContext`, and `ExecContext`
- keep `operationKind` private to `internal/benchrunner`, but extend the stable `benchmarkrun.OperationCounts` and `benchmarkrun.OperationRates` shapes so all profiles still report the same keys

That removes two smells at once:

- `smell-8-too-much-in-one-file`: do not keep growing one workload file
- `smell-5-shared-connection-shape`: stop duplicating near-identical DB-vs-TX execution code paths

## Workload Model

Planned private operation kinds:

- `join`
- `aggregation`
- `lock_contention`
- `hot_update`

The existing operation kinds stay in place for read/write/transaction families so the stats shape remains additive and fixed.

Planned SQL direction:

- join operation:
  - join `accounts`, `branches`, and `tellers` through benchmark-owned foreign-key relationships
  - keep the result bounded by branch/account filters so it is a realistic repeated benchmark operation instead of a full-table blast
- aggregation operation:
  - aggregate over benchmark-owned rows with `GROUP BY`, most likely on `accounts.branch_id` or another always-populated benchmark relation
  - avoid depending on `history` being pre-populated because schema setup starts with zero history rows
- explicit lock-contention operation:
  - use a transaction-scoped locking statement on benchmark-owned tables/rows
  - prefer a fast-failing path such as `NOWAIT` or a short local lock timeout so contention becomes a visible operation error instead of an indefinite stall
- hot-row update operation:
  - direct all workers at the same small hot-row set in `pg_gobench.accounts`
  - use transaction-scoped timeout/conflict semantics so lock waits, serialization conflicts, or deadlocks surface as returned errors rather than hidden blocking

Keep all SQL inside the benchmark-owned schema. No public-schema writes, no direct pgx APIs, and no swallowed contention failures.

If RED shows that the lock family needs a different contention primitive to produce deterministic, benchmark-owned conflicts cleanly, switch this plan back to `TO BE VERIFIED` rather than forcing muddy SQL.

## Stats Contract

Story 04 established that every workload must report the same top-level stats shape. This task should preserve that by extending the fixed per-operation shape rather than adding profile-specific blobs.

Planned stable `operation_rates` / `operation_counts` keys after execution:

- `point_read`
- `range_read`
- `history_insert`
- `account_update`
- `transaction`
- `join`
- `aggregation`
- `lock_contention`
- `hot_update`

Profiles that do not emit a given kind continue to report zero for that kind.

This is preferable to:

- profile-specific stats payloads
- stringly operation labels built outside `internal/benchrunner`
- collapsing new workload families into existing keys and losing visibility

## Error Semantics

Keep the existing fail-fast runner behavior unless RED proves it is wrong. That means:

- the runtime records the failed operation in stats
- `latest_error` keeps the compact Go error text
- the run still finishes with an error after the operation failure is recorded

For contention paths specifically:

- do not special-case lock timeouts, deadlocks, serialization failures, or `NOWAIT` conflicts into success
- return a wrapped Go error with workload context such as `lock contention: ...` or `hot update: ...`
- let the existing runtime/stats path count the failure and stop the run

If RED shows that the task requires continue-on-error semantics instead of the current fail-fast runner contract, switch this plan back to `TO BE VERIFIED` immediately because that is a larger behavioral change than this story currently declares.

## TDD Strategy

Follow vertical slices only. One failing behavior test, the minimum code to pass it, then the next slice.

Planned slices:

- [ ] Slice 1: failing profile-selection test proving `join` and `lock` no longer return unsupported-profile errors and still select only those two public advanced profiles
- [ ] Slice 2: failing `database/sql` integration-style test proving the `join` profile executes both join SQL and aggregation/group-by SQL against benchmark-owned tables
- [ ] Slice 3: failing `database/sql` integration-style test proving the `lock` profile executes both explicit lock-contention SQL and hot-row update SQL through the runner
- [ ] Slice 4: failing stats-shape test proving `Results` and `Sample.Stats()` expose the extended fixed `operation_rates` keys across old and new profiles
- [ ] Slice 5: failing runtime/error test proving contention-related SQL errors are counted as failed operations and surfaced through compact Go error text instead of being ignored
- [ ] Slice 6: refactor after green to split workload families into smaller files and use one shared private SQL executor boundary

Tests must stay on behavior boundaries:

- workload tests should drive the real runner through the existing `database/sql` fake driver harness
- stats/results tests should assert stable JSON keys and rates, not internal struct layout
- error-path tests should assert counted failures and visible error text, not specific internal helper calls

## File Plan

Expected files:

- `internal/benchrunner/runner_test.go`
- `internal/benchrunner/runtime_test.go`
- `internal/benchrunner/stats.go`
- `internal/benchmarkrun/sample.go`
- `internal/benchmarkrun/sample_test.go`
- `internal/benchmarkrun/results.go`
- `internal/benchmarkrun/coordinator_test.go`

Likely workload-module reshaping during execution:

- `internal/benchrunner/workload.go` becomes the small private selector/common entry point
- add focused private files such as:
  - `internal/benchrunner/workload_common.go`
  - `internal/benchrunner/workload_join.go`
  - `internal/benchrunner/workload_lock.go`
  - optionally `internal/benchrunner/workload_read_write.go` if that split keeps the module flatter

Possible small touch points only if RED proves necessary:

- `internal/benchmark/options_test.go`

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless execution unexpectedly changes ultra-long tests or this turns into a story-finish validation

If execution shows that the fixed stats-key model cannot absorb the new workload kinds cleanly, or that the lock family needs new public options to be correct, switch this plan back to `TO BE VERIFIED` instead of forcing a muddy boundary.

NOW EXECUTE
