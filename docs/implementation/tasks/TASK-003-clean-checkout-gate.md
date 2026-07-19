# TASK-003 — Add clean-checkout and generated-drift verification

## Metadata

- Phase: `P0`
- Priority: `P0`
- Remediation group: `RG-REPRO`
- Gap: `GAP-REPRO-003`
- Dependencies: TASK-001, TASK-002
- Initial status: `BACKLOG`

## Goal

Add clean-checkout and generated-drift verification. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A01-R1: Clean checkout is not reproducible
- A01-R2: CI and local quality tooling are not committed
- A13-Q15: CI does not verify generated or dependency state

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `.github/workflows/**`
- `scripts/**`
- `Makefile`
- `Taskfile.yml`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Create a CI job that starts from a clean checkout.
2. Run dependency tidy/check, code generation and formatting.
3. Fail when generation or tidy leaves a diff.
4. Run the build/test command with the declared Go toolchain.

## Acceptance criteria

- [ ] CI detects a missing generated file.
- [ ] CI detects an uncommitted `go mod tidy` change.
- [ ] CI ends with a clean working tree.

## Required verification

- `git clean -xfd && git reset --hard HEAD  # only in disposable clone`
- `git diff --exit-code`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
