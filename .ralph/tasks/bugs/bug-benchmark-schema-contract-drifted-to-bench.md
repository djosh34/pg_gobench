## Bug: Benchmark schema contract drifted from `pg_gobench` to `bench` <status>done</status> <passes>true</passes> <priority>high</priority>

<description>
The manual verification pass for `.ralph/tasks/story-99-manual-verify-everything/task-01-manual-verify-everything.md` found a real contract mismatch in the shipped product.

Reproduction against the real Docker Compose example:

1. `docker compose -f examples/docker-compose-postgres/compose.yaml -p pg-gobench-quickstart up -d --build`
2. `curl --fail -X POST http://127.0.0.1:8080/benchmark/start -H 'Content-Type: application/json' -d '{"scale":1,"clients":2,"duration_seconds":20,"warmup_seconds":1,"reset":true,"profile":"mixed","read_percent":80}'`
3. `docker exec pg-gobench-quickstart-postgres-1 psql -U benchmark_user -d pg_gobench -At -c "SELECT schemaname || '.' || tablename FROM pg_tables ORDER BY 1;"`

Observed tables:

- `bench.accounts`
- `bench.branches`
- `bench.history`
- `bench.tellers`

Expected contract from completed story artifacts:

- `.ralph/tasks/story-03-core-benchmark/task-01-benchmark-schema-scale.md` says all benchmark-owned objects must live under a dedicated schema named `pg_gobench`
- `.ralph/tasks/story-99-manual-verify-everything/task-01-manual-verify-everything.md` requires manual verification of benchmark schema setup under `pg_gobench`

This means the previous reserved-prefix bug was "fixed" by changing the runtime schema name to `bench`, but the product contract, completed task acceptance, and verification surface were not reconciled. The result is a shipped behavior/documentation/spec mismatch.

Fix direction:

- choose one valid dedicated schema contract that PostgreSQL accepts
- make code, tests, docs, task artifacts, and manual verification all agree on that one contract
- remove stale `pg_gobench` schema expectations once the new contract is adopted
</description>

<mandatory_red_green_tdd>
Use Red-Green TDD to solve the problem.
You must make ONE test, and then make ONE test green at the time.

Then verify if bug still holds. If yes, create new Red test, and continue with Red-Green TDD until it does work.
</mandatory_red_green_tdd>

<acceptance_criteria>
- [x] I created a Red unit and/or integration test that captures the bug
- [x] I made the test green by fixing
- [x] I manually verified the bug, and created a new Red test if not working still
- [x] `make check` — passes cleanly
- [x] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [x] `make lint` — passes cleanly
- [ ] If this bug impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>

<plan>.ralph/tasks/bugs/bug-benchmark-schema-contract-drifted-to-bench_plans/2026-04-30-benchmark-schema-contract-plan.md</plan>
