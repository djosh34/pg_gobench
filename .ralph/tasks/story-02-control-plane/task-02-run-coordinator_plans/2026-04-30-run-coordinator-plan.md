# Run Coordinator Plan

## Scope

Implement the single-active-run in-memory coordinator as its own package, with explicit run state and a small worker contract. This task is about orchestration only:

- no HTTP handlers yet
- no database workload implementation yet
- no persisted history or on-disk results
- no extra DTO layer between the coordinator and the later JSON API

Execution should leave the coordinator ready for the next HTTP task and for the later workload/stats tasks.

## Public Interface

Create a new package, `internal/benchmarkrun`, as the owner of live benchmark lifecycle state.

Planned exported surface:

- `type Status string`
- `const (StatusIdle, StatusStarting, StatusRunning, StatusStopping, StatusStopped, StatusFailed)`
- `type State struct`
- `type Runner interface`
- `type Run interface`
- `type Coordinator struct`
- `func New(runner Runner, opts ...Option) *Coordinator`
- `func (c *Coordinator) Start(ctx context.Context, options benchmark.StartOptions) (State, error)`
- `func (c *Coordinator) Alter(options benchmark.AlterOptions) (State, error)`
- `func (c *Coordinator) Stop() (State, error)`
- `func (c *Coordinator) State() State`

`State` should be the same compact shape the HTTP API can later encode directly, not a hidden internal struct that requires a second JSON-facing DTO.

Planned `State` fields:

- `status` as the explicit string enum above
- current `benchmark.StartOptions`
- `started_at` optional timestamp
- `stopped_at` optional timestamp
- `error` compact Go error string, empty when none

Do not add benchmark IDs, history slices, persistence handles, or placeholder result stores in this task.

## Boundary Decision

This task’s `improve-code-boundaries` move is to keep option semantics inside `internal/benchmark` and keep runtime orchestration inside `internal/benchmarkrun`.

Explicit ownership:

- `internal/benchmark` owns start/alter shapes, validation, and how a safe alter updates the canonical options
- `internal/benchmarkrun` owns the state machine, cancellation, runner lifecycle, and failure visibility
- `internal/httpserver` should later serialize `benchmarkrun.State` directly instead of inventing separate response DTOs
- `internal/app` should later wire the coordinator in, but must not become the courier for run state internals

Planned boundary cleanup during execution:

- add a small helper in `internal/benchmark` such as `func (o StartOptions) ApplyAlter(alter AlterOptions) StartOptions`
- do not let the coordinator hand-edit option fields or duplicate alter semantics
- keep most coordinator fields and helper functions private; export only the state snapshot and lifecycle methods

## Worker Contract

Keep the runner boundary narrow and future-compatible with the later SQL workload task.

Planned interface:

```go
type Runner interface {
	Start(ctx context.Context, options benchmark.StartOptions) (Run, error)
}

type Run interface {
	Alter(options benchmark.AlterOptions) error
	Wait() error
}
```

Why this shape:

- `Start` can fail synchronously, which maps cleanly to `starting -> failed`
- `Wait` lets the coordinator observe asynchronous worker completion or failure
- `Alter` keeps live runtime changes explicit without leaking worker internals into the coordinator
- `Stop` should be driven by context cancellation owned by the coordinator, not by a second stop API on the worker

## State Machine

Initial state:

- `idle`

Allowed transitions:

- `idle -> starting -> running`
- `starting -> failed` when runner startup fails
- `running -> stopping -> stopped` when stop cancellation succeeds
- `running -> failed` when the worker exits with a non-cancellation error
- `starting -> stopping -> stopped` if stop is requested during startup and the worker honors cancellation
- `stopped -> starting`
- `failed -> starting`

Rules:

- `Start` must reject when the current status is `starting`, `running`, or `stopping`
- `Stop` must be idempotent when already `stopping` or already not active
- `Alter` is only allowed while `running`
- worker completion with `context.Canceled` after a requested stop should become `stopped`, not `failed`
- every failure must update `State.Error` with the Go error string

## Test Strategy

Use vertical red/green slices against the public coordinator API only. Tests should use a controllable fake runner inside `internal/benchmarkrun` tests; they should not reach into coordinator internals.

Tracer bullet and slices:

- [x] Slice 1: failing test for `Start` moving `idle -> running` and exposing current options in `State`
- [x] Slice 2: failing test for rejecting a second `Start` while the first run is active
- [x] Slice 3: failing test for `Stop` canceling the run and ending in `stopped`
- [x] Slice 4: failing test for idempotent `Stop` when already stopping or already stopped
- [x] Slice 5: failing test for `Alter` while running, including updated current options in state and forwarding to the live run
- [x] Slice 6: failing test for rejecting `Alter` when not running
- [x] Slice 7: failing test for worker failure setting `failed` plus visible Go error text in state
- [x] Slice 8: refactor after green to keep option alteration in `internal/benchmark` and keep coordinator helpers private

## File Plan

Expected files:

- `internal/benchmark/options.go`
- `internal/benchmark/options_test.go`
- `internal/benchmarkrun/coordinator.go`
- `internal/benchmarkrun/coordinator_test.go`

Avoid touching `internal/httpserver` in this task unless a small compile-only wiring adjustment becomes necessary. The API task should consume this package later.

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless execution ends up changing ultra-long test selection, which this task should not do

If execution shows that `State` needs separate `requested_options` versus `effective_options`, or that the runner contract needs stats callbacks right now, switch this plan back to `TO BE VERIFIED` instead of forcing a muddy boundary.

NOW EXECUTE
