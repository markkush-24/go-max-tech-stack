# TASK-030 — Instrument outbound HTTP and model logical retries

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-OTEL`
- Gap: `GAP-OTEL-030`
- Dependencies: TASK-025
- Initial status: `BACKLOG`

## Goal

Instrument outbound HTTP and model logical retries. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A05-T1: Distributed tracing is not implemented
- A05-T6: Outbound correlation is duplicated between context and an explicit string parameter
- A05-T7: Outbound HTTP propagates request ID but not standard trace context
- A05-T8: Retry attempts have no explicit span model

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/outbound/httpclient/**`
- `internal/outbound/profile*.go`
- `internal/outbound/profile/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Wrap the reused Transport with standard HTTP client instrumentation.
2. Preserve pooling, cancellation and CloseIdleConnections.
3. Create one logical Profile operation span and child attempt spans/events.
4. Inject `traceparent` and `tracestate` into upstream requests.

## Acceptance criteria

- [ ] Upstream receives standard trace context.
- [ ] Retries are visible as one operation with attempts.
- [ ] No new client/transport is created per request.

## Required verification

- `go test ./internal/outbound/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
