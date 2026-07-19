# TASK-031 — Instrument gRPC client/server and stream propagation

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-OTEL`
- Gap: `GAP-OTEL-031`
- Dependencies: TASK-025, TASK-021
- Initial status: `BACKLOG`

## Goal

Instrument gRPC client/server and stream propagation. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A02-E4: gRPC bridge propagation is request-id only
- A05-T1: Distributed tracing is not implemented
- A05-T10: `metadata.NewOutgoingContext` replaces existing outgoing metadata
- A05-T13: There is no gRPC client interceptor and no stream interceptor baseline
- A05-T9: The HTTP-to-gRPC bridge propagates only request ID

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/transport/grpcclient/**`
- `internal/transport/grpcserver/**`
- `internal/interceptors/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Use supported gRPC tracing integration for client and server.
2. Preserve existing request-ID and auth metadata.
3. Cover unary and stream interceptor/stats-handler baseline.
4. Map RPC codes and deadlines into spans.

## Acceptance criteria

- [ ] HTTP bridge trace continues through gRPC client and server.
- [ ] Direct gRPC incoming trace context is continued.
- [ ] Metadata is composed, not replaced.

## Required verification

- `go test ./internal/transport/grpcclient/... ./internal/transport/grpcserver/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
