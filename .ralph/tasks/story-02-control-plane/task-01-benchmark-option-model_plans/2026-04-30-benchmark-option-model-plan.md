# Benchmark Option Model Plan

## Scope

Define one canonical benchmark option model for the control-plane JSON API, with defaults and validation owned outside `internal/httpserver`.

This task should:

- define the start and alter JSON shapes for benchmark control
- reject unknown JSON fields at decode time
- apply defaults only for start requests
- keep alter requests intentionally narrower than start requests
- record a short maintainer note explaining why the final option set mirrors ideas from `pgbench`, HammerDB, and sysbench

This turn is planning-only. Execution belongs to the next turn unless the design still looks wrong and must stay open.

## Research Summary

The planned option set is intentionally small, but it follows the recurring knobs exposed by the benchmark tools named in the task:

- PostgreSQL `pgbench` exposes scale factor, clients, run duration, built-in workload variants, and optional rate limiting; its docs also warn that scale should not be unrealistically small for the chosen concurrency.
- HammerDB exposes rampup time, timed duration, virtual-user/session scaling, and transactional versus analytic workload families (`TPROC-C` and `TPROC-H` derived flows).
- sysbench exposes threads, time, warmup time, and optional transaction-rate limiting for OLTP-style workloads.

That combination supports a compact JSON model with:

- dataset scale
- client count
- timed duration
- warmup duration
- workload profile
- optional read/write mix
- optional transaction mix
- optional target TPS limit

## Public Interface

Create a new package, `internal/benchmark`, as the single owner of benchmark option shapes.

Exported surface:

- `type Profile string`
- `type TransactionMix string`
- `type StartOptions struct`
- `type AlterOptions struct`
- `func DecodeStartOptions(r io.Reader) (StartOptions, error)`
- `func DecodeAlterOptions(r io.Reader) (AlterOptions, error)`

Keep validation helpers private to the package so future coordinator and HTTP code consume already-normalized option values instead of re-validating or translating them.

## Boundary Decision

Use `internal/benchmark` as the canonical boundary for both the API contract and the domain option model.

Planned ownership:

- `internal/httpserver` should later decode request bodies by calling `benchmark.DecodeStartOptions` / `benchmark.DecodeAlterOptions`
- the future run coordinator should later accept `benchmark.StartOptions` and `benchmark.AlterOptions` directly
- no `httpserver.StartRequest`, `coordinator.StartConfig`, or other duplicate DTO layer should be introduced for the same data

This is the explicit `improve-code-boundaries` move for this task:

- one option shape instead of request DTO -> domain DTO -> runner DTO conversions
- one package owns defaults, enum validation, and unknown-field rejection
- future HTTP handlers stay thin and future coordinator code stays free of JSON concerns

## Planned JSON Shape

Keep field names explicit and flat. Use integer seconds rather than Go duration strings so the JSON stays simple.

`StartOptions`:

- `scale` integer, default `10`, must be `>= 1`
- `clients` integer, default `1`, must be `>= 1`
- `duration_seconds` integer, default `60`, must be `>= 1`
- `warmup_seconds` integer, default `10`, must be `>= 0` and `< duration_seconds`
- `profile` enum, default `"mixed"`
- `read_percent` integer, default `80` only when `profile == "mixed"`, must be `0..100`
- `transaction_mix` enum, default `"balanced"` only when `profile == "transaction"`
- `target_tps` integer, optional/unlimited when omitted, must be `>= 1` when present

Accepted `profile` values:

- `read`
- `write`
- `transaction`
- `join`
- `lock`
- `mixed`

Accepted `transaction_mix` values:

- `balanced`
- `read-heavy`
- `write-heavy`

Profile-specific validation:

- `read_percent` is allowed only for `mixed`
- `transaction_mix` is allowed only for `transaction`
- `lock` requires `clients >= 2` because a single client cannot exercise contention meaningfully
- start requests must reject any attempt to send unsupported flat fields

`AlterOptions`:

- `clients` optional; when present must be `>= 1`
- `target_tps` optional; when present must be `>= 1`
- at least one field must be present

Alter requests must reject these categories entirely:

- scale or schema-size changes
- profile or mix changes
- duration or warmup changes
- connection settings
- destructive setup/prepare behavior

## Maintainer Note

Execution should add a short package comment or doc note near the profile/options definitions that cites the design inspiration:

- `pgbench` built-in workload families and `--client` / `--time` / `--rate` / scale guidance
- HammerDB rampup plus transactional vs analytic families
- sysbench threads/time/warmup/rate controls

Keep that note brief and visible; do not create a large compatibility matrix.

## TDD Slices

Use vertical red-green slices only.

- [x] Slice 1: failing public test for decoding a minimal start JSON payload and receiving defaults for omitted fields
- [x] Slice 2: failing public test for rejecting unknown JSON fields in start requests
- [x] Slice 3: failing public test for scale, clients, duration, warmup, and profile-specific validation failures
- [x] Slice 4: failing public test for alter requests accepting only `clients` and `target_tps`
- [x] Slice 5: failing public test for alter requests rejecting empty payloads and rejected fields such as `scale`, `profile`, or `duration_seconds`
- [x] Slice 6: refactor after green to keep JSON decode + validation entirely in `internal/benchmark` with no duplicate request shape elsewhere

## File Plan

- `internal/benchmark/options.go`
- `internal/benchmark/options_test.go`

Possible follow-on cleanup during execution if it improves boundaries:

- remove any temporary helper type that only exists to shuttle JSON into canonical options
- avoid threading benchmark options through `internal/app` before the HTTP API task actually needs that wiring

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless execution changes long-test selection or the task later proves it explicitly needs the long lane

If execution shows that `transaction_mix` needs a different enum shape, or that `read_percent` should apply to more than the `mixed` profile, switch this plan back to `TO BE VERIFIED` immediately instead of forcing the wrong model through implementation.

NOW EXECUTE
