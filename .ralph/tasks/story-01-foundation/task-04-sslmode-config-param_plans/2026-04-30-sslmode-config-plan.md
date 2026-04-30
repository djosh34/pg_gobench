# SSLMode Config Parameter Plan

## Scope

Implement explicit `source.sslmode` parsing and validation in the config layer, then pass that validated value through the database adapter without any adapter-side SSL mode derivation.

This task should:

- require `source.sslmode` in every config
- accept only the supported literal YAML scalar values: `disable`, `allow`, `prefer`, `require`, `verify-ca`, and `verify-full`
- reject missing, empty, non-string, unknown, env-style, secret-style, and connection-string-shaped `source.sslmode` inputs
- move config-combination validation into `internal/config` so `internal/database` consumes one validated shape
- pass the configured SSL mode to pgx directly
- update README and runnable example configs so every documented config includes `source.sslmode`

This turn is planning-only. Execution belongs to the next turn unless the design proves incomplete and must be reopened.

## Public Interface

- `internal/config` remains the owner of YAML parsing, allowed-value validation, and config-shape validation.
- `config.Source` should grow one explicit field:
  - `SSLMode config.SSLMode`
- `config.SSLMode` should be a small validated type with constants for:
  - `disable`
  - `allow`
  - `prefer`
  - `require`
  - `verify-ca`
  - `verify-full`
- `internal/database` should continue exposing:
  - `Open(source config.Source) (*sql.DB, error)`
  - `CheckReadiness(ctx context.Context, db pinger) error`

No package outside `internal/config` should decide which SSL modes are supported.

## Boundary Decision

Use `internal/config` as the single owner of SSL mode and TLS compatibility validation.

This is the concrete `improve-code-boundaries` cleanup for this task:

- remove the database-layer `sslMode(source.TLS)` inference entirely
- stop treating TLS path presence as an alternate source of SSL mode
- keep one shared connection settings shape in `config.Source` instead of adding a parallel database-only options type
- validate incompatible config combinations once, at config load time, not again later in the adapter

That addresses the `validation-outside-config` and `shared-connection-shape` smells directly.

## Design Notes

Planned validated config behavior:

- `source.sslmode` is required
- `source.sslmode` must be a YAML scalar string
- the string must match one supported pgx/PostgreSQL SSL mode exactly
- empty strings fail with an error mentioning `source.sslmode`
- mapping nodes such as:
  - `sslmode: { env-ref: PGSSLMODE }`
  - `sslmode: { secret-file: /run/secrets/sslmode }`
  must fail because `sslmode` is literal-only
- connection-string-shaped values such as `sslmode=verify-full` or full DSNs must fail because the field is one literal mode, not a conninfo fragment

Planned TLS compatibility validation inside `internal/config`:

- if `source.sslmode` is `disable`, any configured `source.tls.ca_cert`, `source.tls.cert`, or `source.tls.key` must fail with useful field context
- partial client-cert pairs should fail during config validation:
  - `source.tls.cert` without `source.tls.key`
  - `source.tls.key` without `source.tls.cert`

Planned database behavior:

- build one pgx config from validated `config.Source`
- set `sslmode` from `source.SSLMode` directly
- keep TLS file loading in `internal/database`, because file readability and PEM parsing are runtime I/O, not YAML-shape validation
- only build and attach a custom `tls.Config` when TLS paths are present
- remove the implicit `verify-full`/`disable` decision completely

## TDD Slices

Use vertical red-green slices only.

- [x] Slice 1: failing config-load test for a valid YAML file with explicit `source.sslmode`, proving the loaded `config.Source` exposes the exact validated value
- [x] Slice 2: failing config-load tests for missing, empty, non-string, and unknown `source.sslmode` values with errors that mention `source.sslmode`
- [x] Slice 3: failing config-load tests proving `source.sslmode` is literal-only and rejects env-ref, secret-file, and connection-string-shaped values
- [x] Slice 4: failing database test proving the adapter uses the configured SSL mode instead of deriving it from TLS path presence
- [x] Slice 5: failing config-load tests for incompatible TLS usage when `source.sslmode: disable`, and for partial client certificate/key pairs
- [x] Slice 6: refactor after green to keep all SSL mode knowledge in `internal/config` and remove dead adapter helpers
- [x] Slice 7: update README, Docker Compose config, and Kubernetes config example so every documented config contains `source.sslmode`

## File Plan

- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/database/database.go`
- `internal/database/database_test.go`
- `README.md`
- `examples/docker-compose-postgres/config/pg_gobench.yaml`
- `examples/k8s/20-configmap.yaml`

Possible cleanup during execution if it improves boundaries:

- rename helper logic around TLS presence so it reflects "custom TLS material paths exist" rather than "TLS enabled"
- delete any helper that exists only to infer SSL mode from TLS paths
- keep new parsing and validation helpers private to `internal/config`

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless the task explicitly expands into the long lane or changes long-test selection

NOW EXECUTE
