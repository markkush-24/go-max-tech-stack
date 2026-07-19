# TASK-083 — Add coverage reporting and risk-based CI gates

## Metadata

- Phase: `P6`
- Priority: `P1`
- Remediation group: `RG-QA`
- Gap: `GAP-QA-083`
- Dependencies: TASK-077, TASK-078
- Initial status: `BACKLOG`

## Goal

Add coverage reporting and risk-based CI gates. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A13-Q10: No coverage report or coverage policy exists

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `.github/workflows/**`
- `scripts/**`
- `docs/quality/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Produce coverage profiles and summaries.
2. Set package-focused expectations for critical concurrency/security/persistence code rather than a misleading single global percentage.
3. Upload or retain reports.
4. Keep race, vet and static analysis in separate clear jobs.

## Acceptance criteria

- [ ] CI exposes untested critical packages.
- [ ] Coverage policy is documented and reviewable.
- [ ] Coverage collection does not replace behavioral acceptance tests.

## Required verification

- `go test -coverprofile=coverage.out ./...`
- `go tool cover -func=coverage.out`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
