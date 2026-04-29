## Task: 02 Add Docker Compose PostgreSQL Example <status>not_started</status> <passes>false</passes>

<blocked_by>.ralph/tasks/story-05-delivery/task-01-scratch-dockerfile.md</blocked_by>

<description>
**Goal:** Add a Docker Compose example that runs PostgreSQL and the `pg_gobench` scratch image together. The example must demonstrate the real YAML config format, including manual host/port/dbname fields and username/password supplied through `env-ref` and/or `secret-file`.

The app config must still be file-based. Compose may set environment variables only to demonstrate explicit username/password `env-ref`, not to configure the application generally. Include mounted config and secret files. Include service healthchecks where useful, published HTTP port, and a network connecting Postgres and the benchmark service.

This is a non-code packaging/example task. Do not use TDD for this task. Verification must run the compose stack.
</description>

<acceptance_criteria>
- [ ] Compose example includes PostgreSQL and `pg_gobench` services.
- [ ] Compose example mounts a YAML config file into the app container.
- [ ] Compose example demonstrates `env-ref` and/or `secret-file` only for username/password.
- [ ] Compose example does not imply app-wide env-var config.
- [ ] Manual verification: `docker compose` for the example starts PostgreSQL and `pg_gobench`.
- [ ] Manual verification: `/healthz` responds from the published HTTP port.
- [ ] Manual verification: `/readyz` succeeds against the Compose PostgreSQL service or returns a clear JSON Go error if setup is intentionally incomplete.
</acceptance_criteria>
