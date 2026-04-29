## Task: 02 Implement Single Active Benchmark Run Coordinator <status>not_started</status> <passes>false</passes>

<blocked_by>.ralph/tasks/story-02-control-plane/task-01-benchmark-option-model.md</blocked_by>

<description>
Must use tdd skill to complete

**Goal:** Implement the in-memory benchmark run coordinator. The service supports exactly one benchmark at a time. A new start request must be rejected unless the current benchmark is stopped, failed, or otherwise not running.

The coordinator must own run state, cancellation, worker lifecycle, safe alteration of permitted runtime options, current options, start/stop timestamps, and the latest Go error text when a run fails. State should be simple and explicit, such as `idle`, `starting`, `running`, `stopping`, `stopped`, and `failed`.

Results and state are intentionally in memory only. Do not add persistent history, database-backed job records, or on-disk result storage. Errors must be returned and stored visibly; do not ignore worker errors.
</description>

<acceptance_criteria>
- [ ] TDD red/green coverage exists for the state machine from idle to running to stopped.
- [ ] TDD red/green coverage exists for rejecting a new benchmark while one is running.
- [ ] TDD red/green coverage exists for stop cancellation and idempotent stop behavior.
- [ ] TDD red/green coverage exists for permitted alter behavior while running and rejected unsafe alterations.
- [ ] TDD red/green coverage exists for worker failure causing visible failed state with the Go error string available in state JSON.
- [ ] Results and run history are held only in memory.
- [ ] `make check` — passes cleanly
- [ ] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [ ] `make lint` — passes cleanly
- [ ] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>
