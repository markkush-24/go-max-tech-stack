# TASK-010 — Enforce Job CAS transitions in memory and PostgreSQL repositories

## Metadata

- Phase: `P1`
- Priority: `P0`
- Remediation group: `RG-JOB-CORRECTNESS`
- Gap: `GAP-JOB-CORRECTNESS-010`
- Dependencies: TASK-009
- Initial status: `BACKLOG`

## Goal

Enforce Job CAS transitions in memory and PostgreSQL repositories. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A10-F10: Job transitions consume work without guaranteed terminality
- A10-F2: Active worker can overwrite shutdown terminal state

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

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

1. Memory updates check the expected source state while holding the write lock.
2. SQL updates include expected status/version in `WHERE`.
3. Distinguish not-found from transition conflict.
4. Add concurrent finalization and immutable-terminal tests for both backends.

## Acceptance criteria

- [ ] Only one competing terminal transition wins.
- [ ] `failed -> succeeded` and `succeeded -> failed` are rejected.
- [ ] Memory and PostgreSQL semantics match.

## Required verification

- `go test -race ./internal/store/jobrepo/...`
- `go test ./internal/store/jobrepo/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
