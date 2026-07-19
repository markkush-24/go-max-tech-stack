# TASK-021 — Normalize gRPC metadata, request ID, status codes and bridge timeout

## Metadata

- Phase: `P1`
- Priority: `P0`
- Remediation group: `RG-GRPC-CONTRACT`
- Gap: `GAP-GRPC-CONTRACT-021`
- Dependencies: TASK-020
- Initial status: `BACKLOG`

## Goal

Normalize gRPC metadata, request ID, status codes and bridge timeout. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A02-E4: gRPC bridge propagation is request-id only
- A05-T10: `metadata.NewOutgoingContext` replaces existing outgoing metadata
- A05-T11: gRPC request-ID metadata is trusted without sanitization
- A05-T12: A generated gRPC request ID is not returned to the client
- A05-T13: There is no gRPC client interceptor and no stream interceptor baseline
- A05-T9: The HTTP-to-gRPC bridge propagates only request ID
- A11-F11: HTTP-to-gRPC bridge has no explicit call budget
- A12-F2: gRPC status and metadata contracts are incomplete

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/interceptors/**`
- `internal/transport/grpcclient/**`
- `internal/transport/grpcserver/**`
- `internal/httpapi/jobs.go`
- `internal/httputils/grpc_bridge_error.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Append to existing outgoing metadata rather than replacing it.
2. Sanitize inbound request IDs and return generated IDs in response metadata.
3. Map cancellation, deadline, unavailable, permission and not-found accurately.
4. Apply an explicit HTTP-to-gRPC call budget.

## Acceptance criteria

- [ ] Clients receive stable request ID metadata.
- [ ] Canceled/deadline calls do not become Internal.
- [ ] Bridge timeout is bounded below the outer HTTP budget.

## Required verification

- `go test ./internal/interceptors/... ./internal/transport/grpcclient/... ./internal/transport/grpcserver/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
