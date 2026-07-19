# TASK-072 — Add SSE snapshot, sequence and reconnect/resync semantics

## Metadata

- Phase: `P5`
- Priority: `P1`
- Remediation group: `RG-SSE`
- Gap: `GAP-SSE-072`
- Dependencies: TASK-071, TASK-010
- Initial status: `BACKLOG`

## Goal

Add SSE snapshot, sequence and reconnect/resync semantics. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A11-F7: SSE can permanently miss the terminal event
- A12-F18: SSE delivery has no resumable contract

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/stream/**`
- `internal/routes/job_handler.go`
- `internal/entity/job.go`
- `internal/store/jobrepo/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Send current state atomically with subscription or use a versioned resync protocol.
2. Add monotonic event ID/transition version.
3. Define Last-Event-ID behavior and replay/resync limits.
4. Prevent terminal event loss during subscribe race or buffer drop.

## Acceptance criteria

- [ ] A client connecting during terminal transition learns the terminal state.
- [ ] Reconnect from a known version has deterministic behavior.
- [ ] Dropped events trigger resync rather than silent permanent staleness.

## Required verification

- `go test -race ./internal/stream/... ./internal/routes/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
