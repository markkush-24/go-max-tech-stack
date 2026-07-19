# TASK-068 — Apply configured security headers and host-wide HSTS correctly

## Metadata

- Phase: `P5`
- Priority: `P1`
- Remediation group: `RG-SECURITY`
- Gap: `GAP-SECURITY-068`
- Dependencies: TASK-022
- Initial status: `BACKLOG`

## Goal

Apply configured security headers and host-wide HSTS correctly. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A12-F8: Security-header configuration is partly ignored
- A12-F9: HSTS is scoped to the API subtree instead of the HTTPS host

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/middleware/security_header.go`
- `internal/router/root.go`
- `internal/config/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Apply configured Referrer-Policy.
2. Apply HSTS to secure host responses, not only API subtree.
3. Keep HSTS disabled for effective HTTP.
4. Document CSP/frame and TLS profile choices.

## Acceptance criteria

- [ ] Secure health/debug/root responses receive host policy as intended.
- [ ] HTTP responses never receive HSTS.
- [ ] Config tests verify each header.

## Required verification

- `go test ./internal/middleware/... ./internal/router/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
