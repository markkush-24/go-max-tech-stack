# TASK-018 — Record the direct gRPC trust and exposure model

## Metadata

- Phase: `P1`
- Priority: `P0`
- Remediation group: `RG-GRPC-SECURITY`
- Gap: `GAP-GRPC-SECURITY-018`
- Dependencies: TASK-005
- Initial status: `BACKLOG`

## Goal

Record the direct gRPC trust and exposure model. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A12-F1: Direct gRPC is an authentication and authorization bypass

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `docs/adr/**`
- `README.md`
- `internal/config/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Decide whether direct gRPC is private mTLS, authenticated public/internal, or loopback-only.
2. Define reflection policy by environment.
3. Define identity propagation and resource authorization requirements.
4. Record certificate and caller trust boundaries.

## Acceptance criteria

- [ ] The ADR chooses one deployable model.
- [ ] No direct gRPC exposure remains implicitly trusted.
- [ ] Subsequent tasks have concrete security acceptance criteria.

## Required verification

- `test -f docs/adr/*grpc* || true`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
