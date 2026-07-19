# TASK-033 — Fix queue/auth/bulkhead instrument semantics

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-METRICS`
- Gap: `GAP-METRICS-033`
- Dependencies: TASK-032
- Initial status: `BACKLOG`

## Goal

Fix queue/auth/bulkhead instrument semantics. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A04-M10: Authentication's first increment for a new kind can be lost under concurrency
- A10-F15: Global authentication first-increment update is not atomic
- A10-F16: Bulkhead gauge can transiently disagree with semaphore occupancy

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/queue/**`
- `internal/middleware/**`
- `internal/metrics/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Make queue depth/capacity/state instance-correct.
2. Use atomic instrument update APIs for first auth failure.
3. Define bulkhead in-flight semantics against actual permit ownership.
4. Add direct assertions for every corrected instrument.

## Acceptance criteria

- [ ] No first-increment loss is possible.
- [ ] Gauges match resource ownership under concurrency.
- [ ] Instrument names describe their real semantics.

## Required verification

- `go test -race ./internal/queue/... ./internal/middleware/... ./internal/metrics/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
