# TASK-046 — Add telemetry outage, flush and propagation tests

## Metadata

- Phase: `P3`
- Priority: `P1`
- Remediation group: `RG-OTEL`
- Gap: `GAP-OTEL-046`
- Dependencies: TASK-026, TASK-031, TASK-027, TASK-038
- Initial status: `BACKLOG`

## Goal

Add telemetry outage, flush and propagation tests. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A05-T19: Propagation tests cover request ID and some cancellation, not distributed context
- A06-G11: Collector self-observability
- A08-F15: Collector/exporter outage policy is not implemented
- A08-F16: Current tests do not cover shutdown outcome truth
- A13-Q11: Global `expvar` state prevents test isolation
- A13-Q20: Previously discovered defects lack regression tests

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/telemetry/**/*_test.go`
- `internal/middleware/**/*_test.go`
- `internal/transport/**/*_test.go`
- `internal/workerpool/**/*_test.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Test incoming and outgoing trace propagation.
2. Test async causality and per-job spans.
3. Test Collector unavailable and bounded exporter behavior.
4. Test ForceFlush/Shutdown once on normal and fatal exits.

## Acceptance criteria

- [ ] Business readiness remains healthy during Collector outage.
- [ ] Final spans are exported or timeout outcome is explicit.
- [ ] Route names and attributes remain low-cardinality.

## Required verification

- `go test -race ./internal/telemetry/... ./internal/middleware/... ./internal/transport/... ./internal/workerpool/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
