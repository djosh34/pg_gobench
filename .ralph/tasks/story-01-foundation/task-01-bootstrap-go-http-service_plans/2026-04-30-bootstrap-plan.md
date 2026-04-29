# Bootstrap Go HTTP Service Plan

## Scope

Build the first runnable `pg_gobench` Go service as a small HTTP process with:

- a command entrypoint
- explicit `-addr` flag parsing
- one health endpoint
- graceful shutdown wiring
- repo-local `make check`, `make lint`, and `make test`

This turn is planning-only. Execution belongs to the next turn unless the design needs to be re-opened.

## Public Interface

- Binary entrypoint: `cmd/pg_gobench/main.go`
- CLI surface:
  - `pg_gobench -addr 127.0.0.1:8080`
  - no environment-variable configuration
- HTTP surface:
  - `GET /healthz` returns `200 OK` with a minimal plain response body
- Runtime surface:
  - bootstrap code returns errors upward for invalid flag values, listener setup failures, and shutdown failures

## Boundary Decision

Use one application config type across flag parsing and server startup instead of introducing separate flag DTOs, env loaders, or bootstrap translation layers.

Planned boundary:

- `internal/app.Config` owns the bind address
- `internal/app.Run(ctx, cfg, stdout, stderr) error` owns process startup and graceful shutdown
- `internal/httpserver` owns HTTP handler construction and `http.Server` setup

This deliberately avoids early boundary mud:

- no config package plus mapper package plus server-options adapter
- no stringly startup wiring spread across `main`
- no placeholder abstraction for auth, TLS, or future endpoints

## TDD Slices

Use vertical red-green slices only.

- [x] Slice 1: failing test for flag parsing accepting explicit `-addr` and rejecting invalid/unknown CLI input through a public parse function
- [x] Slice 2: failing test for server construction exposing `/healthz` successfully through an in-memory handler/server test
- [x] Slice 3: failing test for application run wiring that starts the server and exits cleanly when the context is canceled
- [x] Slice 4: failing test for startup failure surfacing an invalid bind address or listener error without swallowing it
- [x] Slice 5: refactor after green to keep `main` thin and keep HTTP/bootstrap concerns separated

## File Plan

- `go.mod`
- `cmd/pg_gobench/main.go`
- `internal/app/app.go`
- `internal/httpserver/server.go`
- `internal/app/app_test.go`
- `internal/httpserver/server_test.go`
- `Makefile`

## Quality Gates

- `make check` runs meaningful static validation and fails loudly if required tools are unavailable
- `make lint` is wired to the same validation lane as required by the current repo instructions
- `make test` runs the default Go test suite with no skipped tests
- no `make test-long` unless execution proves this task changed long-test selection, which is unlikely for bootstrap

## Execution Notes

- Prefer standard library only unless a concrete need appears during RED
- Use ephemeral listeners in tests where possible to avoid port conflicts
- Use `signal.NotifyContext` or equivalent explicit shutdown wiring in production code, but keep the signal boundary out of most tests by testing `Run` via context cancellation
- If the design proves wrong during RED, switch this plan back to `TO BE VERIFIED` immediately instead of papering over the mismatch

NOW EXECUTE
