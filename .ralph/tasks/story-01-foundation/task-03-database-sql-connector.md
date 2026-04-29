## Task: 03 Build database/sql PostgreSQL Connector <status>done</status> <passes>true</passes>

<blocked_by>.ralph/tasks/story-01-foundation/task-02-yaml-config-secrets.md</blocked_by>

<description>
Must use tdd skill to complete

**Goal:** Build the PostgreSQL connection layer using Go's `database/sql` interface. The selected driver is `github.com/jackc/pgx/v5/stdlib`, but all application code must depend on `database/sql` abstractions rather than using pgx connections directly.

Connection parameters must be constructed from the validated YAML config fields, not from a user-supplied connection string. Support host, port, username, password, dbname, and TLS file paths. Add database readiness functionality that can ping the configured source database and report errors directly.

Do not add backwards-compatible connection-string parsing. Do not read database connection settings from environment variables except username/password values resolved by the config secret-reference system.
</description>

<acceptance_criteria>
- [x] TDD red/green coverage exists for converting validated config into driver connection settings without accepting raw connection strings.
- [x] TDD red/green coverage exists for TLS path handling and invalid TLS config failures where applicable.
- [x] TDD red/green coverage exists for readiness/ping success and failure behavior using an appropriate test double or integration-backed test.
- [x] Application code interacts with PostgreSQL through `database/sql`; pgx direct connection APIs are not used outside the driver registration/adapter boundary.
- [x] Connection failures are surfaced to callers and JSON health/readiness responses later can print the Go error text.
- [x] `make check` — passes cleanly
- [x] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [x] `make lint` — passes cleanly
- [ ] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>

<plan>.ralph/tasks/story-01-foundation/task-03-database-sql-connector_plans/2026-04-30-database-sql-connector-plan.md</plan>
