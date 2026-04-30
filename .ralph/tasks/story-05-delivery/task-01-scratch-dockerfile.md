## Task: 01 Add Scratch Dockerfile <status>done</status> <passes>true</passes>

<blocked_by>.ralph/tasks/story-04-observability/task-02-prometheus-metrics.md</blocked_by>

<description>
**Goal:** Add a production Dockerfile that builds `pg_gobench` as a static Go binary and ships it in a `scratch` final image. The final image must not contain a shell, package manager, source tree, or build tools.

The final image must include only what is needed to run the service, including CA certificates if needed for outbound TLS verification and a safe non-root runtime identity if practical. The binary must accept explicit command-line flags for config path and bind address. Do not use environment variables for application config in the image.

This is a non-code packaging task. Do not use TDD for this task. Verification must build and run the actual container image.
</description>

<acceptance_criteria>
 - [x] Dockerfile uses a multi-stage build and has `scratch` as the final stage.
 - [x] Final image contains the compiled service binary and required runtime CA material only.
 - [x] Final image runs without a shell.
 - [x] Manual verification: `docker build -t pg_gobench:local .` succeeds.
 - [x] Manual verification: running the image with an explicit config path and bind address starts the HTTP server or fails loudly with a clear config/database error.
 - [x] Manual verification: inspect the image or Dockerfile to confirm no source tree, package manager, or build tool is present in the final stage.
</acceptance_criteria>
