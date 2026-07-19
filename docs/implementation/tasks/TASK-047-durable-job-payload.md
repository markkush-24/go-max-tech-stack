# TASK-047 — Persist durable Job payload and correlation metadata

## Metadata

- Phase: `P4`
- Priority: `P0`
- Remediation group: `RG-ASYNC-DURABILITY`
- Gap: `GAP-ASYNC-DURABILITY-047`
- Dependencies: TASK-010, TASK-027
- Initial status: `BACKLOG`

## Goal

Persist durable Job payload and correlation metadata. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A11-F1: Accepted async work is not durable or crash-recoverable
- A11-F22: Job model lacks recovery metadata

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/entity/job.go`
- `internal/store/jobrepo/**`
- `migrations/**`
- `internal/service/jobService.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Persist enough payload or a durable payload reference to replay accepted work.
2. Persist request/trace correlation fields needed after restart.
3. Version the payload schema.
4. Avoid storing secrets or unbounded baggage.

## Acceptance criteria

- [ ] A queued job can be reconstructed after process restart.
- [ ] Memory and PostgreSQL representations are semantically aligned.
- [ ] Migration rollback/forward behavior is tested.

## Required verification

- `go test ./internal/store/jobrepo/...`
- `go test ./internal/db/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
