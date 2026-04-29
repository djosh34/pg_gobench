## Task: 01 Bootstrap Go HTTP Service <status>not_started</status> <passes>false</passes>

<description>
Must use tdd skill to complete

**Goal:** Create the initial Go project skeleton for `pg_gobench`, a greenfield PostgreSQL benchmark control service. The service must be a Go application with an HTTP server process that can later host JSON benchmark control endpoints, readiness endpoints, Prometheus metrics, and a static HTML control page.

The implementation must establish the module layout, command entrypoint, server construction, graceful shutdown, and repository quality gates. Add a `Makefile` or equivalent repo-local commands for `make check`, `make test`, and `make lint`; these commands must actually run useful validation and must fail when prerequisites are missing. No tests may be skipped. Do not add legacy compatibility shims or placeholder behavior that hides errors.

The HTTP server must support configurable bind address through an explicit command-line flag such as `-addr`; do not use environment variables for application config. There is intentionally no HTTP auth, no HTTPS listener, and no TLS termination in this service.
</description>

<acceptance_criteria>
- [ ] TDD red/green coverage exists for server construction, bind-address parsing, graceful shutdown wiring, and at least one basic health handler.
- [ ] The application starts an HTTP server from a Go command entrypoint and fails loudly on invalid startup configuration.
- [ ] The server has no auth and no HTTPS support.
- [ ] No environment variables are used for application configuration.
- [ ] Errors are returned or logged and surfaced; no errors are swallowed or ignored.
- [ ] `make check` — passes cleanly
- [ ] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [ ] `make lint` — passes cleanly
- [ ] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>
