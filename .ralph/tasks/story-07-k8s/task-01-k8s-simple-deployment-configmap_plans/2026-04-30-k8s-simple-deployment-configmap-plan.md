# Kubernetes Simple Deployment And ConfigMap Plan

Plan file: `.ralph/tasks/story-07-k8s/task-01-k8s-simple-deployment-configmap_plans/2026-04-30-k8s-simple-deployment-configmap-plan.md`

## Scope

Add one runnable Kubernetes example that deploys the existing scratch `pg_gobench` image plus a local-cluster PostgreSQL dependency using the real YAML config file format already owned by `internal/config`.

This turn is planning-only. Execution belongs to the next turn unless the design must be reopened.

This task should deliver:

- one self-contained `examples/k8s/` directory that can be applied with `kubectl apply -f examples/k8s/`
- a `ConfigMap` containing the real `pg_gobench` YAML config file
- a `Secret` mounted into the application pod and referenced through config-supported credential mechanisms, with `secret-file` preferred
- a simple PostgreSQL manifest suitable for a real local cluster
- a `Deployment` for `pg_gobench` and a `Service` exposing its HTTP API inside the cluster
- minimal operator instructions for image build/load, apply, readiness checks, port-forwarding, endpoint checks, and benchmark start/observe/stop
- real manual verification against a local Kubernetes cluster

This task should not deliver:

- any new application bootstrap path besides `-config` and the existing YAML config model
- app-wide env-var configuration for host, port, dbname, HTTP auth, or HTTPS
- Helm charts, templating layers, wrapper scripts, or kustomize overlays
- fake tests that only assert YAML text instead of exercising a real cluster
- `make test-long` unless execution unexpectedly becomes a full-story validation turn

## Public Interface

Keep the example contract explicit and narrow:

- image build: `docker build -t pg_gobench:local .`
- image load into the chosen local cluster runtime before apply:
  - `kind load docker-image pg_gobench:local`
  - or the equivalent local-cluster import command when not using kind
- cluster apply: `kubectl apply -f examples/k8s/`
- app command shape in the deployment:
  - `/pg_gobench -config /app/config/pg_gobench.yaml -addr 0.0.0.0:8080`
- cluster-local service name:
  - `pg-gobench`
- host verification entrypoint:
  - `kubectl port-forward -n pg-gobench svc/pg-gobench 8080:8080`
- required host-side checks after port-forward:
  - `GET /healthz`
  - `GET /readyz`
  - `GET /benchmark`
  - `GET /metrics`
  - `POST /benchmark/start`
  - `POST /benchmark/stop`

No Go CLI or config-schema changes are planned. The example should teach the current runtime contract, not alter it.

## Boundary Decision

The `improve-code-boundaries` move for this task is to keep deployment wiring and application configuration in their proper layers instead of inventing Kubernetes-only configuration paths.

Planned shape:

- `examples/k8s/` owns Kubernetes objects only
- `examples/k8s/config/pg_gobench.yaml` content lives inside a Kubernetes `ConfigMap`
- one Kubernetes `Secret` owns the credential material mounted into both PostgreSQL and `pg_gobench`
- the root `Dockerfile` remains the only image build contract
- the app still learns database settings from the YAML config file, not from deployment env vars

This removes the main muddy boundary risk:

- do not duplicate `source.host`, `source.port`, `source.dbname`, or HTTP listen config across Deployment env vars and YAML
- do not add a second bootstrap mechanism just because the runtime target is Kubernetes
- do not spread credential translation into wrapper scripts
- do not create a parallel root-level deployment layout when `examples/` already holds delivery examples

One concrete boundary simplification is planned:

- prefer `secret-file` for `source.password`
- prefer `secret-file` for `source.username` as well if the mounted-secret layout stays simple and readable
- only fall back to `username.value` if execution proves that a second username secret file makes the example materially worse

If execution shows that Kubernetes image distribution or credential mounting forces a second app config path, switch this plan back to `TO BE VERIFIED` immediately instead of forcing a muddy bootstrap boundary.

## Planned Example Shape

Planned directory layout:

- `examples/k8s/namespace.yaml`
- `examples/k8s/secret.yaml`
- `examples/k8s/configmap.yaml`
- `examples/k8s/postgres.yaml`
- `examples/k8s/pg-gobench.yaml`
- `examples/k8s/README.md`

Planned Kubernetes objects:

- one namespace, likely `pg-gobench`
- one opaque secret containing at least:
  - `postgres-username`
  - `postgres-password`
- one config map containing the real YAML config file
- one PostgreSQL deployment and service for in-cluster access
- one `pg_gobench` deployment and service for the HTTP API

Planned PostgreSQL direction:

- use an official PostgreSQL image already suitable for local-cluster pulls
- keep storage simple and ephemeral for the example unless execution proves a PVC is required
- use built-in PostgreSQL bootstrap env vars only for the database container's own initialization boundary
- mount the shared secret so PostgreSQL can consume the password from file if the image contract allows it cleanly

Planned application YAML shape:

```yaml
source:
  host: postgres
  port: 5432
  dbname: pg_gobench
  username:
    secret-file: /run/secrets/postgres-username
  password:
    secret-file: /run/secrets/postgres-password
```

Planned application deployment behavior:

- mount the config map read-only at `/app/config/pg_gobench.yaml`
- mount the secret read-only at `/run/secrets/`
- set the container command explicitly because the scratch image entrypoint is only `/pg_gobench`
- expose container port `8080`
- set `imagePullPolicy` for the local-image workflow so the example works with a locally loaded image instead of demanding a registry push

Planned service behavior:

- cluster-internal `ClusterIP` service on port `8080`
- docs use `kubectl port-forward` rather than a NodePort or LoadBalancer

This is intentionally manual and explicit. The example should teach the real config and runtime boundary, not hide it behind templating.

## Verification Strategy

This task is a TDD exception because it is a deployment/example task, but execution should still move in small real slices with cluster-backed verification after each major change.

Planned execution slices:

- [ ] Slice 1: add the Kubernetes example files and README, then build the local image
- [ ] Slice 2: load the image into the chosen local cluster runtime and apply `examples/k8s/` with one `kubectl apply` command
- [ ] Slice 3: wait for PostgreSQL and `pg_gobench` pods to become ready and inspect failures honestly if readiness does not converge
- [ ] Slice 4: port-forward the `pg-gobench` service and confirm `/healthz`, `/readyz`, `/benchmark`, and `/metrics`
- [ ] Slice 5: start at least one benchmark through `POST /benchmark/start`, observe state through `GET /benchmark`, then stop it through `POST /benchmark/stop`
- [ ] Slice 6: if benchmark execution fails, immediately create an add-bug task instead of hiding or hand-waving the error
- [ ] Slice 7: tear down any local verification resources as appropriate, then run `make check`, `make lint`, and `make test`
- [ ] Slice 8: do one final `improve-code-boundaries` pass to ensure the example did not introduce a second app config system or duplicate delivery layout

Planned host-side verification commands during execution:

```bash
docker build -t pg_gobench:local .
kind load docker-image pg_gobench:local
kubectl apply -f examples/k8s/
kubectl wait --namespace pg-gobench --for=condition=Available deployment/postgres --timeout=180s
kubectl wait --namespace pg-gobench --for=condition=Available deployment/pg-gobench --timeout=180s
kubectl port-forward -n pg-gobench svc/pg-gobench 8080:8080
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
curl --fail http://127.0.0.1:8080/benchmark
curl --fail http://127.0.0.1:8080/metrics
curl --fail -X POST http://127.0.0.1:8080/benchmark/start \
  -H 'Content-Type: application/json' \
  -d '{"scale":1,"clients":1,"duration_seconds":15,"warmup_seconds":1,"reset":true}'
curl --fail http://127.0.0.1:8080/benchmark
curl --fail -X POST http://127.0.0.1:8080/benchmark/stop
```

If the local cluster is not `kind`, execution can swap only the image-load step for the equivalent local-cluster command. The `kubectl apply` and runtime verification flow should stay unchanged.

## Implementation Notes

Expected file changes:

- add the new `examples/k8s/` manifest set and README
- possibly adjust `.gitignore` only if execution introduces cluster-local artifacts that should not be tracked
- no Go code changes are expected in the intended design

Avoid these muddy shortcuts:

- no env-driven app config for non-credential fields
- no wrapper entrypoint scripts that rewrite config files
- no second sample config format distinct from the real parser
- no root-level Kubernetes manifest clutter
- no silent readiness or benchmark failures

If execution reveals a real gap, fix the smallest honest layer:

- manifest issue -> fix in `examples/k8s/`
- image/runtime issue -> fix in `Dockerfile`
- real application behavior gap -> only then consider Go code changes

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless this becomes the story-finishing turn or the task text changes
- real local-cluster `kubectl apply` plus endpoint and benchmark verification are required for this task

If execution shows that the selected manifest layout or credential path creates a muddier boundary than expected, switch this plan back to `TO BE VERIFIED` instead of forcing a second bootstrap path.

NOW EXECUTE
