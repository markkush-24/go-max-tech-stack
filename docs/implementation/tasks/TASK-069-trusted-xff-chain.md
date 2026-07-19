# TASK-069 — Define and implement trusted X-Forwarded-For chain semantics

## Metadata

- Phase: `P5`
- Priority: `P1`
- Remediation group: `RG-SECURITY`
- Gap: `GAP-SECURITY-069`
- Dependencies: TASK-022
- Initial status: `BACKLOG`

## Goal

Define and implement trusted X-Forwarded-For chain semantics. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A12-F12: Trusted X-Forwarded-For semantics depend on undocumented proxy sanitization

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/middleware/trustproxy.go`
- `internal/security/requestinfo.go`
- `internal/config/**`
- `README.md`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Document whether trusted proxies overwrite or append XFF.
2. If chains are supported, evaluate trust right-to-left.
3. Reject malformed/untrusted chain values.
4. Keep direct peer as the trust anchor.

## Acceptance criteria

- [ ] Client-supplied leftmost XFF cannot spoof ClientIP through a trusted appending proxy.
- [ ] Single trusted proxy behavior is tested.
- [ ] Documentation matches implementation.

## Required verification

- `go test ./internal/middleware/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
