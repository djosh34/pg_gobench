# Database SQL Connector Plan

## Scope

Implement the PostgreSQL source connection layer through `database/sql` using `github.com/jackc/pgx/v5/stdlib`, with no raw user-supplied connection string parsing.

This task should:

- convert validated YAML-backed source settings into driver config exactly once
- open a `*sql.DB` through the stdlib adapter boundary
- build TLS client settings from configured file paths and fail loudly on invalid TLS inputs
- provide database readiness behavior that pings the configured source database and returns the raw Go error upward

This turn is planning-only. Execution belongs to the next turn unless the design proves incomplete and must be reopened.

## Public Interface

- Config surface remains the validated runtime shape from `internal/config`:
  - `config.Source`
- Database surface should live in a new package, likely `internal/database`:
  - `Open(source config.Source) (*sql.DB, error)`
  - `CheckReadiness(ctx context.Context, db pinger) error`
- Testing seam for readiness:
  - keep `CheckReadiness` on a tiny `PingContext(context.Context) error` interface so tests can use a focused test double
- Application surface:
  - no caller accepts a raw PostgreSQL connection string
  - callers receive direct Go errors from connector build, open, and ping paths

`pinger` should stay private to the database package unless another package truly needs it.

## Boundary Decision

Use `internal/database` as the only owner of PostgreSQL driver adaptation, TLS material loading, and readiness checks.

Planned boundary:

- `internal/config` remains the only owner of YAML decoding, secret resolution, and validation
- `internal/database` consumes the already-validated `config.Source` and turns it into one `pgx.ConnConfig` / `database/sql` connector path
- `internal/app` remains bootstrap only and must not render DSNs, assemble TLS pools, or know pgx-specific details
- `internal/httpserver` stays request/response only and must not own ping logic

This is the explicit `improve-code-boundaries` refactor for this task:

- keep `config.Source` as the single canonical database settings shape instead of inventing `database.Config`, `database.Options`, or ad-hoc conninfo structs
- perform connection-setting conversion in one place only, once
- keep readiness logic in the database boundary instead of smearing it into handlers or bootstrap
- avoid stringly DSN construction spread across tests or runtime code

## Connection Design

Use the pgx stdlib adapter through its config-based `OpenDB` path rather than `sql.Open("pgx", rawConnString)`.

Planned connection facts sourced from `config.Source`:

- host
- port
- username
- password
- dbname
- optional TLS file paths

Planned behavior:

- build one `pgx.ConnConfig` from `config.Source`
- default to plaintext when no TLS paths are configured
- when any TLS path is configured, require a coherent TLS setup and build a client `tls.Config`
- invalid certificate, key, or CA paths fail during connector construction, not later through swallowed background errors

If execution reveals that partial TLS input should be allowed rather than rejected as incoherent, switch this plan back to `TO BE VERIFIED` immediately.

## Readiness Design

Readiness should be a thin behavior:

- call `PingContext` on the opened database handle
- return `nil` on success
- wrap failures with direct context such as readiness/ping while preserving the original error text

Do not create a large readiness status type or JSON-specific DTO here. Later HTTP work can render the plain error string from this boundary.

## TDD Slices

Use vertical red-green slices only.

- [x] Slice 1: failing public test for opening a database handle from a minimal validated `config.Source` without any raw connection-string input path in application code
- [x] Slice 2: failing public test for connector/TLS construction using literal file paths, plus failing cases for unreadable or invalid CA/cert/key files
- [x] Slice 3: failing public test for readiness success using a focused `PingContext` test double
- [x] Slice 4: failing public test for readiness failure returning the direct Go error text with useful context
- [x] Slice 5: refactor after green to keep all pgx-specific code inside `internal/database` and eliminate any duplicate connection-setting shape that appears during implementation

## File Plan

- `go.mod`
- `go.sum`
- `internal/database/database.go`
- `internal/database/database_test.go`
- `internal/app/app.go`
- `internal/app/app_test.go`

Possible cleanup during execution if it improves boundaries:

- reduce `internal/app.Config` if database settings start being couriered redundantly
- remove any helper that only exists to shuttle connection facts between packages
- keep new database helper functions and interfaces private unless another package must compile against them

## Implementation Notes

- add `github.com/jackc/pgx/v5` and use `github.com/jackc/pgx/v5/stdlib`
- prefer config-based opening through the stdlib adapter instead of parsing a rendered DSN string
- use `crypto/tls`, `crypto/x509`, and file reads inside the database package for TLS material loading
- keep tests behavior-focused through `database.Open` and `database.CheckReadiness`
- if a real integration-backed database test becomes necessary, keep it out of `make test-long` unless the default lane becomes genuinely insufficient and the task explicitly requires the long lane

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless execution changes long-test selection or the task proves it is explicitly required

NOW EXECUTE
