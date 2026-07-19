# TASK-085 — Provide cross-platform quality commands and drift/security gates

## Metadata

- Phase: `P6`
- Priority: `P1`
- Remediation group: `RG-QA`
- Gap: `GAP-QA-085`
- Dependencies: TASK-003, TASK-080
- Initial status: `BACKLOG`

## Goal

Provide cross-platform quality commands and drift/security gates. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A01-R2: CI and local quality tooling are not committed
- A13-Q1: The quality system is not part of the tracked repository
- A13-Q15: CI does not verify generated or dependency state
- A13-Q17: Quality scripts are Windows-only
- A13-Q18: CI tool installation and workflow integrity can be hardened
- A13-Q19: Vulnerability scanning covers only the built application binary

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `Makefile`
- `Taskfile.yml`
- `scripts/**`
- `.github/workflows/**`
- `tools/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Provide one cross-platform entrypoint for format, test, race, vet, staticcheck, vulnerability scan, generate and OpenAPI validation.
2. Run `go mod tidy` and generation diff gates.
3. Validate migrations.
4. Scan modules/source and built artifacts according to the chosen tool.

## Acceptance criteria

- [ ] Linux CI does not depend on PowerShell-only scripts.
- [ ] One documented command runs the local quality suite.
- [ ] Drift and vulnerability failures are actionable.

## Required verification

- `make help || task --list || true`
- `git diff --exit-code`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
