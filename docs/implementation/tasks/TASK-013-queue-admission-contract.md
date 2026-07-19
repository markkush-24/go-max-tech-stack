# TASK-013 — Define deterministic queue cancellation and StopAccepting semantics

## Metadata

- Phase: `P1`
- Priority: `P0`
- Remediation group: `RG-CONCURRENCY`
- Gap: `GAP-CONCURRENCY-013`
- Dependencies: TASK-007
- Initial status: `BACKLOG`

## Goal

Define deterministic queue cancellation and StopAccepting semantics. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A10-F7: Canceled context can still enqueue work
- A10-F8: Queue stop is not a strict barrier for an already-entered Enqueue

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/queue/**`
- `internal/routes/users_handler*.go`
- `internal/routes/*test.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Reject an already-canceled context before attempting channel send.
2. Document whether in-flight admissions may complete after StopAccepting.
3. If a strict barrier is required, implement it without closing the producer channel.
4. Add deterministic cancellation/barrier tests.

## Acceptance criteria

- [ ] Canceled-before-call enqueue never succeeds.
- [ ] Shutdown admission semantics are documented and tested.
- [ ] No send-on-closed design is introduced.

## Required verification

- `go test -race ./internal/queue/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
