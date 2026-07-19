# TASK-026 — Integrate telemetry fail-open, ForceFlush and Shutdown lifecycle

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-OTEL`
- Gap: `GAP-OTEL-026`
- Dependencies: TASK-025, TASK-017
- Initial status: `BACKLOG`

## Goal

Integrate telemetry fail-open, ForceFlush and Shutdown lifecycle. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A05-T17: Startup and shutdown contexts are separated from request cancellation, but telemetry lifecycle is absent
- A06-G10: Telemetry shutdown/flush
- A06-G9: Telemetry failure policy
- A08-F14: Component stop order is not explicitly tied to telemetry completion
- A08-F15: Collector/exporter outage policy is not implemented

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/telemetry/**`
- `internal/api/**`
- `cmd/api/main.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Keep telemetry out of business readiness.
2. Use bounded exporter queues and timeouts.
3. Flush and shut down providers after business components.
4. Surface dropped/export failures through internal telemetry without blocking requests.

## Acceptance criteria

- [ ] Collector outage does not make `/readyz` fail.
- [ ] Shutdown calls provider cleanup once within budget.
- [ ] Final shutdown telemetry is attempted before process exit.

## Required verification

- `go test ./internal/telemetry/... ./internal/api/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
