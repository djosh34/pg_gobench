# Standalone HTML Control Page Plan

Plan file: `.ralph/tasks/story-09-ui/task-01-standalone-html-control-page_plans/2026-04-30-standalone-html-control-page-plan.md`

## Scope

Add one standalone browser control page for `pg_gobench` that lives as a single raw HTML file, opens directly from disk, and talks only to the existing HTTP API.

This turn is planning-only. Execution belongs to the next turn after the plan is reviewed and promoted to `NOW EXECUTE`.

This task should deliver:

- one standalone HTML artifact, likely `examples/standalone-control-page.html`
- direct browser control over:
  - API base URL
  - `/healthz`
  - `/readyz`
  - `/benchmark`
  - `/benchmark/results`
  - `/benchmark/start`
  - `/benchmark/alter`
  - `/benchmark/stop`
  - `/metrics`
- a simple operator-first UI with inline CSS and inline JavaScript only
- minimal HTTP API support needed for a `file://` browser page to call the existing JSON/text endpoints honestly
- real browser verification against a running local API

This task should not deliver:

- Go template rendering
- static-file serving from the Go binary
- Node, npm, bundlers, frameworks, or generated frontend assets
- multiple browser assets split across separate JS or CSS files
- duplicate API contracts in a second backend-specific UI layer
- fake verification that only curls endpoints without proving the standalone page works in a browser

## Public Interface

The operator-facing artifact should stay explicit and minimal:

- standalone file:
  - `examples/standalone-control-page.html`
- operator inputs:
  - API base URL text field
  - start controls for:
    - `scale`
    - `clients`
    - `duration_seconds`
    - `warmup_seconds`
    - `reset`
    - `profile`
    - `read_percent`
    - `transaction_mix`
    - `target_tps`
  - alter controls for:
    - `clients`
    - `target_tps`
- operator actions:
  - check health
  - check readiness
  - load state
  - load results
  - start benchmark
  - alter benchmark
  - stop benchmark
  - fetch metrics into the page
  - open metrics in a new browser tab/window
- outputs:
  - clearly rendered request status
  - JSON response panes for state/results/errors
  - metrics text pane for Prometheus output

No new HTTP resources are planned. The page should call the existing routes exactly as they already exist.

## Boundary Decision

The main `improve-code-boundaries` move for this task is to keep the standalone browser artifact and the Go control plane separated cleanly while still making the HTTP boundary browser-safe.

The key design fact is this:

- a raw `file://` page cannot `fetch()` `http://127.0.0.1:8080` JSON successfully unless the API sends CORS headers
- JSON `POST` requests also require `OPTIONS` preflight handling because the page will send `Content-Type: application/json`

That means the clean boundary is:

- the HTML file remains fully standalone and is never served by Go
- the HTTP server gains one central browser-compatibility layer for CORS/preflight instead of per-handler hacks
- existing route shapes, JSON payloads, and benchmark contracts remain unchanged

One concrete boundary simplification is planned:

- add CORS/preflight behavior once at the mux boundary in `internal/httpserver`
- do not duplicate origin/header logic inside each endpoint handler
- keep the HTML page script split by responsibility inside the one file:
  - base URL + request helpers
  - form serialization for start/alter payloads
  - render helpers for JSON and metrics text
  - event wiring
- keep the page working directly with the existing API payloads instead of inventing a second DTO layer in JavaScript

If execution shows that the browser cannot honestly call the API with a small central CORS layer, switch this plan back to `TO BE VERIFIED` immediately instead of forcing template serving or some second proxy path.

## Planned Implementation Shape

Expected file changes:

- add `examples/standalone-control-page.html`
- update `internal/httpserver/server.go` to support browser CORS/preflight centrally
- update `internal/httpserver/server_test.go` with HTTP-level coverage for the CORS/preflight behavior if that support is added
- optionally add one short README pointer only if the new artifact would otherwise be hard to discover

Planned HTML structure:

- top section for API base URL with a default such as `http://127.0.0.1:8080`
- compact health/readiness/state/results action bar
- start form with all allowed start fields and operator-friendly defaults
- alter form limited to `clients` and `target_tps`
- stop button clearly separated from mutating forms
- metrics actions:
  - fetch metrics into a `<pre>`
  - open `/metrics` directly in a new tab/window
- response area:
  - last request summary
  - formatted JSON panel
  - formatted metrics panel

Planned JavaScript behavior:

- normalize the base URL once so button handlers do not concatenate paths ad hoc
- build JSON payloads only from populated/meaningful form fields
- show validation hints in the page for the profile-specific fields:
  - `read_percent` only relevant for `mixed`
  - `transaction_mix` only relevant for `transaction`
  - alter form only sends fields that are actually set
- display server-side JSON errors as-is instead of swallowing them
- treat metrics as plain text, not JSON

Planned HTTP server behavior:

- support `OPTIONS` requests for the UI-relevant routes
- return the necessary `Access-Control-Allow-*` headers for browser access from a standalone page
- keep method enforcement for real endpoint methods intact
- avoid broad new router complexity or endpoint duplication

## Verification Strategy

This task is a TDD exception because the acceptance criteria are about a standalone browser artifact. The primary proof must come from a real browser or browser automation session against a running API, not from brittle tests that inspect HTML strings.

Execution should still keep verification honest in small slices:

- Slice 1: make the HTTP API browser-callable from a `file://` page
- Slice 2: create the standalone HTML page and prove it can load state/health/results
- Slice 3: prove start works from the page against a live local API
- Slice 4: prove alter works from the page against the same running benchmark
- Slice 5: prove stop works from the page
- Slice 6: prove metrics can be fetched into the page and/or opened directly
- Slice 7: run repo quality gates and do one final code-boundary pass

Planned verification during execution:

- `make check`
- `make lint`
- `make test`
- run the local API using the existing quickstart path
- open `examples/standalone-control-page.html` directly from disk in a real browser or a real browser-automation tool available on this machine
- record evidence that the page can:
  - view health/readiness
  - view state/results
  - start a benchmark
  - alter a benchmark
  - stop a benchmark
  - fetch or open metrics

Important verification rules:

- do not use `make test-long` because this is a normal task, not a story-end long-lane gate
- do not pretend curl-only proof satisfies the browser acceptance criterion
- if no real browser or honest browser automation path is available locally, switch back to `TO BE VERIFIED` instead of faking success

## Quality Gates

- `make check`
- `make lint`
- `make test`
- real browser verification against a running local API
- final `improve-code-boundaries` pass:
  - no Go template/static-serving path added
  - no duplicated CORS logic per handler
  - no duplicate JS DTO layer over the existing API
  - no swallowed browser or HTTP errors

If execution shows that the standalone page needs a muddier asset layout, a second backend route family, or template/static coupling to the Go server, switch this plan back to `TO BE VERIFIED` immediately instead of forcing the wrong boundary.

NOW EXECUTE
