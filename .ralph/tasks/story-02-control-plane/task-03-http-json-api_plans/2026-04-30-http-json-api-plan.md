# HTTP JSON API Plan

## Scope

Expose the existing single-run coordinator through a compact JSON HTTP API and wire readiness to the configured database source.

This task should deliver:

- `POST /benchmark/start`
- `POST /benchmark/alter`
- `POST /benchmark/stop`
- `GET /benchmark`
- `GET /benchmark/results`
- `GET /healthz`
- `GET /readyz`

This task should not deliver:

- HTML or server-side rendering
- auth, cookies, HTTPS, sessions, or API keys
- persisted run history or on-disk results
- a speculative stats subsystem before story 04 exists

## Public Interface

Keep `internal/httpserver` as the transport owner and give it only the runtime dependencies it needs.

Planned `httpserver` surface:

- `func New(addr string, deps Dependencies) *http.Server`
- `type Dependencies struct`

Planned `Dependencies` fields:

- a small benchmark control interface with:
  - `Start(context.Context, benchmark.StartOptions) (benchmarkrun.State, error)`
  - `Alter(benchmark.AlterOptions) (benchmarkrun.State, error)`
  - `Stop() (benchmarkrun.State, error)`
  - `State() benchmarkrun.State`
- a readiness function:
  - `Ready(context.Context) error`

Do not pass `*sql.DB`, config structs, or the whole app into `internal/httpserver`. The server should depend on the coordinator contract plus a readiness callback, nothing broader.

## Boundary Decision

The `improve-code-boundaries` move for this task is: keep the transport JSON contract owned by the domain structs that already represent it, and do not invent duplicate API DTOs for benchmark state/options.

Concrete cleanup to do during execution:

- keep `benchmarkrun.State` as the response type for `GET /benchmark`
- return the same compact state snapshot from `GET /benchmark/results` for now, because no stats module exists yet and inventing a placeholder results type would be speculative boundary spam
- add JSON tags to exported benchmark option structs so `benchmarkrun.State` can serialize cleanly without an extra response mapping layer
- remove the duplicate private `alterPayload` shape and decode alter JSON directly into `benchmark.AlterOptions`, since that exported type already uses pointer fields to preserve presence

This keeps the option contract inside `internal/benchmark`, runtime lifecycle inside `internal/benchmarkrun`, and transport orchestration inside `internal/httpserver`.

## HTTP Contract

Success responses should be JSON for every endpoint.

Planned response shapes:

- `GET /healthz` -> compact JSON success payload such as `{"status":"ok"}`
- `GET /readyz` success -> same compact JSON success payload
- `GET /readyz` failure -> JSON error payload containing the raw Go error string
- `GET /benchmark` -> direct JSON encoding of `benchmarkrun.State`
- `GET /benchmark/results` -> same state snapshot until story 04 introduces real stats
- `POST /benchmark/start` -> JSON `benchmarkrun.State`
- `POST /benchmark/alter` -> JSON `benchmarkrun.State`
- `POST /benchmark/stop` -> JSON `benchmarkrun.State`

Planned error payload shape:

```json
{"error":"<go error text>"}
```

Do not add nested error categories, codes, or HTML error bodies.

Planned status mapping:

- `200 OK` for successful `GET` and successful benchmark operations
- `400 Bad Request` for malformed JSON, unknown JSON fields, trailing JSON, and benchmark validation errors
- `405 Method Not Allowed` for wrong HTTP methods
- `409 Conflict` for coordinator state conflicts such as start-while-running or alter-while-not-running
- `503 Service Unavailable` for readiness failures
- `500 Internal Server Error` for unexpected runtime failures that are not client errors

## App Wiring

`internal/app` should start owning real runtime dependencies for the server:

- open a database handle from `cfg.Source` during startup
- close the database handle during shutdown
- instantiate the benchmark coordinator
- pass `func(ctx context.Context) error { return database.CheckReadiness(ctx, db) }` into `httpserver.New`

This keeps readiness logic in `internal/database` and avoids dragging database details into `internal/httpserver`.

The coordinator can still use the current nil runner in this task. A `POST /benchmark/start` failure caused by missing workload implementation should surface as compact JSON error text, which is acceptable until the later workload stories land.

## TDD Strategy

Use vertical slices against the public HTTP surface only. Tests should call `server.Handler.ServeHTTP` with fake dependencies and assert on HTTP status plus JSON body. Do not test private helpers.

Planned slices:

- [x] Slice 1: failing test for `GET /benchmark` returning JSON state with transport field names (`status`, `options.scale`, etc.), then minimal handler code to pass
- [x] Slice 2: failing test for `POST /benchmark/start` decoding JSON, calling the coordinator, and returning updated state
- [x] Slice 3: failing test for rejecting a second start with `409` and compact JSON error text
- [x] Slice 4: failing test for `POST /benchmark/alter` success and failing test for validation or not-running conflict paths
- [x] Slice 5: failing test for `POST /benchmark/stop` success and idempotent state response
- [x] Slice 6: failing test for `GET /benchmark/results` returning the current compact snapshot without inventing a second result DTO
- [x] Slice 7: failing test for `GET /healthz` success JSON and `GET /readyz` success JSON
- [x] Slice 8: failing test for `GET /readyz` failure returning `503` plus the raw Go ping error text in JSON
- [x] Slice 9: failing tests for malformed JSON, unknown fields, trailing data, and invalid methods across the benchmark endpoints
- [x] Slice 10: refactor after green to keep request decoding in `internal/benchmark`, keep response encoding direct, and remove any single-use helper clutter

Additional benchmark package coverage needed during execution:

- [x] Add a red/green test proving JSON encoding of `benchmark.StartOptions` produces snake_case field names through the public type
- [x] Add a red/green test proving `benchmark.DecodeAlterOptions` can decode directly into the exported `AlterOptions` shape while still rejecting unknown fields

## File Plan

Expected files:

- `internal/httpserver/server.go`
- `internal/httpserver/server_test.go`
- `internal/benchmark/options.go`
- `internal/benchmark/options_test.go`
- `internal/app/app.go`
- `internal/app/app_test.go`

Avoid adding a new `internal/api`, `internal/httpdto`, or response-mapper package. The current codebase is too small for that split, and it would create exactly the duplicate boundary this task should remove.

## Quality Gates

- [x] `make check`
- [x] `make lint`
- [x] `make test`
- no `make test-long` unless execution unexpectedly changes long-test selection, which this task should not do

If execution shows that `/benchmark/results` already needs a distinct stats-bearing shape, or that the coordinator must expose a richer public contract than `State()` for this task to stay clean, switch this plan back to `TO BE VERIFIED` immediately instead of forcing a muddy API boundary.

NOW EXECUTE
