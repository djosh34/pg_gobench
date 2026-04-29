## Task: 01 Add Standalone Raw HTML Control Page <status>not_started</status> <passes>false</passes>

<blocked_by>.ralph/tasks/story-08-release-docs/task-02-quickstart-docs.md</blocked_by>

<description>
**Goal:** Add the final standalone browser control page for `pg_gobench`. This must be an old-school raw HTML file that can be opened directly in a browser and calls the JSON API. It must not require server-side rendering, bundling, Node, a frontend framework, or coupling to Go templates.

The page should let a user set the API base URL, view benchmark state/results, start a benchmark, alter permitted runtime options, stop the benchmark, and open or fetch Prometheus metrics. Keep the page simple and operational rather than decorative. It must call the existing JSON API only.

This is the final final task for the project backlog. It is a standalone static browser artifact, so do not use TDD for this task. Verification must use a real browser or browser automation against a running API.
</description>

<acceptance_criteria>
- [ ] A single raw HTML file exists and can be opened directly from disk in a browser.
- [ ] The page allows configuring the API base URL.
- [ ] The page can call start, alter, stop, state/results, health/readiness, and metrics endpoints.
- [ ] The page is not served through Go templates and has no server-side coupling.
- [ ] The page requires no Node, bundler, frontend framework, or build step.
- [ ] Manual verification: open the file in a browser against a running local API and successfully view state, start a benchmark, alter it, stop it, and view metrics or a clear JSON error.
</acceptance_criteria>
