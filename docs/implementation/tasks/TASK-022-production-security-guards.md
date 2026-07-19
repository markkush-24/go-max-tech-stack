# TASK-022 — Add environment guards for JWT, TLS, proxy and security-header defaults

## Metadata

- Phase: `P1`
- Priority: `P0`
- Remediation group: `RG-SECURITY`
- Gap: `GAP-SECURITY-022`
- Dependencies: TASK-005
- Initial status: `BACKLOG`

## Goal

Add environment guards for JWT, TLS, proxy and security-header defaults. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A03-L7: Privacy and redaction policy is not defined or enforced
- A03-L8: Security decisions have counters but no safe audit events
- A12-F10: TLS and HTTP2 have no explicit production security profile
- A12-F11: Development JWT defaults are unsafe without an environment guard

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/config/**`
- `internal/middleware/security_header.go`
- `internal/middleware/trustproxy.go`
- `cmd/api/main.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Reject known development JWT secrets outside explicit development mode.
2. Require issuer/audience and suitable key material in protected environments.
3. Apply configured Referrer-Policy instead of a hardcoded value.
4. Define minimum TLS version and trusted-proxy requirements.

## Acceptance criteria

- [ ] Unsafe development defaults cannot start in production-like mode.
- [ ] Configured security headers are the runtime behavior.
- [ ] TLS/proxy validation errors are actionable.

## Required verification

- `go test ./internal/config/... ./internal/middleware/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
