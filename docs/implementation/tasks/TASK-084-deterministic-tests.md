# TASK-084 — Remove avoidable sleeps, port races and brittle representation assertions

## Metadata

- Phase: `P6`
- Priority: `P1`
- Remediation group: `RG-QA`
- Gap: `GAP-QA-084`
- Dependencies: TASK-076
- Initial status: `BACKLOG`

## Goal

Remove avoidable sleeps, port races and brittle representation assertions. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A13-Q12: Timing-based tests contain avoidable flakiness

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/**/*_test.go`
- `internal/testkit/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Replace fixed sleeps with synchronization or eventually helpers with deadlines.
2. Use listener ownership instead of bind-close-rebind port reservation.
3. Assert semantic JSON/headers rather than unstable formatting.
4. Make test failure messages include causal state.

## Acceptance criteria

- [ ] Async/stream tests pass repeatedly under load and race detector.
- [ ] No fixed sleep is required to establish event ordering.
- [ ] Ephemeral ports cannot be stolen between allocation and use.

## Required verification

- `go test -count=50 ./internal/routes/... ./internal/api/... ./internal/transport/grpcserver/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
