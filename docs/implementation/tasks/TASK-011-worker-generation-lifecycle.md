# TASK-011 — Redesign WorkerPool around an immutable worker generation

## Metadata

- Phase: `P1`
- Priority: `P0`
- Remediation group: `RG-LIFECYCLE`
- Gap: `GAP-LIFECYCLE-011`
- Dependencies: TASK-009
- Initial status: `BACKLOG`

## Goal

Redesign WorkerPool around an immutable worker generation. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A10-F1: `WorkerPool.Stop` is not idempotent
- A10-F3: Timed-out Stop allows unsafe restart of the same pool
- A10-F4: `WaitGroup` lifecycle can be reused before previous Wait returns
- A10-F5: Stop timeout can leak waiter goroutines
- A10-F6: `running` does not represent actual worker health

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/workerpool/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Each `Start` creates an immutable generation containing context, cancel and done.
2. Worker loops receive their generation context as an argument.
3. Do not allow restart until the prior generation is fully stopped.
4. Expose actual lifecycle state without mutable context reuse.

## Acceptance criteria

- [ ] A timed-out stop cannot cause old workers to join a new generation.
- [ ] WaitGroup usage obeys one generation lifecycle.
- [ ] Concurrent Start/Stop calls have deterministic results.

## Required verification

- `go test -race ./internal/workerpool/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
