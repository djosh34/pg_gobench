## Task: 02 Implement Strict YAML Config With Secret References <status>done</status> <passes>true</passes>

<blocked_by>.ralph/tasks/story-01-foundation/task-01-bootstrap-go-http-service.md</blocked_by>

<description>
Must use tdd skill to complete

**Goal:** Implement the YAML configuration loader and validator for PostgreSQL connection settings. The config file is the only way to set database connection parameters. Do not accept connection strings. Do not use environment variables for general configuration.

The supported config shape must include a `source` object with manual fields for `host`, `port`, `username`, `password`, `dbname`, and optional TLS file paths:

```yaml
source:
  host: localhost
  port: 5432
  username:
    env-ref: POSTGRES_USERNAME
  password:
    secret-file: ../path/to/secret
  dbname: postgres
  tls:
    ca_cert: /path/to/ca.crt
    cert: /path/to/client.crt
    key: /path/to/client.key
```

`username` and `password` must each support exactly one of `value`, `env-ref`, or `secret-file`. `env-ref` means lookup that exact environment variable name at resolution time for only that username/password field. `secret-file` means read the referenced file path at resolution time for only that username/password field. TLS fields are paths only; do not add inline TLS PEM or env-ref support for TLS.

Validation must be strict: unknown fields fail; missing required source fields fail; multiple secret source modes in one username/password field fail; empty resolved username/password fail; invalid port fails. This is a greenfield project, so do not preserve or introduce legacy config names.
</description>

<acceptance_criteria>
- [x] TDD red/green coverage exists for valid literal, env-ref, and secret-file username/password resolution.
- [x] TDD red/green coverage exists for strict unknown-field rejection and all required validation failures.
- [x] TDD red/green coverage exists proving env vars are not expanded or read anywhere except explicit username/password `env-ref`.
- [x] TDD red/green coverage exists proving TLS values are treated as file paths only.
- [x] The application accepts a config path through an explicit command-line flag such as `-config`.
- [x] Secret-file reads trim only conventional trailing line endings if needed and fail loudly on unreadable or empty files.
- [x] Errors are returned with useful context and are not swallowed.
- [x] `make check` — passes cleanly
- [x] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [x] `make lint` — passes cleanly
- [x] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only, not required for this task)
</acceptance_criteria>

<plan>.ralph/tasks/story-01-foundation/task-02-yaml-config-secrets_plans/2026-04-30-yaml-config-plan.md</plan>
