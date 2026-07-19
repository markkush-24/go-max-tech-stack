# TASK-028 — Inject request ID, trace ID and span ID into contextual slog events

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-OTEL`
- Gap: `GAP-OTEL-028`
- Dependencies: TASK-023, TASK-025
- Initial status: `BACKLOG`

## Goal

Inject request ID, trace ID and span ID into contextual slog events. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A03-L1: Async worker logs cannot correlate to the initiating request
- A03-L2: No trace ID or span ID correlation exists
- A05-T6: Outbound correlation is duplicated between context and an explicit string parameter

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/telemetry/**`
- `internal/middleware/**`
- `internal/interceptors/**`
- `internal/outbound/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Provide a context-aware logger helper.
2. Attach valid trace/span identifiers only when present.
3. Keep request ID distinct from trace ID.
4. Avoid using trace/request IDs as Loki labels or metric attributes.

## Acceptance criteria

- [ ] HTTP, outbound, gRPC and worker logs share trace correlation where expected.
- [ ] No fake zero identifiers are emitted.
- [ ] Correlation tests use an in-memory exporter/handler.

## Required verification

- `go test ./internal/telemetry/... ./internal/middleware/... ./internal/interceptors/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
