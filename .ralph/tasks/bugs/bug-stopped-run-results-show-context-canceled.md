## Bug: Stopped Run Results Surface Context Canceled As Latest Error <status>done</status> <passes>true</passes> <priority>high</priority>

<description>
During the story-99 manual verification pass against the real Docker Compose stack, a user-triggered stop left the benchmark in `status: "stopped"` but `/benchmark/results` still reported `stats.latest_error: "context canceled"`.

This is a real product bug because a normal, successful user stop should not surface an error-looking latest error string in the final results snapshot. It makes the JSON API and standalone UI report a misleading failure signal after an otherwise successful stop.

Reproduction used the documented quick-start stack:

1. `docker compose -f examples/docker-compose-postgres/compose.yaml -p pg-gobench-quickstart up -d --build`
2. `POST /benchmark/start` with `{"scale":1,"clients":2,"duration_seconds":20,"warmup_seconds":1,"reset":true,"profile":"mixed","read_percent":80}`
3. `POST /benchmark/alter` with `{"clients":3,"target_tps":150}` (optional; bug still matters without alter)
4. `POST /benchmark/stop`
5. `GET /benchmark/results`

Observed result:

```json
{
  "status": "stopped",
  "stats": {
    "latest_error": "context canceled"
  }
}
```

Expected result:

- `status` remains `stopped`
- coordinator `error` remains empty
- `stats.latest_error` is empty after a normal stop unless a real workload error happened before the stop
</description>

<mandatory_red_green_tdd>
Use Red-Green TDD to solve the problem.
You must make ONE test, and then make ONE test green at the time.

Then verify if bug still holds. If yes, create new Red test, and continue with Red-Green TDD until it does work.
</mandatory_red_green_tdd>

<acceptance_criteria>
- [x] I created a Red unit and/or integration test that captures the bug
- [x] I made the test green by fixing
- [x] I manually verified the bug, and created a new Red test if not working still
- [x] `make check` — passes cleanly
- [x] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [x] `make lint` — passes cleanly
- [ ] If this bug impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>

<plan>.ralph/tasks/bugs/bug-stopped-run-results-show-context-canceled_plans/2026-04-30-stopped-run-results-context-canceled-plan.md</plan>
