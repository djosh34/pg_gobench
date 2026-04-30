# Docker Compose PostgreSQL Example Plan

Plan file: `.ralph/tasks/story-05-delivery/task-02-docker-compose-postgres-example_plans/2026-04-30-docker-compose-postgres-example-plan.md`

## Scope

Add one runnable Docker Compose example that wires the existing `pg_gobench` scratch image to a PostgreSQL container using the real YAML config file format already owned by `internal/config`.

This turn is planning-only. Execution belongs to the next turn unless the design still needs to be reopened.

This task should deliver:

- one self-contained example directory for the compose stack rather than more root-level delivery clutter
- a mounted YAML app config file that sets `source.host`, `source.port`, and `source.dbname` as plain file values
- username and password demonstrated through the existing credential boundary only: `env-ref` and `secret-file`
- a Compose-defined PostgreSQL service, app service, network wiring, published HTTP port, and useful healthchecks
- real `docker compose` verification commands that bring the stack up and prove `/healthz` and `/readyz` behavior from the published port

This task should not deliver:

- any new application env-var bootstrap path
- wrapper scripts that translate env vars into flags or rewrite config files
- extra binaries in the scratch image just to satisfy an internal container healthcheck
- fake tests that assert YAML text instead of executing the stack

## Public Interface

Keep the example contract explicit and narrow:

- example entrypoint: `docker compose -f examples/docker-compose-postgres/compose.yaml up --build -d`
- app container command shape: `-config /app/config/pg_gobench.yaml -addr 0.0.0.0:8080`
- host verification endpoints:
  - `http://127.0.0.1:8080/healthz`
  - `http://127.0.0.1:8080/readyz`
- teardown: `docker compose -f examples/docker-compose-postgres/compose.yaml down -v`

The existing Go CLI and config model already expose the interface this task needs:

- `-config` remains the only application config path selector
- `-addr` remains the only bind-address selector
- YAML `source.username` already supports `env-ref`
- YAML `source.password` already supports `secret-file`

No application interface changes are planned unless execution proves the current runtime image or startup contract cannot support a clean compose example.

## Boundary Decision

Use the example directory as the delivery boundary and keep application bootstrap entirely file-based.

Planned shape:

- `examples/docker-compose-postgres/compose.yaml` owns service wiring, published ports, network, and startup ordering
- `examples/docker-compose-postgres/config/pg_gobench.yaml` owns the real application config shape
- `examples/docker-compose-postgres/secrets/postgres-password.txt` owns the demo password source for both containers
- the existing root `Dockerfile` remains the only image build contract

The `improve-code-boundaries` move for this task is to avoid muddy delivery boundaries:

- do not add root-level `compose.yaml`; keep this example isolated under `examples/`
- do not add env-driven app configuration; environment is allowed only for the explicit `source.username.env-ref` demonstration
- do not duplicate the password in both Compose environment and YAML; prefer one mounted secret file that PostgreSQL consumes through `_FILE` and `pg_gobench` consumes through `secret-file`
- do not bloat the scratch image with `curl`, `wget`, or a shell just to implement an in-container app healthcheck

One concrete boundary simplification is planned:

- demonstrate `env-ref` for `source.username`
- demonstrate `secret-file` for `source.password`

That keeps username/password sourcing at the config boundary while avoiding any second, container-specific config mechanism.

## Planned Example Shape

Planned directory layout:

- `examples/docker-compose-postgres/compose.yaml`
- `examples/docker-compose-postgres/config/pg_gobench.yaml`
- `examples/docker-compose-postgres/secrets/postgres-password.txt`

Planned Compose behavior:

- `postgres` service uses the official PostgreSQL image
- `pg_gobench` service builds from the repo root `Dockerfile`
- both services join the same default example network
- `postgres` gets a real healthcheck via `pg_isready`
- `pg_gobench` depends on PostgreSQL health before startup
- `pg_gobench` publishes `8080:8080`
- `pg_gobench` mounts the YAML config file read-only
- `pg_gobench` reads username from `POSTGRES_USERNAME` only because the YAML explicitly says `env-ref: POSTGRES_USERNAME`
- `pg_gobench` reads password from a mounted file path referenced by `secret-file`

Planned YAML shape:

```yaml
source:
  host: postgres
  port: 5432
  dbname: pg_gobench
  username:
    env-ref: POSTGRES_USERNAME
  password:
    secret-file: /run/secrets/postgres-password
```

This is intentionally manual and explicit. The example should teach the real app config model, not hide it behind Compose interpolation tricks.

## Verification Strategy

This task is a TDD exception because it is a packaging/example task, but execution should still follow tracer-bullet discipline with one real compose verification slice at a time.

Planned execution slices:

- [x] Slice 1: add the example files and bring the stack up with `docker compose ... up --build -d`
- [x] Slice 2: confirm PostgreSQL reaches healthy status through Compose
- [x] Slice 3: confirm `pg_gobench` is reachable on the published host port and `/healthz` returns OK JSON
- [x] Slice 4: confirm `/readyz` reaches success once PostgreSQL is actually accepting connections, or if startup timing is still in flight, returns the existing clear JSON Go error with readiness context
- [x] Slice 5: tear the stack down cleanly and then run `make check`, `make lint`, and `make test`
- [x] Slice 6: do one final `improve-code-boundaries` pass to ensure the example did not introduce a second config system or scatter delivery files across the repo root

Planned host-side verification commands during execution:

```bash
docker compose -f examples/docker-compose-postgres/compose.yaml up --build -d
docker compose -f examples/docker-compose-postgres/compose.yaml ps
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
docker compose -f examples/docker-compose-postgres/compose.yaml down -v
```

If `/readyz` races PostgreSQL startup, execution may use a short bounded retry loop from the host. The goal is a real ready service, not a brittle single-shot probe.

## Implementation Notes

Expected file changes:

- add the example directory and its tracked demo config/secret files
- update the task file with the plan pointer

No Go code changes are expected in the intended design. If execution reveals a real delivery-boundary gap, fix the smallest honest layer:

- Compose issue -> fix in the example files
- image/runtime artifact issue -> fix in `Dockerfile`
- application config/readiness issue -> only then consider Go code changes

Avoid these muddy shortcuts:

- no entrypoint wrapper scripts
- no env-var expansion inside the app beyond the existing `env-ref` feature
- no duplicate password value across multiple config channels
- no extra runtime debug tooling in the scratch image
- no alternate sample config format that differs from the real parser

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless this becomes the story-finishing turn or the task text changes to require it
- real `docker compose` startup and HTTP verification are required for this task

If execution shows that the example needs application-level behavior changes, or that the chosen file layout creates a muddier boundary than expected, switch this plan back to `TO BE VERIFIED` immediately instead of forcing a second bootstrap path.

NOW EXECUTE
