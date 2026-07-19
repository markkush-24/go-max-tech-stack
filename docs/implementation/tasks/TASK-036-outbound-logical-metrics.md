# TASK-036 — Add logical Profile operation and retry metrics

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-METRICS`
- Gap: `GAP-METRICS-036`
- Dependencies: TASK-032
- Initial status: `BACKLOG`

## Goal

Add logical Profile operation and retry metrics. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A03-L9: Retry behavior is not visible as a coherent operation
- A04-M9: Outbound metrics describe physical attempts, not logical operations
- A07-SLO-6: Outbound user outcome is not represented

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/outbound/**`
- `internal/metrics/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Separate logical operations from physical HTTP attempts.
2. Measure final outcome and total operation duration.
3. Record retry count, exhausted retries and backoff duration.
4. Expose active operations with bounded dependency/route/error attributes.

## Acceptance criteria

- [ ] One user operation remains one denominator across retries.
- [ ] Retry amplification is directly measurable.
- [ ] 4xx/5xx/timeout/cancel classifications are stable.

## Required verification

- `go test ./internal/outbound/... ./internal/metrics/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
