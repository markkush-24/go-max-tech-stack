# P0-CORR-003 — Complete operational documentation and P0 provenance

## Goal

Make the repository self-contained for a developer following README and preserve the accepted P0 scope decisions inside repository documentation.

## Review findings

- P0-04-F1 — 23 supported ENV variables are missing from the canonical configuration table.
- P0-04-F2 — primary API curl examples omit required JWT authorization.
- P0-04-F3 — PostgreSQL default startup has no concrete migration command.
- P0-01 — dependency upgrades and manual dead-file cleanup were explained during review but not fully preserved as repository provenance.

## Allowed scope

- `README.md`
- `docs/**`
- `docs/implementation/COMPLETION_LOG.md`
- focused documentation consistency tests if already present

## Forbidden scope

- no runtime behavior changes;
- no migration runner implementation;
- no authentication redesign;
- no telemetry implementation.

## Requirements

1. Document every environment variable read by `config.Load`, including default/type/conditions.
2. Ensure the config documentation is derived or tested against the same 58-variable surface used by config tests.
3. Define `$AdminToken` and `$UserToken` before protected API examples.
4. Add appropriate `Authorization: Bearer ...` headers to every protected curl example.
5. Keep deliberately unauthorized examples explicit and labelled.
6. Add a concrete reproducible command to apply `migrations/*.up.sql` to the default Docker Compose PostgreSQL.
7. Add a command that verifies the `users` and `jobs` tables exist.
8. State who owns migrations in local and deployment workflows.
9. Record accepted P0 provenance notes:
   - pgx/transitive dependency updates were made to clear reproduced govulncheck findings;
   - removal of empty/unused `internal/db` files was manual cleanup outside the task-card scopes;
   - TASK-006 telemetry validation was N/A because telemetry config did not yet exist.

## Acceptance criteria

- no `config.Load` environment variable is undocumented;
- protected examples no longer produce an unexpected 401 when followed as written;
- a new developer can start PostgreSQL, apply schema and start the application using README only;
- P0 scope exceptions are visible in repository-owned history/documentation;
- no runtime/API behavior is changed.

## Required verification

```text
go test ./internal/config/... ./internal/router/...
go test -count=20 ./internal/config/...
git grep -n "Authorization: Bearer" README.md
git grep -n "migrations" README.md docs
git diff --check
```

Also run or statically verify the documented migration commands against the Docker Compose service/container names.

## Required completion report

Report:

- config variables added to documentation;
- authenticated examples corrected;
- exact migration commands;
- provenance notes added;
- changed files and verification results.
