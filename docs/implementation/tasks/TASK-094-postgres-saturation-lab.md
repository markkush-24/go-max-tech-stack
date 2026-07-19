# TASK-094 — Create reproducible PostgreSQL pool and query saturation experiments

## Metadata

- Phase: `P7`
- Priority: `P1`
- Remediation group: `RG-PERF`
- Gap: `GAP-PERF-094`
- Dependencies: TASK-078, TASK-088, TASK-037
- Initial status: `BACKLOG`

## Goal

Create reproducible PostgreSQL pool and query saturation experiments. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A09-F22: PostgreSQL baseline environment is not reproducibly constrained
- A09-F8: Worker, queue and DB pool sizing have no explicit capacity relationship

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `loadtest/**`
- `docker-compose*.yml`
- `docs/performance/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Pin dataset size and PostgreSQL resource settings.
2. Sweep max-open connections and workload mix.
3. Measure pool wait, query latency, locks, CPU and I/O.
4. Include schema/version and seed commands.

## Acceptance criteria

- [ ] Baseline environment can be recreated.
- [ ] Pool starvation and DB saturation are distinguishable.
- [ ] Results state whether bottleneck is app or database.

## Required verification

- `docker compose config`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
