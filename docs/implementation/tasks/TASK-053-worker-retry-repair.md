# TASK-053 — Execute durable retry, repair and reconciliation policies in workers

## Metadata

- Phase: `P4`
- Priority: `P1`
- Remediation group: `RG-ASYNC-DURABILITY`
- Gap: `GAP-ASYNC-DURABILITY-053`
- Dependencies: TASK-052, TASK-012
- Initial status: `BACKLOG`

## Goal

Execute durable retry, repair and reconciliation policies in workers. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A10-F10: Job transitions consume work without guaranteed terminality
- A10-F2: Active worker can overwrite shutdown terminal state
- A11-F5: No poison-job, retry, DLQ or reconciliation model exists

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/workerpool/**`
- `internal/service/**`
- `internal/store/jobrepo/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Claim work before processing.
2. Renew or respect leases for long operations.
3. Schedule retry or terminal failure through CAS transitions.
4. Make shutdown repair and startup reconciliation share one tested policy.

## Acceptance criteria

- [ ] A transient repository failure does not strand a job in queued/running.
- [ ] A terminal transition cannot be overwritten.
- [ ] Retry attempts are observable and bounded.

## Required verification

- `go test -race ./internal/workerpool/... ./internal/store/jobrepo/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
