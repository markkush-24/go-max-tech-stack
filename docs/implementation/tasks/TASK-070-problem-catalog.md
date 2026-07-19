# TASK-070 — Adopt an RFC 9457 Problem catalog and explicit cache policy

## Metadata

- Phase: `P5`
- Priority: `P1`
- Remediation group: `RG-ERROR-CONTRACT`
- Gap: `GAP-ERROR-CONTRACT-070`
- Dependencies: TASK-061
- Initial status: `BACKLOG`

## Goal

Adopt an RFC 9457 Problem catalog and explicit cache policy. Complete this outcome without implementing adjacent roadmap tasks.

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

- `internal/httputils/problem.go`
- `internal/httputils/errmap.go`
- `internal/apperr/**`
- `docs/api/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Define stable type URIs and machine error codes.
2. Separate public detail from internal cause.
3. Set no-store or a reviewed cache policy for request-specific Problems.
4. Align validation, auth, overload, dependency and internal variants.

## Acceptance criteria

- [ ] Clients need not parse free-text detail to identify error type.
- [ ] Request-specific Problems are not accidentally shared-cached.
- [ ] HTTP and job/gRPC error representations derive from one application taxonomy.

## Required verification

- `go test ./internal/httputils/... ./internal/apperr/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
