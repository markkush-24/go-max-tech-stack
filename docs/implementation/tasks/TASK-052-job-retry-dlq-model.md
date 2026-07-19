# TASK-052 — Add retry, backoff, poison-job and dead-letter state model

## Metadata

- Phase: `P4`
- Priority: `P1`
- Remediation group: `RG-ASYNC-DURABILITY`
- Gap: `GAP-ASYNC-DURABILITY-052`
- Dependencies: TASK-048
- Initial status: `BACKLOG`

## Goal

Add retry, backoff, poison-job and dead-letter state model. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A11-F5: No poison-job, retry, DLQ or reconciliation model exists

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/entity/job.go`
- `internal/service/jobService.go`
- `internal/store/jobrepo/**`
- `migrations/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Classify retryable and terminal job failures.
2. Persist attempt count and `next_attempt_at`.
3. Define a terminal dead-letter/manual-review state or equivalent.
4. Make policies configurable and bounded.

## Acceptance criteria

- [ ] Transient failures can be retried after a delay.
- [ ] Poison jobs reach a bounded terminal state.
- [ ] Operators can distinguish business failure from infrastructure failure.

## Required verification

- `go test ./internal/service/... ./internal/store/jobrepo/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
