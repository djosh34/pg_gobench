# Multi-Source Credential Validation Plan

## Scope

Fix the config-loading bug where a credential field that sets more than one source mode can report an ambient-environment lookup error instead of the structural validation error for the invalid config shape.

This task should deliver:

- one canonical validation order for credential fields so config shape errors win before any env-var or secret-file resolution
- a Red-Green test that proves the same invalid config reports the same structural error regardless of whether the referenced environment variable is set
- a small config-boundary cleanup that separates "which credential mode is selected" from "resolve that selected mode into a concrete value"
- manual verification against the documented container reproduction so the shipped runtime matches the task report

This task should not deliver:

- any public config-schema change
- any new credential source mode
- validation moved outside `internal/config`
- extra compatibility behavior for invalid configs

## Public Interface

Keep the public contract small and unchanged:

- `config.Load(path)` remains the entry point
- YAML still supports exactly the existing `value`, `env-ref`, and `secret-file` modes for `source.username` and `source.password`
- the observable fix is only the returned error precedence for structurally invalid credential objects

Expected observable behavior after execution:

- when a credential object sets more than one of `value`, `env-ref`, or `secret-file`, `config.Load` returns `<field> must set exactly one of value, env-ref, or secret-file`
- that exact-one error is returned whether the referenced env var exists, is missing, or the secret file would otherwise resolve
- valid single-mode credentials still resolve exactly as before

If RED or manual verification shows the contract needs a broader change than error precedence inside `internal/config`, switch this plan back to `TO BE VERIFIED` immediately instead of widening the boundary implicitly.

## Boundary Decision

The `improve-code-boundaries` move for this bug is to remove the mixed responsibility currently packed into `resolveCredential`.

Current smell:

- `resolveCredential` both validates the credential-object shape and performs side-effecting resolution of env vars / secret files
- because those responsibilities are interleaved, ambient environment state can change which error the same invalid config reports
- that means the validated config boundary is not actually deterministic for invalid shapes

Planned cleanup:

- keep all credential validation and resolution private inside `internal/config`
- introduce one small private representation for the chosen credential source mode after shape validation succeeds
- validate the mapping shape first, including the exact-one rule, before any lookup or file read occurs
- resolve the already-validated mode in a second private step

That keeps validation once and only once inside config, while making the returned error depend on config shape rather than ambient process state.

## TDD Strategy

Follow strict vertical slices. One failing behavior test, the minimum implementation to pass it, then manual verification. Only add another RED if the real reproduction still fails in a different way.

Planned slices:

- [ ] Slice 1: add one failing `config.Load` test that uses an invalid `source.username` object with both `value` and `env-ref`, leaves the env var unset, and asserts the returned error is the exact-one structural error rather than the env-var lookup error
- [ ] Slice 2: make that test green with the smallest private config refactor that validates credential mode selection before any mode resolution happens
- [ ] Slice 3: manually rerun the reported container reproduction; if the bug still reproduces through another multi-mode combination such as `env-ref` plus `secret-file`, add exactly one new failing test for that observed path and make it green before changing more code
- [ ] Slice 4: refactor only after GREEN to keep the credential-selection boundary simple, private, and free of duplicated mode-count logic

The first RED must stay at the public behavior boundary:

- use `config.Load`, not a private helper
- assert on the returned error text users/operators observe
- avoid brittle tests tied to helper names or internal function splits

## Implementation Notes

Start with the smallest test change needed to capture the real bug:

- extend `internal/config/config_test.go` with one focused invalid-config test case that proves missing `POSTGRES_USERNAME` must not outrank the exact-one error
- keep the test integration-style by driving the real config file through `config.Load`

Then make the code change in `internal/config/config.go`:

- separate credential parsing into two private phases: choose/validate mode, then resolve that validated mode
- ensure `rejectUnknownFields` and the exact-one check happen before `os.LookupEnv` or `os.ReadFile`
- keep empty-value handling on the resolution side for valid single-mode credentials

Possible private code shape:

- a small private `credentialMode` enum-like string or struct carrying the selected kind and source payload
- a `parseCredentialSource(...)` helper that returns the validated mode without side effects
- a `resolveCredentialSource(...)` helper that performs env/file/value resolution after shape validation succeeds

Avoid these muddy shortcuts:

- do not duplicate three separate mode-count branches across username and password parsing
- do not keep validation and side effects interleaved in one long helper
- do not move config validation into callers or later runtime stages

## File Plan

Expected files:

- `internal/config/config.go`
- `internal/config/config_test.go`
- `.ralph/tasks/bugs/bug-multi-source-credential-validation-reports-wrong-error.md`

No exported API expansion and no new package should be needed.

## Manual Verification

After the green test is in place:

1. `docker compose -f examples/docker-compose-postgres/compose.yaml -p pg-gobench-quickstart up -d --build`
2. create the invalid config from the task description with both `value` and `env-ref` under `source.username`
3. run the shipped container without `POSTGRES_USERNAME` set
4. rerun with `POSTGRES_USERNAME=benchmark_user`

Expected result:

- both runs fail with `source.username must set exactly one of value, env-ref, or secret-file`
- the reported error no longer changes with ambient environment state

If manual verification exposes a different structural-precedence failure than the first test captured, add one new failing test for that exact case before changing more code again.

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless execution proves this bug changed the ultra-long lane

## Execution Rule

If execution shows that the only honest fix requires a public error-type redesign, duplicated credential models across packages, or config validation outside `internal/config`, switch this plan back to `TO BE VERIFIED` immediately instead of forcing a muddier boundary.

NOW EXECUTE
