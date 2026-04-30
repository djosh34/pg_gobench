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
- persist the token only in page runtime state or an explicit local browser storage choice if the implementation intentionally chooses persistence
- after setting the token, clear the input field and show a status such as `Bearer token set`
- never render the raw token value into status text, request logs, error panes, or the document body
- include `Authorization: Bearer <token>` on API requests only while a token is configured
- provide clear and replace actions

API target handling:

- keep a default API target compatible with the existing page behavior
- allow the operator to change the base API URL or prefix from the UI
- normalize common input cases such as trailing slashes
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

## Verification Strategy

Use the `tdd` skill during execution. Add the smallest reliable test surface that proves the browser-facing behavior without inspecting implementation details more than necessary.

Expected verification:

- first create failing coverage for token set/hidden behavior
- create failing coverage for authorization header behavior
- create failing coverage for token clear/replace behavior
- create failing coverage that every API action uses the configured API target
- implement the UI/request changes to make those tests pass
- manually open the standalone HTML file in a real browser or browser automation session and exercise the configured-token/configured-target workflow
- run `make check`
- run `make test`
- run `make lint`
- run `make test-long` only if the implementation changes ultra-long test selection or ultra-long behavior

If the existing project has no browser automation harness, the executor should add the minimal project-appropriate harness needed to prove these browser UI behaviors rather than skipping verification.
