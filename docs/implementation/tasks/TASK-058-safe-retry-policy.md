# TASK-058 — Fix backoff overflow and honor bounded HTTP retry guidance

## Metadata

- Phase: `P4`
- Priority: `P1`
- Remediation group: `RG-OUTBOUND`
- Gap: `GAP-OUTBOUND-058`
- Dependencies: None
- Initial status: `BACKLOG`

## Goal

Fix backoff overflow and honor bounded HTTP retry guidance. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A11-F13: Backoff can overflow before applying its configured cap
- A11-F14: Retry policy is too coarse for HTTP status and server guidance

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/outbound/profile_retry.go`
- `internal/outbound/profile/**`
- `internal/config/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Compute exponential delay without duration overflow.
2. Bound maximum attempts and total retry budget.
3. Handle Retry-After where allowed.
4. Use a reviewed status/error retry matrix and preserve context deadlines.

## Acceptance criteria

- [ ] Large attempt values cannot wrap to zero/negative delay.
- [ ] 4xx and parse failures are not retried accidentally.
- [ ] Retry-After never exceeds remaining operation budget.

## Required verification

- `go test ./internal/outbound/... ./internal/config/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
