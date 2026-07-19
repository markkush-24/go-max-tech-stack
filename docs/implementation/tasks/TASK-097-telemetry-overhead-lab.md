# TASK-097 — Measure telemetry overhead and Collector outage behavior

## Metadata

- Phase: `P7`
- Priority: `P1`
- Remediation group: `RG-PERF`
- Gap: `GAP-PERF-097`
- Dependencies: TASK-038, TASK-046, TASK-089
- Initial status: `BACKLOG`

## Goal

Measure telemetry overhead and Collector outage behavior. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A06-G11: Collector self-observability
- A09-F16: HTTP access logging can become a load bottleneck
- A09-F17: Current expvar metrics add per-request string work and shared-map contention
- A09-F21: No process/container/host metrics exist yet

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `loadtest/**`
- `docs/performance/**`
- `observability/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Compare telemetry disabled, metrics only, traces+metrics and full logs.
2. Vary sampling ratio.
3. Simulate Collector unavailable and saturated queues.
4. Measure throughput, p99, CPU, heap and dropped telemetry.

## Acceptance criteria

- [ ] Telemetry overhead is quantified.
- [ ] Collector outage does not create business outage or unbounded memory.
- [ ] Recommended default sampling/export settings are recorded.

## Required verification

- `docker compose config`
- `test -d loadtest`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
