# Scratch Dockerfile Plan

Plan file: `.ralph/tasks/story-05-delivery/task-01-scratch-dockerfile_plans/2026-04-30-scratch-dockerfile-plan.md`

## Scope

Add a production container build for `pg_gobench` that compiles a static Go binary in a builder stage and ships only the runtime binary plus required CA certificate material in a `scratch` final image.

This turn is planning-only. Execution belongs to the next turn after the plan is reviewed and explicitly switched to `NOW EXECUTE`.

This task should deliver:

- a multi-stage `Dockerfile` with `scratch` as the final stage
- a direct binary entrypoint with no shell wrapper
- a non-root runtime identity in the final image if the binary can run that way cleanly
- executable verification using real `docker build` and `docker run` commands

This task should not deliver:

- environment-variable based app bootstrap in the image
- shell scripts, package managers, or debug tools in the runtime image
- brittle tests that assert Dockerfile text instead of verifying container behavior

## Public Interface

Keep the container contract minimal and explicit:

- build command: `docker build -t pg_gobench:local .`
- runtime command shape: `docker run --rm pg_gobench:local -config /path/in/container/config.yaml -addr 0.0.0.0:8080`
- entrypoint: the final image should invoke the compiled `pg_gobench` binary directly
- config delivery: mount a config file into the container at runtime; do not bake app config into the image and do not translate env vars into flags

The existing Go CLI already supports the required flags:

- `-config` for the YAML path
- `-addr` for the HTTP bind address

No application interface changes are planned unless execution proves the current binary cannot be built statically or the runtime output is too unclear to satisfy the acceptance criteria.

## Boundary Decision

Use the image boundary itself as the packaging contract instead of adding wrapper code around the binary.

Planned shape:

- `Dockerfile` owns build-stage concerns such as Go toolchain usage and static compilation flags
- the final `scratch` stage owns only runtime artifacts: the binary and CA bundle
- `cmd/pg_gobench` keeps owning CLI parsing and runtime error messages
- `docker run ... -config ... -addr ...` remains the only runtime bootstrap path

The `improve-code-boundaries` move for this task is to avoid muddy bootstrap boundaries:

- no shell entrypoint translating env vars into flags
- no duplicate config source just for containers
- no helper script that decides defaults outside the Go binary
- no packaging knowledge pushed into Go code unless RED proves the binary itself is missing a real runtime requirement

If execution reveals a genuine boundary smell, the most likely one is a misplaced runtime concern such as certificates, user identity, or file paths being handled in the application instead of the image. Fix that at the container boundary first.

## Verification Strategy

This task is a TDD exception because it is a packaging task. Verification must execute the real image.

Planned verification slices:

- [x] Slice 1: build the image with `docker build -t pg_gobench:local .` and confirm the final stage is `scratch`
- [x] Slice 2: inspect the final image metadata with a real container/image inspection command to confirm the entrypoint is the binary and there is no shell
- [x] Slice 3: run the image with a mounted minimal YAML config and explicit `-config` and `-addr` flags
- [x] Slice 4: confirm runtime behavior is acceptable:
  - either the process starts with the explicit bind address
  - or it exits loudly with a clear config or database error such as config load failure or database connection failure
- [x] Slice 5: inspect the Dockerfile and/or image contents through container tooling to confirm the final image does not contain source, package manager files, or build tools
- [x] Slice 6: after green, run repo quality gates and do one final `improve-code-boundaries` pass to ensure no bootstrap mud was introduced

Planned runtime fixture:

- create a temporary local YAML file with a minimal valid `source` section using inline `value` credentials
- mount it read-only into the container
- prefer a database target that will fail loudly and quickly if no disposable Postgres is available, since the task acceptance explicitly allows a clear config/database failure

Example intended command shape during execution:

```bash
docker run --rm \
  -v "$PWD/.tmp/pg_gobench-container-config.yaml:/config.yaml:ro" \
  pg_gobench:local \
  -config /config.yaml \
  -addr 0.0.0.0:8080
```

## Implementation Notes

Expected packaging changes:

- add `Dockerfile`
- adjust `.dockerignore` only if execution shows the build context is carrying irrelevant material

Planned Dockerfile shape:

- builder stage based on a Go image
- static Linux build with `CGO_ENABLED=0`
- copy only the compiled binary and CA certificate bundle into the final stage
- set a numeric `USER` in the final stage if file ownership and runtime behavior allow it cleanly
- use JSON-array `ENTRYPOINT` so the binary receives flags directly

Avoid these muddy shortcuts:

- do not install a shell in the final image
- do not use `sh -c`
- do not add an env-driven wrapper
- do not copy the entire repo into the final stage
- do not add backwards-compatibility config paths or fallback env vars

## File Plan

- `Dockerfile`
- `.dockerignore` only if needed
- `.ralph/tasks/story-05-delivery/task-01-scratch-dockerfile_plans/2026-04-30-scratch-dockerfile-plan.md`

No Go code changes are expected in the intended design.

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless the task text is later changed to require the long lane or this task becomes the story-finishing validation turn
- real `docker build` and `docker run` verification are required for this task even though it is not a code-TDD task

If execution shows the current plan is wrong, especially if static linking, runtime certificates, or non-root execution require application-level changes, switch the plan back to `TO BE VERIFIED` immediately instead of forcing a muddy container wrapper.

NOW EXECUTE
