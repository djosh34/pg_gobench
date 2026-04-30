## Bug: Multi-source credential validation reports the wrong error when `env-ref` is unset <status>done</status> <passes>true</passes> <priority>high</priority>

<description>
During the story-99 manual verification pass, strict config rejection mostly worked but one user-visible rejection path was wrong.

Reproduction against the real shipped Docker Compose environment:

1. Start the stack:
   `docker compose -f examples/docker-compose-postgres/compose.yaml -p pg-gobench-quickstart up -d --build`
2. Create an invalid config file with more than one credential source for the same field:
   ```yaml
   source:
     host: postgres
     port: 5432
     dbname: pg_gobench
     username:
       value: benchmark_user
       env-ref: POSTGRES_USERNAME
     password:
       secret-file: /run/secrets/postgres-password
   ```
3. Run the shipped container against that config without setting `POSTGRES_USERNAME`:
   `docker run --rm --network pg-gobench-quickstart_default -v "$PWD/invalid-strict.yaml:/app/config/pg_gobench.yaml:ro" -v "$PWD/examples/docker-compose-postgres/secrets/postgres-password.txt:/run/secrets/postgres-password:ro" pg-gobench-quickstart-pg_gobench:latest -config /app/config/pg_gobench.yaml -addr 0.0.0.0:8080`

Observed result:

- startup fails with `parse config: load config: validate config file "/app/config/pg_gobench.yaml": source.username env-ref "POSTGRES_USERNAME" is not set`

Expected result:

- startup fails with the structural validation error for the real problem:
  `source.username must set exactly one of value, env-ref, or secret-file`
- the exact-one validation must win regardless of whether the referenced environment variable exists

Additional confirmation:

- running the same invalid config with `POSTGRES_USERNAME=benchmark_user` changes the error to `source.username must set exactly one of value, env-ref, or secret-file`

This means the validation path depends on ambient environment state and can surface a misleading error for the same invalid config shape. That breaks the strict config-rejection contract verified by the manual task and makes operator troubleshooting worse than necessary.
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

<plan>.ralph/tasks/bugs/bug-multi-source-credential-validation-reports-wrong-error_plans/2026-04-30-multi-source-credential-validation-plan.md</plan>
