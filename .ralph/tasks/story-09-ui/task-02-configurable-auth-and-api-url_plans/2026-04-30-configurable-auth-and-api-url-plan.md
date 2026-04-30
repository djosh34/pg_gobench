# Configurable Auth Token And API URL Plan

Plan file: `.ralph/tasks/story-09-ui/task-02-configurable-auth-and-api-url_plans/2026-04-30-configurable-auth-and-api-url-plan.md`

## Scope

Extend the existing standalone raw HTML control page so operators can:

- enter, replace, and clear a bearer token
- see that a bearer token is configured without seeing the token value
- send all API requests with `Authorization: Bearer <token>` when configured
- configure the API URL or prefix used by all page requests
- see which API target is currently active

This task should not introduce:

- a frontend framework
- Node, npm, a bundler, or generated frontend assets
- Go templates or server-side UI rendering
- multiple UI asset files when the existing task requires a standalone HTML artifact
- unrelated visual redesign work

## Planned Behavior

Bearer token handling:

- provide a password-style input for entering a token
- keep the bearer token only in page runtime state, not in localStorage and not in any rendered status text
- after setting the token, clear the input field and show a status such as `Bearer token set`
- never render the raw token value into status text, request logs, error panes, or the document body
- include `Authorization: Bearer <token>` on API requests only while a token is configured
- provide clear and replace actions

API target handling:

- keep a default API target compatible with the existing page behavior
- allow the operator to change the base API URL or path-prefixed target from the UI using one canonical base value
- normalize common input cases such as missing `http://` for host:port input and trailing slashes
- reject invalid targets, query strings, fragments, and relative `file://`-style mistakes instead of silently guessing
- apply the configured target consistently to every API action:
  - health
  - readiness
  - benchmark state
  - benchmark results
  - benchmark start
  - benchmark alter
  - benchmark stop
- metrics fetch/open
- display the active target so the operator can see where requests are being sent

## Boundary Cleanup

Use `improve-code-boundaries` during execution by flattening the current mixed-responsibility script instead of layering more ad hoc handlers onto it.

Concrete boundary change to make:

- keep the page as one standalone HTML file, but split the script into small single-purpose pieces inside that file:
  - request configuration state for `apiBaseUrl` and `bearerToken`
  - target normalization and endpoint URL construction
  - request execution that owns header construction and fetch decoding
  - UI rendering/status updates
  - action wiring through a single route table for all endpoint-triggering controls
- remove duplicated per-button request assembly so every API action flows through one request boundary
- ensure the token boundary is runtime-only so the secret cannot leak through persistence/rendering paths by accident

## Public Interface Decisions

The HTML page should expose these operator-facing controls:

- API target input with save/apply action and visible active-target status
- bearer token password input with set/replace action
- separate clear-token action
- token status text that distinguishes `not set` from `set` without exposing the token

Internal execution decisions:

- keep API target persistence in `localStorage` because that is already part of page behavior and is safe for non-secret configuration
- do not persist bearer tokens in `localStorage`
- centralize endpoint paths in one map/table so all existing actions are forced through the same base-target logic

## Verification Strategy

Use the `tdd` skill during execution. Add the smallest reliable test surface that proves the browser-facing behavior without inspecting implementation details more than necessary.

Expected verification:

- first create failing coverage for token set/hidden behavior
- then create failing coverage for authorization header behavior
- then create failing coverage for token clear/replace behavior
- then create failing coverage that every API action uses the configured API target
- implement the UI/request changes to make those tests pass
- manually open the standalone HTML file in a real browser or browser automation session and exercise the configured-token/configured-target workflow
- run `make check`
- run `make test`
- run `make lint`
- run `make test-long` only if the implementation changes ultra-long test selection or ultra-long behavior

Test harness choice for this repository:

- `make test` is `go test ./...`, so the coverage must run from Go without a Node pipeline
- add a Go test that loads the standalone HTML source and exercises the page script through a minimal JS/DOM harness
- the harness should stub only the browser surface the page actually uses:
  - `document.getElementById`
  - element properties like `value`, `textContent`, `checked`, `disabled`, and `dataset`
  - event listeners for button clicks and form submits
  - `window.localStorage`
  - `window.fetch`
  - `window.open`
- the tests should assert observable behavior through DOM text/status and recorded fetch/open calls, not private helper names
- manual browser verification is still required at the end; if no browser is available for that step, do not claim the task complete

## Execution Order

1. Add the first failing Go integration-style test for token set/hidden behavior.
2. Implement the smallest page-state/token rendering change to pass it.
3. Add the next failing test for the authorization header.
4. Implement the centralized request-config/header path to pass it.
5. Add the failing clear/replace token behavior test.
6. Implement token replacement/clearing through the same request boundary.
7. Add the failing configured-target coverage across every API action, including metrics open/fetch.
8. Refactor the script into the planned state/URL/request/UI boundaries while keeping the standalone single-file artifact.
9. Run manual browser verification.
10. Run `make check`, `make test`, and `make lint`.

NOW EXECUTE
