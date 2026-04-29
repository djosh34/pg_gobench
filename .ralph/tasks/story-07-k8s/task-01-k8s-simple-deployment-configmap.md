## Task: 01 Add Ultra-Simple Kubernetes Deployment And ConfigMap <status>not_started</status> <passes>false</passes>

<blocked_by>.ralph/tasks/story-06-advanced-workloads/task-01-join-lock-contention-workloads.md</blocked_by>

<description>
**Goal:** Add an ultra-simple Kubernetes example that deploys `pg_gobench` and its config with one simple `kubectl apply` against a real local Kubernetes cluster. The example must be operational, not theoretical.

The Kubernetes example must include a PostgreSQL dependency or a clearly included local-cluster PostgreSQL manifest, a ConfigMap containing the `pg_gobench` YAML config, any required Secret or mounted secret-file data for username/password, a Deployment for the scratch `pg_gobench` image, and a Service exposing the HTTP API inside the cluster. The application config must still come from the YAML config file. Environment variables may only be used when explicitly referenced by username/password `env-ref`; prefer demonstrating `secret-file` through a mounted Kubernetes Secret where practical.

The user must be able to apply the example with a single command such as `kubectl apply -f examples/k8s/` or one manifest file. Include minimal instructions for port-forwarding and for checking `/healthz`, `/readyz`, `/benchmark`, and `/metrics`.

This is a non-code deployment task. Do not use TDD for this task. Verification must run against a real local Kubernetes cluster, such as kind, minikube, Docker Desktop Kubernetes, k3d, or an already available local cluster.
</description>

<acceptance_criteria>
- [ ] Kubernetes manifests can be applied with one simple `kubectl apply` command.
- [ ] Manifests include PostgreSQL or a clearly usable local-cluster PostgreSQL dependency.
- [ ] Manifests include a ConfigMap containing the real `pg_gobench` YAML config.
- [ ] Manifests include username/password handling through Kubernetes Secret material mounted or referenced only by the config-supported username/password mechanisms.
- [ ] Manifests include a Deployment for the scratch `pg_gobench` image and a Service for its HTTP API.
- [ ] Manifests do not introduce app-wide env-var configuration, HTTP auth, or HTTPS.
- [ ] Manual verification: apply the manifests to a real local Kubernetes cluster with the documented single `kubectl apply` command.
- [ ] Manual verification: wait for PostgreSQL and `pg_gobench` pods to become ready.
- [ ] Manual verification: port-forward the service and call `/healthz`, `/readyz`, `/benchmark`, and `/metrics`.
- [ ] Manual verification: start, observe, and stop at least one benchmark through the Kubernetes-deployed service, or immediately create an add-bug task for any failure.
</acceptance_criteria>
