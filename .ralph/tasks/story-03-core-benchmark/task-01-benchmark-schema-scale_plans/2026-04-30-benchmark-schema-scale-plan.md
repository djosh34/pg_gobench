# Benchmark Schema Scale Plan

## Scope

Implement the benchmark-owned PostgreSQL schema setup path behind the benchmark runner boundary, with all objects created under the dedicated `pg_gobench` schema and no destructive operations outside that schema.

This task should deliver:

- a documented scale-to-row-count mapping for benchmark data sizes
- explicit schema-qualified DDL and deterministic seed data generation
- optional schema reset controlled by benchmark start options
- setup execution through `database/sql` only
- visible start failure when setup cannot complete

This task should not deliver:

- actual benchmark query workloads beyond setup/bootstrap
- direct pgx query execution
- compatibility with old schema names or public-schema objects
- a second, duplicate setup DTO layer separate from the benchmark option contract

This turn is planning-only. Execution belongs to the next turn once the plan is promoted to `NOW EXECUTE`.

## Public Interface

Keep benchmark option semantics in `internal/benchmark`, keep lifecycle in `internal/benchmarkrun`, and introduce one concrete SQL benchmark runner package that owns schema setup.

Planned interface changes:

- extend `benchmark.StartOptions` with an explicit reset flag:
  - `Reset bool` with JSON field `reset`
- add scale resolution to `internal/benchmark`:
  - `type ScaleModel struct`
  - `func ResolveScale(scale int) ScaleModel`
- add one concrete runner package, likely `internal/benchrunner`:
  - `func New(db *sql.DB) benchmarkrun.Runner`

Planned `ScaleModel` fields:

- `Branches int`
- `Tellers int`
- `Accounts int`
- `HistoryRows int`

The runner package should consume `benchmark.StartOptions` directly and call `benchmark.ResolveScale(options.Scale)`. Do not introduce `database.SetupOptions`, `schema.Options`, or another near-copy of the benchmark contract just to shuttle the same fields around.

## Boundary Decision

The `improve-code-boundaries` move for this task is: remove the current bootstrap leak where `internal/app` owns the database handle but wires a nil benchmark runner, and replace it with one concrete runner boundary that owns all benchmark SQL setup behavior.

Concrete cleanup to do during execution:

- keep scale semantics in `internal/benchmark` because `scale` is part of the public option contract
- keep the coordinator unchanged as a lifecycle/state machine that depends only on `benchmarkrun.Runner`
- move all schema SQL rendering, reset logic, and deterministic seed generation behind the concrete runner package
- keep `internal/app` as bootstrap only: open `*sql.DB`, construct the concrete runner, construct the coordinator
- avoid a split where one package renders SQL strings and another package reinterprets them into yet another command shape

This should flatten the boundary instead of spreading benchmark-owned SQL concerns across `app`, `database`, and `httpserver`.

## Schema Design

All benchmark-owned objects must live under `pg_gobench`, with explicit schema-qualified names. Do not rely on `search_path` mutation.

Planned objects:

- schema `pg_gobench`
- table `pg_gobench.accounts`
- table `pg_gobench.branches`
- table `pg_gobench.tellers`
- table `pg_gobench.history`
- indexes needed for primary-key lookups and later range-read / transaction workloads

Planned data rules:

- deterministic row identifiers and seed values derived from the chosen scale
- concrete row-count mapping documented in code comments next to `ResolveScale`
- idempotent setup when `reset=false`: create missing schema objects and seed only when the benchmark-owned tables are empty
- destructive setup only when `reset=true`: drop `pg_gobench` schema with cascade, then recreate it

If execution shows that later workload coverage requires a different minimal table set than `accounts/branches/tellers/history`, switch this plan back to `TO BE VERIFIED` instead of forcing an awkward intermediate schema.

## TDD Strategy

Use vertical slices only. One failing behavior test, then the smallest implementation to pass, then the next slice.

Planned slices:

- [x] Slice 1: failing `internal/benchmark` test for `ResolveScale` mapping representative scales to concrete row counts and documenting the intended size model
- [x] Slice 2: failing `internal/benchmark` JSON test for the new `reset` start option so the public contract stays explicit and transport-compatible
- [x] Slice 3: failing schema-render test proving generated SQL targets only `pg_gobench` object names and includes no unqualified destructive statement outside that schema
- [x] Slice 4: failing `database/sql` execution test using a recording SQL driver or equivalent public-interface harness to prove setup executes schema/table/index creation through `database/sql`
- [x] Slice 5: failing execution test proving `reset=false` never drops the schema and `reset=true` drops only `pg_gobench`
- [x] Slice 6: failing runner/coordinator test proving setup failure is returned from `Runner.Start`, drives the coordinator to failed state, and leaves the raw Go error text visible
- [x] Slice 7: refactor after green to keep SQL generation and setup execution inside the concrete runner package without duplicate option or schema DTOs

The `database/sql` setup test should stay on the public `database/sql` boundary. Avoid direct pgx use and avoid brittle tests that only assert random SQL substrings with no execution path.

## File Plan

Expected files:

- `internal/benchmark/options.go`
- `internal/benchmark/options_test.go`
- `internal/benchmark/scale.go`
- `internal/benchmark/scale_test.go`
- `internal/benchmarkrun/coordinator_test.go`
- `internal/app/app.go`
- one new concrete runner package, likely:
  - `internal/benchrunner/runner.go`
  - `internal/benchrunner/runner_test.go`

Possible cleanup during execution if it improves boundaries:

- remove any nil-runner bootstrap path once the concrete runner exists
- keep schema constants private to the runner package unless another package truly needs them
- collapse helper types if they only mirror `benchmark.StartOptions` or `ScaleModel`

## Quality Gates

- [x] `make check`
- [x] `make lint`
- [x] `make test`
- [ ] no `make test-long` unless this task ends up changing long-test selection, which it should not

If execution reveals that the concrete runner cannot stay honest without also delivering real workload loops from task-02 in the same patch, switch this plan back to `TO BE VERIFIED` immediately instead of shipping a muddy fake-running implementation.

NOW EXECUTE
