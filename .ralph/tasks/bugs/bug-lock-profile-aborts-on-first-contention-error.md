## Bug: Lock Profile Aborts On First Contention Error <status>done</status> <passes>true</passes> <priority>high</priority>

<description>
During the story-99 manual verification pass against the real Docker Compose deployment, the advanced `lock` workload aborted the entire benchmark as soon as PostgreSQL returned a row-lock conflict.

Reproduction used the shipped Compose example and public HTTP API:

1. Start the stack:
   `docker compose -f examples/docker-compose-postgres/compose.yaml -p pg-gobench-manual up -d --build`
2. Start a high-contention lock run:
   `curl -X POST http://127.0.0.1:8080/benchmark/start -H 'Content-Type: application/json' -d '{"scale":1,"clients":32,"duration_seconds":10,"warmup_seconds":1,"reset":false,"profile":"lock"}'`
3. Immediately inspect the result surface:
   `curl http://127.0.0.1:8080/benchmark/results`

Observed result:

- the start call returns `status: "running"`
- within milliseconds the run flips to `status: "failed"`
- `error` and `stats.latest_error` are `lock contention: ERROR: could not obtain lock on row in relation "accounts" (SQLSTATE 55P03)`
- `stats.total_operations`, `stats.successful_operations`, and `stats.failed_operations` are all `0`

This breaks the advanced workload contract from `.ralph/tasks/story-06-advanced-workloads/task-01-join-lock-contention-workloads.md`, which says contention SQL errors must be counted and surfaced as benchmark operation errors instead of terminating the whole run.
</description>

<mandatory_red_green_tdd>
Use Red-Green TDD to solve the problem.
You must make ONE test, and then make ONE test green at the time.

Then verify if bug still holds. If yes, create new Red test, and continue with Red-Green TDD until it does work.
</mandatory_red_green_tdd>

<acceptance_criteria>
- [x] I created a Red unit and/or integration test that captures the bug
- [x] I made the test green by fixing
- [x] A lock-profile run that hits contention records failed benchmark operations and surfaced error text without terminating the entire benchmark immediately
- [x] `GET /benchmark/results` shows non-zero failed lock-related operations when contention happens, rather than an all-zero failed run snapshot
- [x] I manually verified the bug, and created a new Red test if not working still
- [x] `make check` — passes cleanly
- [x] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [x] `make lint` — passes cleanly
- [x] If this bug impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only) (not applicable for this bug)
</acceptance_criteria>

<plan>.ralph/tasks/bugs/bug-lock-profile-aborts-on-first-contention-error_plans/2026-04-30-lock-profile-contention-plan.md</plan>
