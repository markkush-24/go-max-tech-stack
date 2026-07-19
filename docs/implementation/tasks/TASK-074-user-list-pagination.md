# TASK-074 — Add bounded pagination to user-list endpoints

## Metadata

- Phase: `P5`
- Priority: `P1`
- Remediation group: `RG-PERFORMANCE`
- Gap: `GAP-PERFORMANCE-074`
- Dependencies: None
- Initial status: `BACKLOG`

## Goal

Add bounded pagination to user-list endpoints. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A09-F6: The user list path is unbounded

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/routes/users_handler*.go`
- `internal/service/**`
- `internal/store/userrepo/**`
- `migrations/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Define limit/cursor or limit/offset contract with a safe maximum.
2. Push limits into PostgreSQL queries.
3. Preserve deterministic ordering.
4. Document response metadata and validation errors.

## Acceptance criteria

- [ ] No list request can fetch all rows without an explicit bounded policy.
- [ ] Pagination is covered for memory and PostgreSQL repositories.
- [ ] Invalid/oversized page parameters are rejected.

## Required verification

- `go test ./internal/routes/... ./internal/store/userrepo/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
