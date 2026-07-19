# TASK-007 — Establish the Go 1.25.12 verification baseline

## Metadata

- Phase: `P0`
- Priority: `P0`
- Remediation group: `RG-QA`
- Gap: `GAP-QA-007`
- Dependencies: TASK-003
- Initial status: `BACKLOG`

## Goal

Establish the Go 1.25.12 verification baseline. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A01-R7: Full automated verification unavailable in this audit environment

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `go.mod`
- `.github/workflows/**`
- `docs/implementation/**`
- `README.md`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Run the full project with the declared Go 1.25.12 toolchain.
2. Record build, test, race, vet and vulnerability-scan results.
3. Do not change the target toolchain merely to satisfy an older local runtime.
4. Persist the baseline commands in the repository.

## Acceptance criteria

- [ ] `go test ./...` succeeds on Go 1.25.12.
- [ ] Race/vet results are recorded or explicitly blocked with evidence.
- [ ] The CI toolchain matches `go.mod`.

## Required verification

- `go version`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
