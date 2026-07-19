# TASK-067 — Introduce route-aware CORS method/header policies

## Metadata

- Phase: `P5`
- Priority: `P2`
- Remediation group: `RG-CORS`
- Gap: `GAP-CORS-067`
- Dependencies: TASK-066, TASK-064
- Initial status: `BACKLOG`

## Goal

Introduce route-aware CORS method/header policies. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A12-F7: CORS policy is global rather than route-specific

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/middleware/cors.go`
- `internal/router/**`
- `internal/config/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Derive allowed methods/headers from reviewed route groups or explicit policies.
2. Bypass auth only for valid CORS preflight, not every OPTIONS request.
3. Keep health/debug outside API CORS.
4. Avoid broadening credentials or origins.

## Acceptance criteria

- [ ] Preflight reflects actual route capability.
- [ ] Invalid/non-CORS OPTIONS follows normal routing/security.
- [ ] Policies remain testable without raw path matching.

## Required verification

- `go test ./internal/middleware/... ./internal/router/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
