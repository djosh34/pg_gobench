# Kubernetes Example

This example keeps the application boundary the same as the local Docker Compose setup:

- `pg_gobench` still reads its real YAML config from `/app/config/pg_gobench.yaml`
- database credentials come from mounted Kubernetes `Secret` files
- the app does not use Kubernetes-only env vars for host, port, dbname, or HTTP configuration

## Prerequisites

- a real local Kubernetes cluster such as kind, minikube, Docker Desktop Kubernetes, or k3d
- `kubectl`
- a locally built `pg_gobench:local` image imported into that cluster runtime

## Build And Load The Image

```bash
docker build -t pg_gobench:local .
```

For kind:

```bash
kind get clusters
kind load docker-image pg_gobench:local --name <kind-cluster-name>
```

For another local cluster runtime, use the equivalent local image import command before applying the manifests.

## Apply The Manifests

```bash
kubectl apply -f examples/k8s/
kubectl wait --namespace pg-gobench --for=condition=Available deployment/postgres --timeout=180s
kubectl wait --namespace pg-gobench --for=condition=Available deployment/pg-gobench --timeout=180s
```

## Port-Forward And Check The API

In one terminal:

```bash
kubectl port-forward --namespace pg-gobench svc/pg-gobench 8080:8080
```

In another terminal:

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
curl --fail http://127.0.0.1:8080/benchmark
curl --fail http://127.0.0.1:8080/metrics
```

## Start, Observe, And Stop A Benchmark

Start a small benchmark:

```bash
curl --fail -X POST http://127.0.0.1:8080/benchmark/start \
  -H 'Content-Type: application/json' \
  -d '{"scale":1,"duration_seconds":15,"warmup_seconds":1,"reset":true}'
```

Observe the run:

```bash
curl --fail http://127.0.0.1:8080/benchmark
curl --fail http://127.0.0.1:8080/metrics
```

Stop the run before it completes:

```bash
curl --fail -X POST http://127.0.0.1:8080/benchmark/stop
```

## Cleanup

```bash
kubectl delete -f examples/k8s/
```
