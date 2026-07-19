# TASK-050 — Implement durable async admission without insert-delete compensation

## Metadata

- Phase: `P4`
- Priority: `P0`
- Remediation group: `RG-ASYNC-DURABILITY`
- Gap: `GAP-ASYNC-DURABILITY-050`
- Dependencies: TASK-049
- Initial status: `BACKLOG`

## Goal

Implement durable async admission without insert-delete compensation. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A10-F11: Async enqueue rollback uses the request context
- A11-F21: Queue overload response lacks durable admission semantics
- A11-F4: Queue-full compensation can create orphan jobs and amplify DB load

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/routes/users_handler*.go`
- `internal/service/jobService.go`
- `internal/store/jobrepo/**`
- `internal/queue/**`
- `internal/db/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Persist job and durable publication/outbox state according to the ADR.
2. Return 202 only after the accepted state is recoverable.
3. Do not delete a just-created job through the canceled request context.
4. Expose full/stopped/unavailable outcomes consistently.

## Acceptance criteria

- [ ] Queue saturation cannot create orphan queued jobs.
- [ ] A crash after 202 does not lose the payload.
- [ ] Overload no longer performs avoidable insert+delete amplification.

## Required verification

- `go test ./internal/routes/... ./internal/store/jobrepo/... ./internal/queue/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
