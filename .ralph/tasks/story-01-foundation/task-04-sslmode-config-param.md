## Task: 04 Add Explicit PostgreSQL sslmode Config Parameter <status>done</status> <passes>true</passes>

<blocked_by>.ralph/tasks/story-01-foundation/task-03-database-sql-connector.md</blocked_by>

<description>
Must use tdd skill to complete

**Goal:** Add `source.sslmode` as an explicit YAML configuration parameter for the PostgreSQL source connection. Operators must be able to choose the PostgreSQL SSL mode from config instead of having the database layer derive `disable` or `verify-full` from whether TLS file paths are present.

The supported config shape must include `sslmode` under `source` as a literal scalar string:

```yaml
source:
  host: postgres
  port: 5432
  dbname: pg_gobench
  sslmode: verify-full
  username:
    env-ref: POSTGRES_USERNAME
  password:
    secret-file: /run/secrets/postgres-password
  tls:
    ca_cert: /run/certs/ca.pem
    cert: /run/certs/client.crt
    key: /run/certs/client.key
```

Allow only PostgreSQL/pgx SSL mode values that this project intentionally supports: `disable`, `allow`, `prefer`, `require`, `verify-ca`, and `verify-full`. Reject unknown values, empty values, non-string values, and config files missing `source.sslmode`. Do not accept connection strings and do not add environment-variable expansion or secret-reference support for `sslmode`.

The database connection adapter must pass the configured `sslmode` into pgx connection construction. Remove the current implicit behavior that chooses `verify-full` when any TLS path exists and `disable` otherwise. TLS file paths remain literal path fields and still provide CA/client certificate material when TLS is enabled. Validate incompatible combinations instead of silently ignoring them; for example, `source.tls.*` paths with `source.sslmode: disable` must fail with a useful error.

Update README, Docker Compose, and Kubernetes example config files so every documented/runnable config includes `source.sslmode`. This is greenfield work: do not preserve a config shape where `sslmode` is optional for backwards compatibility.
</description>

<acceptance_criteria>
- [x] TDD red/green coverage exists for loading a valid config with `source.sslmode` and exposing it on `config.Source`.
- [x] TDD red/green coverage exists for rejecting missing, empty, non-string, and unknown `source.sslmode` values.
- [x] TDD red/green coverage exists proving `source.sslmode` is treated as a literal YAML value and is not resolved from environment variables, secret files, or connection strings.
- [x] TDD red/green coverage exists proving the database adapter passes the configured SSL mode to pgx instead of deriving it from TLS path presence.
- [x] TDD red/green coverage exists for rejecting incompatible TLS path usage when `source.sslmode: disable`.
- [x] README, Docker Compose config, and Kubernetes config examples include `source.sslmode`.
- [x] Errors include useful field context such as `source.sslmode` or the incompatible `source.tls` field.
- [x] `make check` — passes cleanly
- [x] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [x] `make lint` — passes cleanly
- [ ] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>

<plan>.ralph/tasks/story-01-foundation/task-04-sslmode-config-param_plans/2026-04-30-sslmode-config-plan.md</plan>
