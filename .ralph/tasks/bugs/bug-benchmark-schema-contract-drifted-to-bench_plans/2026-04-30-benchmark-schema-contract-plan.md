# Benchmark Schema Contract Plan

Plan file: `.ralph/tasks/bugs/bug-benchmark-schema-contract-drifted-to-bench_plans/2026-04-30-benchmark-schema-contract-plan.md`

## Scope

Fix the shipped contract drift where the real runtime creates benchmark-owned tables under schema `bench`, but current contract artifacts still describe the live product as using schema `pg_gobench`.

This task should deliver:

- one canonical benchmark schema contract: benchmark-owned objects live under `bench`
- Red-Green coverage that protects the textual contract artifacts that define and verify this behavior
- manual verification against the real Docker Compose example proving PostgreSQL tables are created under `bench`
- cleanup of stale non-historical schema literals that still imply `pg_gobench` is the live benchmark schema

This task should not deliver:

- a configurable benchmark schema option
- a broad rename of the module, binary, image names, metric prefix, or PostgreSQL database name away from `pg_gobench`
- compatibility aliases that create both `bench` and `pg_gobench`
- `make test-long` unless execution proves this bug reaches the ultra-long lane

This turn is planning-only. Execution belongs to the next turn once this plan is promoted to `NOW EXECUTE`.

## Public Interface

Keep the public contract small and explicit:

- benchmark-owned tables live under schema `bench`
- reset behavior drops only schema `bench`
- manual verification and story artifacts refer to `bench` when they describe benchmark-owned tables
- `pg_gobench` remains valid where it refers to the application identity rather than the benchmark schema:
  - Go module path
  - binary and image name
  - metric prefix
  - example database name

Expected observable behavior after execution:

- the Compose reproduction query against `pg_tables` returns `bench.accounts`, `bench.branches`, `bench.history`, and `bench.tellers`
- no current contract artifact tells engineers or future manual verification to expect `pg_gobench.*` benchmark tables
- runtime behavior stays unchanged unless manual verification exposes a real remaining schema drift

If RED or manual verification shows that the benchmark schema must become configurable or exposed outside the runner-owned contract to stay honest, switch this plan back to `TO BE VERIFIED` immediately instead of widening the wrong boundary.

## Boundary Decision

The `improve-code-boundaries` move for this bug is to remove the split source of truth for the benchmark schema contract.

Current smell:

- `internal/benchrunner` already owns the real schema name through a private runtime constant
- story artifacts and manual-verification files became a second, stale contract owner
- a few non-runtime literals still mention `pg_gobench.*`, which makes drift harder to detect

Planned cleanup:

- keep `internal/benchrunner` as the only code owner of the canonical benchmark schema name
- do not export a schema constant just so other packages or docs can depend on it
- update only the contract artifacts that define the live product expectation to match the runner-owned contract
- clean up stale non-historical literals in tests where they imply the live schema is still `pg_gobench`
- leave historical bug reports alone when they are intentionally documenting the old failure state rather than current product behavior

That keeps the benchmark schema contract deep in one runtime module while eliminating stale text that pretends another live contract exists.

## TDD Strategy

This bug is unusual: the shipped runtime is already behaving correctly, and the broken surface is mainly textual contract drift. Because those contract files are the product specification and manual-verification source, the first RED must target that contract-artifact boundary directly rather than inventing a fake runtime failure.

Use strict vertical slices:

- [x] Slice 1: add one failing Go test that reads only the canonical live-contract artifacts and fails if they still describe the benchmark schema as `pg_gobench` or use `pg_gobench.*` table qualifiers for the current product contract
- [x] Slice 2: make that test green by updating the stale contract artifacts to `bench`, while preserving legitimate non-schema uses of `pg_gobench`
- [x] Slice 3: clean up stale supporting literals that are not historical records, such as fake setup-error text in tests, if they still imply the live schema contract is `pg_gobench`
- [x] Slice 4: manually rerun the documented Docker Compose reproduction and confirm PostgreSQL tables appear only under `bench`
- [x] Slice 5: manual verification exposed no remaining runtime drift, so no extra runner- or integration-level red slice was required before moving to the repo gates

The contract test must stay tight:

- include only files that define the current live schema expectation
- exclude historical bug reports that intentionally preserve old failure evidence
- reject only schema-contract drift, not unrelated `pg_gobench` identifiers like module paths, binary names, metrics, or database names

## Implementation Notes

Start with a small repo-level Go test because `make test` already runs `go test ./...` and the bug lives in repo artifacts that are not otherwise executable.

Expected test shape:

- read a short, explicit allowlist of contract files
- assert that any benchmark-schema expectation in those files refers to `bench`
- assert that `pg_gobench.*` table qualifiers do not appear in those live-contract files
- keep the matcher narrow enough that valid app-identity uses of `pg_gobench` do not fail the test

Then update the live contract sources, likely including:

- `.ralph/tasks/story-03-core-benchmark/task-01-benchmark-schema-scale.md`
- `.ralph/tasks/story-03-core-benchmark/task-01-benchmark-schema-scale_plans/2026-04-30-benchmark-schema-scale-plan.md`
- `.ralph/tasks/story-06-advanced-workloads/task-01-join-lock-contention-workloads_plans/2026-04-30-join-lock-contention-workloads-plan.md`
- `.ralph/tasks/story-99-manual-verify-everything/task-01-manual-verify-everything.md`

Small supporting cleanup is expected in:

- `internal/benchmarkrun/coordinator_test.go`

Only touch `internal/benchrunner/runner.go` or `internal/benchrunner/runner_test.go` if manual verification reveals an actual runtime mismatch beyond artifact drift.

Avoid these muddy shortcuts:

- do not rename every `pg_gobench` string in the repository
- do not rewrite historical bug tasks to pretend the old bug never happened
- do not export runtime-only schema internals just to feed markdown or tests
- do not add a second schema alias for backwards compatibility

## Manual Verification

Use the real documented stack after the green contract cleanup:

1. `docker compose -f examples/docker-compose-postgres/compose.yaml -p pg-gobench-quickstart up -d --build`
2. `curl --fail -X POST http://127.0.0.1:8080/benchmark/start -H 'Content-Type: application/json' -d '{"scale":1,"clients":2,"duration_seconds":20,"warmup_seconds":1,"reset":true,"profile":"mixed","read_percent":80}'`
3. `docker exec pg-gobench-quickstart-postgres-1 psql -U benchmark_user -d pg_gobench -At -c "SELECT schemaname || '.' || tablename FROM pg_tables ORDER BY 1;"`

Expected result:

- `bench.accounts`
- `bench.branches`
- `bench.history`
- `bench.tellers`

If the real stack still exposes any `pg_gobench.*` benchmark tables or another conflicting schema contract, capture that exact behavior with one new failing test before changing code again.

## Quality Gates

- [x] `make check`
- [x] `make lint`
- [x] `make test`
- no `make test-long` unless execution proves this bug changed the ultra-long lane

## Execution Rule

If execution shows that the chosen contract test is too broad, or that the only honest fix requires a broader application-identity rename rather than benchmark-schema reconciliation, switch this plan back to `TO BE VERIFIED` immediately instead of forcing a muddy repository-wide rename.

NOW EXECUTE
