## Task: 03 Add Ultra-Simple JSON Benchmark API <status>not_started</status> <passes>false</passes>

<blocked_by>.ralph/tasks/story-02-control-plane/task-02-run-coordinator.md</blocked_by>

<description>
Must use tdd skill to complete

**Goal:** Expose the benchmark control plane over a deliberately small JSON HTTP API. The API must let users view, start, alter, and stop the single in-memory benchmark run. HTTP requests are plain JSON; there is no auth, no HTTPS, and no API key.

Add endpoints equivalent to:

- `POST /benchmark/start`
- `POST /benchmark/alter`
- `POST /benchmark/stop`
- `GET /benchmark`
- `GET /benchmark/results`
- `GET /healthz`
- `GET /readyz`

`/healthz` reports process/server health. `/readyz` reports database readiness and should include the Go error text in JSON when the database check fails. Benchmark state and error responses must be direct and compact; when an operation fails, print the Go error string in JSON rather than adding a large nested error taxonomy.

Unknown JSON fields must be rejected. Invalid methods and malformed JSON must return appropriate HTTP status codes. Do not add HTML UI in this task; the standalone HTML page is a final separate task and must not be coupled to server-side rendering.
</description>

<acceptance_criteria>
- [ ] TDD red/green HTTP handler coverage exists for start, alter, stop, state, results, health, and readiness endpoints.
- [ ] TDD red/green coverage exists for rejecting start while a benchmark is already running.
- [ ] TDD red/green coverage exists for malformed JSON, unknown JSON fields, invalid methods, and validation errors.
- [ ] JSON error responses include the actual Go error text in a compact field.
- [ ] No auth, HTTPS, sessions, cookies, or env-var config are introduced.
- [ ] The API remains separate from any future static HTML page.
- [ ] `make check` — passes cleanly
- [ ] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [ ] `make lint` — passes cleanly
- [ ] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>
