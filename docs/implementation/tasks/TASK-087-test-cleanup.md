# TASK-087 — Harden configuration and representation tests

## Metadata

- Phase: `P6`
- Priority: `P2`
- Remediation group: `RG-QA`
- Gap: `GAP-QA-087`
- Dependencies: TASK-006, TASK-084
- Initial status: `BACKLOG`

## Goal

Harden configuration and representation tests. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A13-Q13: Several tests assert brittle representation details
- A13-Q14: Configuration tests are environment-sensitive and incomplete

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/config/config_test.go`
- `internal/httputils/**/*_test.go`
- `internal/routes/**/*_test.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Remove dependence on undeclared environment.
2. Prefer semantic assertions over exact incidental encoding/order.
3. Add table cases for documented defaults and invalid combinations.
4. Keep golden files only where representation stability is intentional.

## Acceptance criteria

- [ ] Tests remain stable across harmless formatting changes.
- [ ] Host environment cannot change expected defaults.
- [ ] Intentional wire contracts remain explicit.

## Required verification

- `go test -count=20 ./internal/config/... ./internal/httputils/... ./internal/routes/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
