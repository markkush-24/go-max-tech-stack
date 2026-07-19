# TASK-035 — Add complete queue and Job lifecycle metrics

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-METRICS`
- Gap: `GAP-METRICS-035`
- Dependencies: TASK-032, TASK-009
- Initial status: `BACKLOG`

## Goal

Add complete queue and Job lifecycle metrics. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A04-M7: `jobs_total` is a transition-event counter with incomplete terminal coverage
- A04-M8: Queue observability is too coarse even aside from the first-instance defect
- A07-SLO-5: Async acceptance and completion are conflated/incomplete
- A09-F9: Queue observability cannot support capacity tuning

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/metrics/**`
- `internal/queue/**`
- `internal/workerpool/**`
- `internal/service/jobService.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Separate accepted, transitions, current states and terminal outcomes.
2. Measure queue wait, processing and end-to-end duration.
3. Expose capacity, utilization, rejections, oldest age and accepting state.
4. Normalize system/business/shutdown failure reasons.

## Acceptance criteria

- [ ] Accepted-to-terminal ratio can be computed.
- [ ] Queue saturation and age are observable.
- [ ] Transition metrics remain correct under retry/repair.

## Required verification

- `go test -race ./internal/metrics/... ./internal/queue/... ./internal/workerpool/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
