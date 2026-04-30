# Manual Verify Everything End-To-End Plan

Plan file: `.ralph/tasks/story-99-manual-verify-everything/task-01-manual-verify-everything_plans/2026-04-30-manual-verify-everything-plan.md`

## Scope

Perform the final human verification pass for the whole shipped product by exercising the real runnable surfaces end to end:

- local Go service behavior
- YAML config loading and rejection behavior
- PostgreSQL readiness and benchmark schema setup
- JSON benchmark control API
- core and advanced workloads
- stats and Prometheus metrics output
- scratch container image
- Docker Compose example
- Kubernetes example
- GitHub Actions / GHCR release behavior where locally observable
- quick start docs
- standalone raw HTML control page opened from disk

This turn is planning-only. Execution belongs to the next turn after this plan is reviewed and promoted to `NOW EXECUTE`.

This task should deliver:

- one complete manual verification pass across every checklist item already recorded in the task file
- real evidence gathered from live commands, live containers, live Kubernetes resources, live HTTP responses, and a real browser session for the standalone HTML page
- immediate bug creation for any real failure found during the pass
- early task switching to the bug if a failure is found before the rest of the matrix is hand-waved through

This task should not deliver:

- `make check`
- `make lint`
- `make test`
- `make test-long`
- fake verification by reading files and assuming behavior
- partial completion marked as passing

The task description is explicit: this is a non-code manual verification task, so the normal TDD workflow is intentionally not used here.

## Public Verification Surface

Existing artifacts and surfaces to verify:

- binary entrypoint:
  - `cmd/pg_gobench/main.go`
- config and HTTP server implementation:
  - `internal/config/config.go`
  - `internal/httpserver/server.go`
- container/runtime assets:
  - `Dockerfile`
  - `examples/docker-compose-postgres/compose.yaml`
  - `examples/docker-compose-postgres/config/pg_gobench.yaml`
  - `examples/docker-compose-postgres/secrets/postgres-password.txt`
- Kubernetes assets:
  - `examples/k8s/00-namespace.yaml`
  - `examples/k8s/10-secret.yaml`
  - `examples/k8s/20-configmap.yaml`
  - `examples/k8s/30-postgres.yaml`
  - `examples/k8s/40-pg-gobench.yaml`
  - `examples/k8s/README.md`
- UI/docs/release assets:
  - `examples/standalone-control-page.html`
  - `README.md`
  - `.github/workflows/publish-ghcr.yml`

HTTP surfaces that must be exercised against a live service:

- `GET /healthz`
- `GET /readyz`
- `GET /benchmark`
- `GET /benchmark/results`
- `POST /benchmark/start`
- `POST /benchmark/alter`
- `POST /benchmark/stop`
- `GET /metrics`

## Boundary Decision

The `improve-code-boundaries` move for this task is to keep the manual pass anchored to the product's real operator boundaries rather than inventing a second verification harness:

- Docker Compose is the canonical local environment for PostgreSQL-backed verification
- the scratch image is verified against that same real database environment rather than a separate mock setup
- Kubernetes is verified from the committed manifests directly, not from ad-hoc patched YAML
- the standalone HTML page is verified as a raw `file://` artifact against the same live API
- documentation and workflow claims are checked against the exact committed artifacts, not paraphrased summaries

That keeps this task from becoming muddy in two common ways:

- no duplicate one-off scripts that create a different environment than the examples users will actually run
- no split between "manual verification behavior" and the real committed operator surfaces

If execution shows that one of the committed delivery surfaces cannot be verified honestly without inventing a parallel setup, stop and create a bug instead of forcing an alternative path.

## Planned Execution Order

Follow the checklist in the task file, but execute in this order so failures surface early and reuse the same environment where possible:

1. Repository artifact read-through
   - re-open `README.md`, Compose assets, Kubernetes assets, standalone HTML, and workflow file
   - extract the exact commands and behaviors that will be verified
   - confirm whether any acceptance item is only locally observable through GitHub API / registry inspection rather than execution

2. Compose-backed live service
   - start `examples/docker-compose-postgres/compose.yaml`
   - verify health and readiness
   - verify the committed YAML config works with:
     - username `env-ref`
     - password `secret-file`
   - create minimal alternate config files only if needed for manual proof of:
     - username/password `value`
     - password `env-ref`
     - strict rejection cases
     - TLS path validation failures
   - verify `/benchmark/start`, `/benchmark/alter`, `/benchmark/stop`, `/benchmark`, `/benchmark/results`, and compact error JSON
   - verify single-active-run rejection
   - verify schema setup, reset behavior, scale initialization, core workloads, advanced workloads, surfaced contention errors, stats fields, and Prometheus metrics names

3. Standalone HTML page
   - open `examples/standalone-control-page.html` directly from disk in a real browser
   - point it at the running Compose-backed API
   - verify health/readiness/state/results/start/alter/stop/metrics through the page itself

4. Scratch container image
   - build or pull the scratch image as needed from the committed `Dockerfile`
   - run it against the same PostgreSQL environment with explicit flags and mounted config/secrets
   - verify health, readiness, and benchmark control behavior from the containerized process

5. Kubernetes deployment
   - apply the committed manifests from `examples/k8s/`
   - wait for PostgreSQL and `pg_gobench` readiness
   - verify the service from inside or via port-forward using the same HTTP/API checks
   - confirm ConfigMap/Secret-backed config actually drives the live pod

6. Release/docs verification
   - inspect `README.md` accuracy against the commands already executed
   - inspect `.github/workflows/publish-ghcr.yml` behavior where locally observable
   - if possible, verify the latest published GHCR manifest / workflow run evidence from the current committed workflow shape

7. Completion or interruption
   - if any real bug is found at any stage:
     - create an add-bug task immediately with concrete reproduction
     - stop further unrelated verification
     - run `/bin/bash .ralph/task_switch.sh`
     - leave this task incomplete
   - if the whole matrix passes:
     - check off the checklist and acceptance boxes
     - set `<passes>true</passes>`
     - run `/bin/bash .ralph/task_switch.sh`
     - commit all repo changes and push

## Verification Matrix

Manual slices to execute next turn:

- [x] Slice 1: use the committed task checklist as the canonical verification matrix
- [x] Slice 2: verify the Compose example as the main reusable runtime environment
- [x] Slice 3: prove config loading success paths and rejection paths against the real service
- [x] Slice 4: prove HTTP control-plane behavior and error handling against the live API
- [x] Slice 5: prove schema setup, scale behavior, core workloads, advanced workloads, and visible contention errors
- [x] Slice 6: prove stats shape and Prometheus metric names from live responses
- [x] Slice 7: prove the standalone HTML page works from disk against the live API
- [x] Slice 8: prove the scratch image works with explicit flags and mounted config/secrets
- [x] Slice 9: prove the Kubernetes manifests work as committed
- [x] Slice 10: verify docs and release claims only to the extent they are honestly locally observable
- [x] Slice 11: if any failure appears, create a bug immediately and switch tasks before continuing
- [x] Slice 12: only if every slice passes, mark the task passing and perform the required Ralph closeout steps

## Planned Commands

Expected command families during execution:

```bash
docker compose -f examples/docker-compose-postgres/compose.yaml up -d --build
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
curl --fail http://127.0.0.1:8080/benchmark
curl --fail http://127.0.0.1:8080/benchmark/results
curl --fail -X POST http://127.0.0.1:8080/benchmark/start -H 'Content-Type: application/json' -d '...'
curl --fail -X POST http://127.0.0.1:8080/benchmark/alter -H 'Content-Type: application/json' -d '...'
curl --fail -X POST http://127.0.0.1:8080/benchmark/stop
curl --fail http://127.0.0.1:8080/metrics
docker build -t pg_gobench:manual .
docker run ... pg_gobench:manual -config /app/config/pg_gobench.yaml -addr 0.0.0.0:8080
kubectl apply -f examples/k8s/
kubectl get pods -n pg-gobench
kubectl port-forward -n pg-gobench svc/pg-gobench 18080:8080
```

Browser/manual tools expected during execution:

- a real local browser for `examples/standalone-control-page.html`
- Docker and Docker Compose
- Kubernetes tooling available on this machine
- GitHub API / registry inspection only if credentials and network allow honest observation

Do not run automated repo gates during execution for this task unless the task text itself changes. This task is manual-product verification only.

## Interruption Rules

Switch this plan back to `TO BE VERIFIED` immediately if any of these happen during execution:

- the committed examples are too incomplete to define a single honest verification path
- the browser verification cannot be performed with a real local browser and no truthful fallback exists
- the Kubernetes or GHCR verification requires a materially different interface or asset layout than the committed examples claim
- proving a checklist item requires code or doc changes large enough that the bug should be handled as a separate task first

If a real product defect is found, do not keep verifying unrelated areas. Create the bug task, switch tasks, and stop.

NOW EXECUTE
