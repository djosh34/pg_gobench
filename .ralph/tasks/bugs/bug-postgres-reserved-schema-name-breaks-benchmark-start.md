## Bug: PostgreSQL benchmark start fails because schema name uses reserved `pg_` prefix <status>done</status> <passes>true</passes> <priority>high</priority>

<description>
Manual verification for `.ralph/tasks/story-07-k8s/task-01-k8s-simple-deployment-configmap.md` hit a real runtime failure after the Kubernetes deployment became healthy.

Evidence:

- `POST /benchmark/start` returned `500 Internal Server Error`
- response body: `{"error":"setup benchmark schema: ERROR: unacceptable schema name \"pg_gobench\" (SQLSTATE 42939)"}`
- reproduced against a real local kind cluster running `postgres:18-bookworm`

The current benchmark bootstrap uses the schema name `pg_gobench`, but PostgreSQL rejects schema names with the reserved `pg_` prefix. This blocks any fresh benchmark run, including the Kubernetes example verification flow.
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
- [x] If this bug impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only) (not applicable for this bug)
</acceptance_criteria>
