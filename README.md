# pg_gobench

`pg_gobench` is a PostgreSQL benchmark service with a small HTTP control plane. It reads one YAML config file, opens a database connection, exposes readiness and metrics endpoints, and lets you start, alter, stop, and inspect runs with `curl`.

This quick start is the canonical operator guide for local usage. The Compose and Kubernetes example directories provide runnable assets, but the configuration rules and HTTP API contract live here.

## Prerequisites

- `docker`
- `docker compose`
- `curl`

## Configuration Model

`pg_gobench` only supports file-based YAML configuration through `-config /path/to/pg_gobench.yaml`.

- `source.host`, `source.port`, and `source.dbname` are literal YAML values.
- `source.tls.ca_cert`, `source.tls.cert`, and `source.tls.key` are literal filesystem paths.
- Only `source.username` and `source.password` support credential modes.
- Connection strings are not supported.
- General environment-variable-driven config is not supported.
- `${VAR}` text in YAML is treated literally unless you explicitly use `env-ref` under `source.username` or `source.password`.

Complete example:

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

Credential modes for `source.username` and `source.password`:

Use a literal value:

```yaml
source:
  username:
    value: benchmark_user
  password:
    value: supersecret
```

Read from an environment variable by name:

```yaml
source:
  username:
    env-ref: POSTGRES_USERNAME
  password:
    env-ref: POSTGRES_PASSWORD
```

Read from a mounted secret file:

```yaml
source:
  username:
    secret-file: /run/secrets/postgres-username
  password:
    secret-file: /run/secrets/postgres-password
```

Each credential field must choose exactly one of `value`, `env-ref`, or `secret-file`.

## Docker Compose Quick Start

The repository already includes a runnable Compose example in [`examples/docker-compose-postgres`](examples/docker-compose-postgres/).

Start the stack with a fixed project name so the scratch-image section can reuse the same Docker network:

```bash
docker compose -f examples/docker-compose-postgres/compose.yaml -p pg-gobench-quickstart up -d --build
```

Check the service:

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
curl --fail http://127.0.0.1:8080/benchmark
curl --fail http://127.0.0.1:8080/benchmark/results
curl --fail http://127.0.0.1:8080/metrics
```

Expected high-level behavior:

- `/healthz` returns `{"status":"ok"}` when the HTTP server is alive.
- `/readyz` returns `{"status":"ok"}` when the database connection is ready.
- `/benchmark` returns the current run state such as `idle`, `running`, `stopping`, `stopped`, or `failed`.
- `/benchmark/results` returns the last known benchmark state plus accumulated stats.
- `/metrics` returns Prometheus text metrics.

## Run The Scratch Image Directly

The project `Dockerfile` builds a scratch-based runtime image. Build it locally:

```bash
docker build -t pg_gobench:local .
```

Run that image directly against the same Compose-managed PostgreSQL container:

```bash
docker run --rm \
  --name pg-gobench-scratch \
  --network pg-gobench-quickstart_default \
  -p 18080:8080 \
  -e POSTGRES_USERNAME=benchmark_user \
  -v "$PWD/examples/docker-compose-postgres/config/pg_gobench.yaml:/app/config/pg_gobench.yaml:ro" \
  -v "$PWD/examples/docker-compose-postgres/secrets/postgres-password.txt:/run/secrets/postgres-password:ro" \
  pg_gobench:local \
  -config /app/config/pg_gobench.yaml \
  -addr 0.0.0.0:8080
```

In another terminal:

```bash
curl --fail http://127.0.0.1:18080/healthz
curl --fail http://127.0.0.1:18080/readyz
curl --fail http://127.0.0.1:18080/benchmark
curl --fail http://127.0.0.1:18080/metrics
```

## Control A Benchmark With `curl`

Start a run:

```bash
curl --fail -X POST http://127.0.0.1:8080/benchmark/start \
  -H 'Content-Type: application/json' \
  -d '{"scale":1,"clients":2,"duration_seconds":30,"warmup_seconds":5,"reset":true,"profile":"mixed","read_percent":80}'
```

The start payload supports:

- `scale`
- `clients`
- `duration_seconds`
- `warmup_seconds`
- `reset`
- `profile`: `read`, `write`, `transaction`, `join`, `lock`, or `mixed`
- `read_percent` for `profile: "mixed"`
- `transaction_mix` for `profile: "transaction"`
- `target_tps`

Alter a running benchmark:

```bash
curl --fail -X POST http://127.0.0.1:8080/benchmark/alter \
  -H 'Content-Type: application/json' \
  -d '{"clients":4,"target_tps":200}'
```

`/benchmark/alter` requires at least one field and only accepts `clients` and `target_tps`.

Inspect state while the run is active:

```bash
curl --fail http://127.0.0.1:8080/benchmark
```

Stop the run:

```bash
curl --fail -X POST http://127.0.0.1:8080/benchmark/stop
```

Inspect the final result snapshot:

```bash
curl --fail http://127.0.0.1:8080/benchmark/results
```

## Read Results And Metrics

`/benchmark/results` returns benchmark state plus a `stats` object. Focus on these fields first:

- `stats.latency.p95_ms`: 95th percentile latency in milliseconds. Five percent of operations were slower than this.
- `stats.latency.p99_ms`: 99th percentile latency in milliseconds. This highlights the long tail.
- `stats.tps`: transactions or operations per second across the active workload.
- `stats.failed_operations`: operations that returned an error.
- `stats.active_clients`: clients currently doing work.
- `stats.total_operations`: all observed operations so far.
- `stats.successful_operations`: successful operations so you can compare success volume against failures.
- `stats.operation_rates`: per-operation throughput breakdown for point reads, range reads, joins, transactions, lock contention, and related workload pieces.

`/metrics` exposes Prometheus-compatible metrics including:

- `pg_gobench_run_active`
- `pg_gobench_run_duration_seconds`
- `pg_gobench_configured_clients`
- `pg_gobench_active_clients`
- `pg_gobench_operations_total`
- `pg_gobench_operation_errors_total`
- `pg_gobench_tps`
- `pg_gobench_operation_latency_seconds`

## Cleanup

Stop the Compose stack when you are done:

```bash
docker compose -f examples/docker-compose-postgres/compose.yaml -p pg-gobench-quickstart down -v
```
