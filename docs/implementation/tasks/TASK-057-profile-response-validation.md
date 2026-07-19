# TASK-057 — Strictly validate and byte-bound Profile responses

## Metadata

- Phase: `P4`
- Priority: `P1`
- Remediation group: `RG-OUTBOUND`
- Gap: `GAP-OUTBOUND-057`
- Dependencies: None
- Initial status: `BACKLOG`

## Goal

Strictly validate and byte-bound Profile responses. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A09-F12: Outbound response draining is not byte-bounded
- A11-F15: Profile response validation is not strict enough

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/outbound/profile/**`
- `internal/outbound/profile_outbound_test.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Limit response body size.
2. Require one complete JSON value with EOF.
3. Validate Content-Type where contractually required.
4. Validate returned user ID and required semantic fields.

## Acceptance criteria

- [ ] Trailing JSON/data is rejected.
- [ ] Mismatched user ID is rejected.
- [ ] Oversized or wrong-media responses fail with typed errors.

## Required verification

- `go test ./internal/outbound/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
