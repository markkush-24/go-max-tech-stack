# TASK-078 — Add PostgreSQL repository, migration and transaction integration tests

## Metadata

- Phase: `P6`
- Priority: `P0`
- Remediation group: `RG-QA`
- Gap: `GAP-QA-078`
- Dependencies: TASK-007
- Initial status: `BACKLOG`

## Goal

Add PostgreSQL repository, migration and transaction integration tests. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A13-Q6: PostgreSQL and migrations are untested

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/db/**/*_test.go`
- `internal/store/userrepo/**/*_test.go`
- `internal/store/jobrepo/**/*_test.go`
- `migrations/**`
- `docker-compose*.yml`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Start a disposable PostgreSQL test dependency.
2. Apply migrations and verify expected schema version.
3. Test uniqueness, CAS transitions, transaction commit/rollback and context timeout.
4. Test migration up/down or forward-only policy explicitly.

## Acceptance criteria

- [ ] Default PostgreSQL backend is exercised in CI.
- [ ] Repository semantics match memory backend where intended.
- [ ] Migration drift breaks the integration job.

## Required verification

- `go test ./internal/db/... ./internal/store/userrepo/... ./internal/store/jobrepo/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
