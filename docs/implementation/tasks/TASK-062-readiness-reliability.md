# TASK-062 — Add schema-aware, redacted and independently budgeted readiness checks

## Metadata

- Phase: `P4`
- Priority: `P1`
- Remediation group: `RG-HEALTH`
- Gap: `GAP-HEALTH-062`
- Dependencies: TASK-061
- Initial status: `BACKLOG`

## Goal

Add schema-aware, redacted and independently budgeted readiness checks. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A02-E9: Readiness has one shared sequential budget
- A05-T18: DB/startup operations are only partly connected to application cancellation
- A11-F18: Startup readiness validates connectivity, not schema or migrations
- A11-F19: Readiness checks are sequential under one shared 200 ms deadline
- A12-F13: Public readiness responses expose raw internal errors

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/health/**`
- `internal/db/health.go`
- `internal/db/**`
- `migrations/**`
- `internal/config/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Check required schema/migration version, not only DB ping.
2. Return redacted public check states while logging internal cause.
3. Run checks with independent or concurrent budgets.
4. Define startup retry/orchestration behavior for temporarily unavailable DB.

## Acceptance criteria

- [ ] Reachable but incompatible DB keeps readiness false.
- [ ] Public `/readyz` does not leak host/schema error strings.
- [ ] One slow check does not prevent all other checks from reporting.

## Required verification

- `go test ./internal/health/... ./internal/db/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
