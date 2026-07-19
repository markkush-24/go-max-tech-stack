# TASK-054 — Make user creation and Job completion atomic or idempotently recoverable

## Metadata

- Phase: `P4`
- Priority: `P0`
- Remediation group: `RG-ASYNC-DURABILITY`
- Gap: `GAP-ASYNC-DURABILITY-054`
- Dependencies: TASK-050, TASK-010
- Initial status: `BACKLOG`

## Goal

Make user creation and Job completion atomic or idempotently recoverable. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A11-F2: User creation and job terminal success are not atomic
- A11-F23: `WithinTransaction` exists but is not integrated into application operations

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/db/tx.go`
- `internal/service/**`
- `internal/store/userrepo/**`
- `internal/store/jobrepo/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Use a transaction where both repositories share PostgreSQL, or define an idempotent recovery protocol.
2. Handle user-created/job-not-finalized failures.
3. Store result identity safely.
4. Test commit, rollback and retry after partial failure.

## Acceptance criteria

- [ ] A successful user create cannot leave an unrecoverable running job.
- [ ] Retry does not create duplicate users.
- [ ] Partial-commit scenarios have deterministic outcomes.

## Required verification

- `go test ./internal/db/... ./internal/store/userrepo/... ./internal/store/jobrepo/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
