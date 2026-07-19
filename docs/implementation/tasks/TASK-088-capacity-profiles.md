# TASK-088 — Add explicit functional, baseline, observability and saturation config profiles

## Metadata

- Phase: `P7`
- Priority: `P1`
- Remediation group: `RG-PERF`
- Gap: `GAP-PERF-088`
- Dependencies: TASK-060, TASK-062
- Initial status: `BACKLOG`

## Goal

Add explicit functional, baseline, observability and saturation config profiles. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A07-SLO-12: Numerical objectives are provisional until baseline audit
- A08-F13: SSE timing has a boundary race with server `WriteTimeout`
- A09-F2: Current defaults are not a high-load profile
- A09-F22: PostgreSQL baseline environment is not reproducibly constrained
- A09-F8: Worker, queue and DB pool sizing have no explicit capacity relationship

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `configs/**`
- `docker-compose*.yml`
- `docs/performance/**`
- `scripts/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Create separate profiles for development correctness, in-memory baseline, PostgreSQL baseline, full observability and fault/saturation.
2. Set rate/bulkhead/worker/queue/DB/outbound values explicitly.
3. Record GOMAXPROCS, GOGC and memory limits.
4. Prevent accidental use of 5 RPS/global bulkhead 1 as a capacity baseline.

## Acceptance criteria

- [ ] Every load test names a profile.
- [ ] Profile values are version-controlled.
- [ ] Baseline and intentional-overload settings are visibly distinct.

## Required verification

- `find configs docs/performance -type f`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
