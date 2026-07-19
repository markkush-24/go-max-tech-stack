# TASK-090 — Add queue, worker and DB saturation experiments

## Metadata

- Phase: `P7`
- Priority: `P1`
- Remediation group: `RG-PERF`
- Gap: `GAP-PERF-090`
- Dependencies: TASK-089, TASK-035, TASK-051
- Initial status: `BACKLOG`

## Goal

Add queue, worker and DB saturation experiments. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A09-F8: Worker, queue and DB pool sizing have no explicit capacity relationship
- A09-F9: Queue observability cannot support capacity tuning

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `loadtest/**`
- `docs/performance/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Sweep worker count, queue capacity, DB pool and incoming RPS.
2. Measure queue age, wait, processing and terminality.
3. Include overload and recovery phases.
4. Capture profiles at the first saturation knee.

## Acceptance criteria

- [ ] The experiment identifies the bottleneck for each matrix point.
- [ ] No accepted job is silently lost.
- [ ] Recovery after reducing load is measured.

## Required verification

- `test -d loadtest`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
