# Core Read/Write/Transaction Workloads Plan

## Scope

Implement the first real `database/sql` workload runner behind the existing `benchmarkrun.Runner` boundary. This task should deliver:

- executable point-read and indexed/range-read SQL operations
- executable insert and update SQL operations
- a mixed read/write profile driven by `read_percent`
- a multi-statement transaction profile driven by `transaction_mix`
- client concurrency, timed run shutdown, and optional target TPS pacing
- visible worker/startup SQL failures returned through the existing coordinator state machine

This task should not deliver:

- a public stats/results API before story 04
- `join` or `lock` workload execution yet
- direct pgx query execution
- a second benchmark request/runtime DTO layer separate from `benchmark.StartOptions`

## Public Interface

Keep the exported surface intentionally small:

- `internal/benchmark` keeps the existing `StartOptions`, `AlterOptions`, `Profile`, and `TransactionMix`
- `internal/benchmarkrun` keeps the existing `Runner` and `Run` interfaces
- `internal/benchrunner` continues to export only:
  - `func New(db *sql.DB) benchmarkrun.Runner`

No new public profile enum is needed for point-vs-range or insert-vs-update selection. Instead:

- `profile == "read"` selects a read family that executes both point and range operations
- `profile == "write"` selects a write family that executes both insert and update operations
- `profile == "mixed"` uses `read_percent` to choose between the read and write families
- `profile == "transaction"` executes a multi-statement SQL transaction and uses `transaction_mix` to bias the statement mix inside that transaction
- `profile == "join"` and `profile == "lock"` must fail fast with a clear Go error string until story 06 implements them

If RED shows that callers truly need separate public profiles for point/range or insert/update instead of deterministic family composition under `read` and `write`, switch this plan back to `TO BE VERIFIED` immediately.

## Boundary Decision

The `improve-code-boundaries` move for this task is to remove the current mixed-responsibility shape where `internal/benchrunner` both prepares schema and pretends a live benchmark exists while the returned `run` only blocks on context cancellation.

Execution should flatten that boundary like this:

- `internal/benchmarkrun` remains only a lifecycle/state machine; it must not learn SQL, client loops, timers, or workload subtypes
- `internal/benchrunner` owns schema preparation, workload selection, worker lifecycle, pacing, and SQL execution end-to-end
- `internal/app` remains bootstrap only: open `*sql.DB`, build the concrete runner, build the coordinator
- keep workload-family selection on canonical `benchmark.StartOptions`; do not introduce `ReadWorkloadOptions`, `WriteRunnerConfig`, or another near-copy config layer

Inside `internal/benchrunner`, use private types to deepen the module instead of widening exported APIs. Likely private seams:

- one workload-plan selector that maps `benchmark.StartOptions` to executable operation families
- one worker/session type that owns goroutines, cancellation, and pacing
- one private operation observer/no-op sink so warmup and later stats plumbing can stay inside the runner without exposing a fake public results contract yet

## Workload Model

Planned workload behavior:

- point read: single-row account lookup by primary key
- range read: bounded indexed account scan by `(branch_id, id)` or equivalent existing benchmark index
- insert: append a history row or other benchmark-owned write that stays inside `pg_gobench`
- update: update balances on benchmark-owned rows
- mixed: choose read versus write by `read_percent`, then choose a concrete operation within that family deterministically
- transaction: use `database/sql` transaction APIs for a multi-statement flow across benchmark-owned tables, with `transaction_mix` controlling whether the transaction is balanced, read-heavy, or write-heavy

Keep SQL limited to the benchmark-owned schema and tables introduced in task 01. No public-schema writes, no search-path tricks, and no query execution outside `database/sql`.

For advanced profiles not yet implemented:

- `join` should return a visible unsupported-profile error
- `lock` should return a visible unsupported-profile error

That is better than silently mapping them to another workload family.

## Runtime Semantics

The runner must honor the benchmark options that already exist:

- `clients`: start that many concurrent workers
- `duration_seconds`: stop the run when the configured runtime window ends
- `warmup_seconds`: keep the warmup/measurement phase boundary inside the runner so later stats aggregation can distinguish it without changing the public coordinator contract
- `target_tps`: pace total operation issue rate instead of running fully unbounded
- `AlterOptions`: support runtime changes only for `clients` and `target_tps`, matching the existing coordinator/option model

Use deterministic/private time dependencies in tests instead of sleeping the real clock. The runner should accept time/pacing helpers internally so tests can prove:

- the run stops when the duration window expires
- changing clients affects active worker count
- target TPS pacing gates operation issue rate

Do not export a clock interface from the package just to satisfy tests; keep that seam private to `internal/benchrunner`.

## TDD Strategy

Follow vertical slices only. One failing behavior test, the minimum code to pass it, then the next slice.

Planned slices:

- [x] Slice 1: failing profile-selection test proving `read`, `write`, `mixed`, and `transaction` choose the expected workload families, while `join` and `lock` fail fast with explicit unsupported-profile errors
- [x] Slice 2: failing `database/sql` integration-style test for the `read` profile proving both point-read and range-read SQL paths execute against benchmark-owned tables
- [x] Slice 3: failing `database/sql` integration-style test for the `write` profile proving both insert and update SQL paths execute against benchmark-owned tables
- [x] Slice 4: failing mixed-profile test proving `read_percent` controls whether the runner chooses from the read family or write family
- [x] Slice 5: failing transaction-profile test proving the runner uses real `database/sql` transaction boundaries and executes the expected multi-statement flow
- [x] Slice 6: failing runtime-control test proving configured client count and duration drive worker startup/shutdown through deterministic time or controllable worker seams
- [x] Slice 7: failing pacing test proving `target_tps` limits operation issuance and altered TPS/client settings take effect while running
- [x] Slice 8: failing coordinator-facing test proving worker SQL errors surface as run failure with visible Go error text
- [x] Slice 9: refactor after green to keep workload selection, pacing, and worker/session orchestration private to `internal/benchrunner` without duplicate option/config structs

Tests should stay on public behavior boundaries:

- `internal/benchrunner` tests can use recording/fake `database/sql` drivers and private injected time/pacing seams
- `internal/benchmarkrun` tests should keep asserting failed-run state through the coordinator API
- avoid brittle tests that only compare raw SQL strings without exercising the actual runner path

## File Plan

Expected files:

- `internal/benchrunner/runner.go`
- `internal/benchrunner/runner_test.go`
- `internal/benchmarkrun/coordinator_test.go`

Likely new private files if they keep the module deep and readable:

- `internal/benchrunner/workload.go`
- `internal/benchrunner/workload_test.go`
- `internal/benchrunner/runtime.go`

Possible small touch points only if execution proves necessary:

- `internal/benchmark/options.go`
- `internal/benchmark/options_test.go`

Avoid changing `internal/httpserver` or inventing a public results type in this task. Story 04 should add the real stats pipeline on top of the runner behavior delivered here.

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless execution unexpectedly changes ultra-long test selection, which this task should not do

If execution shows that warmup cannot stay an internal runner concern, or that the current public profile model is insufficient to express the required workload families cleanly, switch this plan back to `TO BE VERIFIED` instead of forcing a muddy boundary.

NOW EXECUTE
