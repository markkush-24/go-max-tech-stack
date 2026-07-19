# TASK-012 — Make worker Stop and terminal repair bounded and truly idempotent

## Metadata

- Phase: `P1`
- Priority: `P0`
- Remediation group: `RG-LIFECYCLE`
- Gap: `GAP-LIFECYCLE-012`
- Dependencies: TASK-011, TASK-010
- Initial status: `BACKLOG`

## Goal

Make worker Stop and terminal repair bounded and truly idempotent. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A02-E5: Worker terminal write uses unbounded background context
- A08-F10: Worker terminal fallback uses an unbounded background context
- A08-F11: A timed-out `WorkerPool.Stop` leaves a waiter goroutine behind
- A08-F9: Worker wait and terminal repair consume the same context budget
- A10-F1: `WorkerPool.Stop` is not idempotent
- A10-F11: Async enqueue rollback uses the request context
- A10-F3: Timed-out Stop allows unsafe restart of the same pool
- A10-F4: `WaitGroup` lifecycle can be reused before previous Wait returns
- A10-F5: Stop timeout can leak waiter goroutines
- A10-F6: `running` does not represent actual worker health

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/workerpool/**`
- `internal/service/jobService.go`
- `internal/store/jobrepo/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Use one stable generation `done` channel rather than a waiter goroutine per Stop.
2. Separate worker-wait and terminal-repair budgets.
3. Use a detached but bounded repair context.
4. Repeated Stop calls return the same completed outcome and perform no conflicting transitions.

## Acceptance criteria

- [ ] Repeated Stop cannot change a terminal job.
- [ ] No waiter goroutine remains after timeout scenarios.
- [ ] Repair outcome is surfaced and testable.

## Required verification

- `go test -race ./internal/workerpool/... ./internal/store/jobrepo/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
