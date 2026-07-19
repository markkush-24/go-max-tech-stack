# TASK-005 — Align runtime defaults, README and actual route surface

## Metadata

- Phase: `P0`
- Priority: `P1`
- Remediation group: `RG-DOCS`
- Gap: `GAP-DOCS-005`
- Dependencies: None
- Initial status: `BACKLOG`

## Goal

Align runtime defaults, README and actual route surface. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A01-R3: Uploaded project contains a large uncommitted persistence change
- A01-R5: README/config drift around storage defaults
- A01-R6: Current context/documentation and actual route surface differ

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/config/**`
- `cmd/api/main.go`
- `internal/router/**`
- `README.md`
- `docs/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Choose the canonical default storage backend and document it.
2. Make README environment defaults match code.
3. Document only routes that are actually registered.
4. Mark optional integrations and their enabling conditions.

## Acceptance criteria

- [ ] A developer following README starts the intended backend.
- [ ] Documented routes match router registration.
- [ ] No contradictory default values remain.

## Required verification

- `go test ./internal/config/... ./internal/router/...`
- `git grep -n "STORAGE_BACKEND\|DB_DSN" README.md internal/config`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
