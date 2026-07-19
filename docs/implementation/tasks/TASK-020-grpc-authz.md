# TASK-020 — Add gRPC authentication, RBAC and owner authorization

## Metadata

- Phase: `P1`
- Priority: `P0`
- Remediation group: `RG-GRPC-SECURITY`
- Gap: `GAP-GRPC-SECURITY-020`
- Dependencies: TASK-018, TASK-019
- Initial status: `BACKLOG`

## Goal

Add gRPC authentication, RBAC and owner authorization. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A12-F1: Direct gRPC is an authentication and authorization bypass

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/interceptors/**`
- `internal/transport/grpcserver/**`
- `internal/security/**`
- `internal/config/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Authenticate direct RPC callers using the chosen credential model.
2. Map identity into context without using context as a service locator.
3. Apply RBAC and owner/admin checks equivalent to HTTP.
4. Add tests for missing, invalid, forbidden and owner/admin calls.

## Acceptance criteria

- [ ] Direct unauthenticated GetJob is rejected.
- [ ] A user cannot read another user’s job.
- [ ] Admin access remains available.

## Required verification

- `go test ./internal/interceptors/... ./internal/transport/grpcserver/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
