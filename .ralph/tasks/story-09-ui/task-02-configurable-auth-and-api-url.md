## Task: 02 Add Configurable Auth Token And API URL To HTML UI <status>done</status> <passes>true</passes>

<blocked_by>.ralph/tasks/story-09-ui/task-01-standalone-html-control-page.md</blocked_by>

<description>
Must use tdd skill to complete

**Goal:** Extend the standalone raw HTML control page so an operator can configure authentication and direct API requests to a non-default API endpoint from the page itself.

The HTML UI must allow the operator to enter a bearer token. After the token is set, the UI must hide the token value, avoid rendering the raw token back into the document, and still clearly show that a token has been set. Subsequent API requests from the page must include the configured token as an `Authorization: Bearer <token>` header. The page must also let the operator clear or replace the token without reloading the page.

The HTML UI must also allow the operator to alter the API URL/prefix used by all API requests so requests can be sent somewhere other than the default local API. The configured API base URL or prefix must be applied consistently to all existing controls, including health, readiness, benchmark state/results, start, alter, stop, and metrics actions. The UI should make the active API target visible and should validate or normalize common input mistakes without silently sending requests to the wrong endpoint.

This task is scoped to the standalone HTML control page behavior and the HTTP request behavior needed by that page. Do not add a frontend framework, bundler, server-side template, or separate UI asset pipeline. Do not implement unrelated UI redesigns.
</description>

<acceptance_criteria>
- [x] Red/green test or browser automation coverage proves that entering a bearer token marks the token as set while the raw token value is not visible in the rendered HTML.
- [x] Red/green test or browser automation coverage proves that API requests include `Authorization: Bearer <token>` after the token is configured.
- [x] Red/green test or browser automation coverage proves that clearing or replacing the bearer token changes subsequent request headers correctly.
- [x] Red/green test or browser automation coverage proves that changing the API URL/prefix changes the target for all API actions, including health, readiness, benchmark state/results, start, alter, stop, and metrics.
- [x] The standalone page still opens directly from disk in a browser and requires no Node, bundler, frontend framework, Go template, or server-side rendering.
- [x] Manual browser verification: open the standalone HTML file, set a bearer token, confirm the UI shows that a token is set without exposing it, change the API URL/prefix, and successfully send requests to the configured target or observe clear request errors from that target.
- [x] `make check` — passes cleanly
- [x] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [x] `make lint` — passes cleanly
- [x] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only, not required for this task)
</acceptance_criteria>

<plan>.ralph/tasks/story-09-ui/task-02-configurable-auth-and-api-url_plans/2026-04-30-configurable-auth-and-api-url-plan.md</plan>
