# YAML Config Secrets Plan

## Scope

Implement strict YAML-backed PostgreSQL source configuration with secret references and explicit CLI loading:

- add `-config` flag support
- load a YAML file as the only source of database connection settings
- resolve `username` and `password` from exactly one of `value`, `env-ref`, or `secret-file`
- reject unknown fields and incomplete or invalid config
- keep TLS fields as plain file-path strings only

This turn is planning-only. Execution belongs to the next turn unless the design proves incomplete and must be reopened.

## Public Interface

- CLI surface:
  - `pg_gobench -addr 127.0.0.1:8080 -config /path/to/config.yaml`
  - `-config` is required for application startup
- YAML surface:

```yaml
source:
  host: localhost
  port: 5432
  username:
    value: postgres
  password:
    secret-file: ../secrets/postgres-password
  dbname: postgres
  tls:
    ca_cert: /path/to/ca.crt
    cert: /path/to/client.crt
    key: /path/to/client.key
```

- Runtime surface:
  - one validated config object reaches the application runtime
  - credential values are already resolved before runtime uses them
  - all config and secret resolution errors return with field/path context and are never swallowed

## Boundary Decision

Use `internal/config` as the only owner of YAML parsing, strict decoding, secret resolution, and validation.

Planned boundary:

- `internal/config` exposes one loader, likely `Load(path string) (Config, error)`
- `internal/config.Config` is the validated runtime config shape consumed by the rest of the program
- `internal/app.ParseConfig(args)` remains the CLI boundary only: parse flags, require `-config`, call `config.Load`, and return a single app/runtime config

This is the explicit `improve-code-boundaries` refactor for this task:

- remove database-config knowledge from `internal/app`
- avoid raw YAML DTOs or half-validated secret refs escaping the config package
- validate once inside config loading and pass only the final reduced shape outward
- keep CLI parsing separate from YAML schema logic instead of mixing both concerns in `app`

## Config Shape

Plan the validated runtime shape around one source config, not multiple duplicate structs:

- top-level config contains `Source`
- `Source` contains:
  - `Host string`
  - `Port int`
  - `Username string`
  - `Password string`
  - `DBName string`
  - optional `TLS` with path strings only

The YAML-only helper shapes stay private to `internal/config`:

- raw document struct for strict YAML decoding
- private credential reference struct with optional `value`, `env-ref`, and `secret-file`

No connection string DTOs, no env-expanding general config layer, and no public "raw config" type.

## Validation Rules

- unknown YAML fields fail through strict decoder behavior
- `source` is required
- `host`, `port`, `username`, `password`, and `dbname` are required
- `port` must be a valid TCP port in range `1..65535`
- each credential field must specify exactly one source mode
- `env-ref` resolves only that exact environment variable name for that one credential field
- `secret-file` resolves only that file for that one credential field
- secret file content trims only trailing `\n` and `\r` bytes, then must remain non-empty
- unresolved, unreadable, multi-mode, or empty credentials fail with useful field/path context
- TLS fields are plain paths only and must not support `env-ref`, `secret-file`, or inline PEM blocks

## TDD Slices

Use vertical red-green slices only.

- [ ] Slice 1: failing public test for `-config` flag requirement and successful loading of a minimal YAML file with literal `value` credentials
- [ ] Slice 2: failing public test for `env-ref` resolution on `username`/`password`, including empty or missing env vars failing loudly
- [ ] Slice 3: failing public test for `secret-file` resolution, including newline trimming plus unreadable or empty secret-file failures
- [ ] Slice 4: failing public test for strict validation: unknown YAML fields, missing required fields, invalid port, and multiple credential source modes
- [ ] Slice 5: failing public test proving env vars are not expanded anywhere except explicit credential `env-ref`, and proving TLS values are treated as literal path strings only
- [ ] Slice 6: refactor after green to keep `app.ParseConfig` thin and keep raw config-only types private to `internal/config`

## File Plan

- `go.mod`
- `cmd/pg_gobench/main.go`
- `internal/app/app.go`
- `internal/app/app_test.go`
- `internal/config/config.go`
- `internal/config/config_test.go`

Possible cleanup during execution if it improves boundaries:

- shrink or rename `internal/app.Config` if it becomes a thin wrapper around runtime config
- remove any helper or DTO that exists only to shuttle raw config between packages

## Implementation Notes

- prefer `gopkg.in/yaml.v3` with a decoder configured to reject unknown fields
- keep secret-resolution helpers private to `internal/config`
- use temp files and `t.Setenv` in tests instead of mocks
- keep tests behavior-focused through public loaders or CLI parsing entrypoints
- if runtime config shape needs to change materially during RED, switch this plan back to `TO BE VERIFIED` immediately

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless execution changes long-test selection or the task proves it is explicitly required

NOW EXECUTE
