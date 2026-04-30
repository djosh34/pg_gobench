# Quick Start Documentation Plan

Plan file: `.ralph/tasks/story-08-release-docs/task-02-quickstart-docs_plans/2026-04-30-quickstart-docs-plan.md`

## Scope

Add one canonical quick start document for `pg_gobench` that shows:

- the file-based configuration model
- a complete YAML config example with all required source fields plus TLS path fields
- how username/password credentials can use `value`, `env-ref`, or `secret-file`
- how to start the stack with Docker Compose
- how to run the published scratch image directly with `docker run`
- how to control the benchmark with the HTTP JSON API
- how to observe readiness, state, results, and Prometheus metrics
- how to interpret core stats such as `p95`, `p99`, `TPS`, errors, active clients, and operation totals

This task is planning-only in this turn. Execution belongs to the next turn after the plan is promoted to `NOW EXECUTE`.

This task should deliver:

- one top-level quickstart document at `README.md`
- runnable command examples anchored to the existing binary, Dockerfile, and example Compose assets
- a clear statement that config is YAML-file based and that only username/password support explicit `env-ref`
- real manual verification of the documented commands in the next turn

This task should not deliver:

- fake tests that only assert documentation strings
- any new config mode such as connection strings or general env-var substitution
- a second competing quickstart doc under `docs/` or `examples/`
- `make test-long` unless this later becomes a story-finishing turn

## Public Interface

The documentation itself is the public interface change for this task.

Planned documentation surface:

- `README.md`

Planned runnable surfaces already present and to be documented honestly:

- config file flag: `-config /path/to/pg_gobench.yaml`
- listen address flag: `-addr 0.0.0.0:8080`
- JSON endpoints:
  - `GET /healthz`
  - `GET /readyz`
  - `GET /benchmark`
  - `GET /benchmark/results`
  - `POST /benchmark/start`
  - `POST /benchmark/alter`
  - `POST /benchmark/stop`
- Prometheus endpoint:
  - `GET /metrics`

Planned quickstart config template shape in the docs:

```yaml
source:
  host: postgres
  port: 5432
  dbname: pg_gobench
  username:
    env-ref: POSTGRES_USERNAME
  password:
    secret-file: /run/secrets/postgres-password
  tls:
    ca_cert: /run/certs/ca.pem
    cert: /run/certs/client.crt
    key: /run/certs/client.key
```

The README should also show concise alternate credential snippets for:

- `value`
- `env-ref`
- `secret-file`

The README should explicitly say:

- host, port, and dbname are literal YAML values
- TLS fields are literal filesystem paths, not secret modes
- `${VAR}` text inside YAML is treated literally unless used under username/password `env-ref`
- connection strings are not supported
- general environment-variable-driven config is not supported

## Boundary Decision

The `improve-code-boundaries` move for this docs task is to create one canonical quickstart at the repository root and keep the example assets as runnable inputs rather than duplicating the same API and config walkthrough in several places.

Planned boundary shape:

- `README.md` owns the canonical quickstart narrative
- `examples/docker-compose-postgres/compose.yaml` and `examples/docker-compose-postgres/config/pg_gobench.yaml` remain executable fixtures, not prose-heavy documentation
- `examples/k8s/README.md` remains Kubernetes-specific and should only be touched if execution shows it duplicates generic quickstart/API content badly enough to justify replacing repeated prose with a short pointer back to `README.md`

This avoids the main documentation smells:

- no duplicate benchmark API walkthroughs scattered across docs
- no second full config explanation in example directories
- no divergence between a "docs example" and the actual runnable Compose files
- no hidden operator assumptions about how the scratch image finds config or credentials

One concrete boundary simplification is planned:

- use an explicit Compose project name in the documented commands so the scratch-image section can refer to one deterministic Docker network instead of asking readers to discover ephemeral network names

If execution shows that the Compose-network approach makes the scratch-image instructions muddier than expected, switch this plan back to `TO BE VERIFIED` instead of inventing a second standalone environment.

## Planned Content Shape

Planned README sections:

- project overview in 2-3 sentences
- prerequisites:
  - `docker`
  - `docker compose`
  - `curl`
- config model:
  - complete YAML example with TLS paths
  - short explanation of `value`, `env-ref`, and `secret-file`
  - explicit non-support note for connection strings and generic env config
- Docker Compose quick start:
  - bring up the provided example stack with a fixed project name
  - confirm `/healthz`, `/readyz`, `/benchmark`, and `/metrics`
- scratch image quick start:
  - pull or use the published scratch image
  - mount the same config and secret files
  - join the same Docker network as the database container
  - expose port `8080`
- benchmark control examples:
  - start example with a valid `POST /benchmark/start` payload
  - alter example with `POST /benchmark/alter`
  - stop example with `POST /benchmark/stop`
  - inspect current state with `GET /benchmark`
  - inspect final results with `GET /benchmark/results`
- metrics and stats interpretation:
  - explain `p95_ms` and `p99_ms` as tail latency percentiles
  - explain `tps`
  - explain `failed_operations`, `active_clients`, and `total_operations`
  - point at Prometheus metric names such as `pg_gobench_tps`, `pg_gobench_active_clients`, and `pg_gobench_operation_latency_seconds`

Planned HTTP examples:

- `POST /benchmark/start` should use a real accepted payload such as:

```json
{"scale":1,"clients":2,"duration_seconds":30,"warmup_seconds":5,"reset":true,"profile":"mixed","read_percent":80}
```

- `POST /benchmark/alter` should use a real accepted payload such as:

```json
{"clients":4,"target_tps":200}
```

The README should mention that `alter` requires at least one field and only supports `clients` and `target_tps`.

## Verification Strategy

This task is a TDD exception because it is a documentation task. Verification must execute the documented commands rather than assert file text from tests.

Planned execution slices:

- [x] Slice 1: add `README.md` with the canonical quickstart content
- [x] Slice 2: if needed, make the smallest doc-only cleanup to reduce duplicate guidance in existing example READMEs
- [x] Slice 3: run `make check`
- [x] Slice 4: run `make lint`
- [x] Slice 5: run `make test`
- [x] Slice 6: execute the documented Docker Compose startup commands with the fixed project name
- [x] Slice 7: execute the documented `curl` commands against the Compose-started service and confirm JSON/metrics responses
- [x] Slice 8: execute the documented scratch-image run commands against the same database environment and confirm the health/readiness/API endpoints
- [x] Slice 9: if any documented command fails because of a real product or environment gap, create an add-bug task instead of hand-waving the failure
- [x] Slice 10: do one final `improve-code-boundaries` pass so the docs remain canonical and non-duplicated

Planned verification commands during execution:

```bash
make check
make lint
make test
docker compose -f examples/docker-compose-postgres/compose.yaml -p pg-gobench-quickstart up -d --build
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
curl --fail http://127.0.0.1:8080/benchmark
curl --fail -X POST http://127.0.0.1:8080/benchmark/start -H 'Content-Type: application/json' -d '{"scale":1,"clients":2,"duration_seconds":30,"warmup_seconds":5,"reset":true,"profile":"mixed","read_percent":80}'
curl --fail -X POST http://127.0.0.1:8080/benchmark/alter -H 'Content-Type: application/json' -d '{"clients":4,"target_tps":200}'
curl --fail -X POST http://127.0.0.1:8080/benchmark/stop
curl --fail http://127.0.0.1:8080/benchmark/results
curl --fail http://127.0.0.1:8080/metrics
docker build -t pg_gobench:local .
docker run --rm --name pg-gobench-scratch --network pg-gobench-quickstart_default -p 18080:8080 -e POSTGRES_USERNAME=benchmark_user -v "$PWD/examples/docker-compose-postgres/config/pg_gobench.yaml:/app/config/pg_gobench.yaml:ro" -v "$PWD/examples/docker-compose-postgres/secrets/postgres-password.txt:/run/secrets/postgres-password:ro" pg_gobench:local -config /app/config/pg_gobench.yaml -addr 0.0.0.0:8080
curl --fail http://127.0.0.1:18080/healthz
curl --fail http://127.0.0.1:18080/readyz
```

Planned execution notes:

- if the scratch-image docs use GHCR instead of a locally built tag, update the verification commands to pull and run the published image while keeping the same mounts and network
- if the Compose quickstart itself is the cleanest source of truth, do not fork its config into a second docs-only example file
- clean up the Compose stack after verification

## Implementation Notes

Expected file changes:

- add `README.md`
- possibly make a minimal edit to `examples/k8s/README.md` if that is the smallest honest way to prevent duplicated generic quickstart instructions

Avoid these muddy shortcuts:

- no separate `docs/quickstart.md` plus `README.md` with near-identical content
- no fake config examples that use unsupported env expansion outside `env-ref`
- no undocumented reliance on Docker defaults such as random project names when the scratch-image steps need a stable network
- no claims about metrics or JSON fields that do not match the actual public response types

If execution reveals a real gap, fix the smallest honest layer:

- docs wording issue -> fix `README.md`
- example-command mismatch -> fix the relevant example file or README command
- actual product behavior gap -> create an add-bug task immediately

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless this becomes the story-finishing turn or the task text changes
- real Docker Compose and scratch-container command execution are required for this task
- final docs boundary review is required so quickstart guidance stays canonical

If execution shows that the planned README location, deterministic network approach, or API/stat explanations create a muddier documentation boundary than expected, switch this plan back to `TO BE VERIFIED` instead of forcing duplicate docs.

NOW EXECUTE
