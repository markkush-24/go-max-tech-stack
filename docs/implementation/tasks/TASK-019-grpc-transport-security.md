# TASK-019 — Add gRPC TLS or mTLS and environment-controlled reflection

## Metadata

- Phase: `P1`
- Priority: `P0`
- Remediation group: `RG-GRPC-SECURITY`
- Gap: `GAP-GRPC-SECURITY-019`
- Dependencies: TASK-018, TASK-015
- Initial status: `BACKLOG`

## Goal

Add gRPC TLS or mTLS and environment-controlled reflection. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A12-F1: Direct gRPC is an authentication and authorization bypass
- A12-F10: TLS and HTTP2 have no explicit production security profile

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/transport/grpcserver/**`
- `internal/transport/grpcclient/**`
- `internal/config/**`
- `cmd/api/main.go`
- `certs/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Apply transport credentials according to the ADR.
2. Validate certificate/server-name configuration.
3. Disable reflection outside explicit development mode.
4. Add secure client/server integration tests.

## Acceptance criteria

- [ ] Plaintext direct access is impossible in protected mode.
- [ ] Reflection follows the environment policy.
- [ ] Misconfigured credentials fail startup clearly.

## Required verification

- `go test ./internal/transport/grpcserver/... ./internal/transport/grpcclient/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
