# TASK-061 — Normalize DB, context, outbound and gRPC error taxonomy

## Metadata

- Phase: `P4`
- Priority: `P1`
- Remediation group: `RG-ERROR-CONTRACT`
- Gap: `GAP-ERROR-CONTRACT-061`
- Dependencies: TASK-021
- Initial status: `BACKLOG`

## Goal

Normalize DB, context, outbound and gRPC error taxonomy. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A11-F10: gRPC error mapping loses cancellation, deadline and availability semantics
- A11-F16: Profile 4xx semantics are flattened to HTTP 502
- A11-F9: Dependency/cancellation errors collapse into misleading HTTP 500 responses
- A12-F14: Problem Details source and machine contract are outdated/incomplete
- A12-F15: Some HTTP status mappings are semantically misleading
- A12-F16: Problem responses containing request-specific data have no cache policy

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/apperr/**`
- `internal/httputils/errmap.go`
- `internal/db/errors.go`
- `internal/outbound/profile/errors.go`
- `internal/httputils/grpc_bridge_error.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Distinguish client cancellation, operation deadline, dependency timeout/unavailable and internal failure.
2. Map HTTP and gRPC representations consistently.
3. Preserve errors.Is/As behavior.
4. Define retryability and logging severity independently from public detail.

## Acceptance criteria

- [ ] Canceled/deadline DB and gRPC operations do not collapse to generic 500/Internal.
- [ ] Profile 4xx policy is explicit.
- [ ] Tests cover each normalized error class.

## Required verification

- `go test ./internal/httputils/... ./internal/db/... ./internal/outbound/... ./internal/transport/grpcserver/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
