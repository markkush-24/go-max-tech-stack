# TASK-079 — Create the machine-readable OpenAPI source of truth

## Metadata

- Phase: `P6`
- Priority: `P1`
- Remediation group: `RG-OPENAPI`
- Gap: `GAP-OPENAPI-079`
- Dependencies: TASK-063, TASK-065, TASK-070, TASK-066
- Initial status: `BACKLOG`

## Goal

Create the machine-readable OpenAPI source of truth. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A12-F24: No machine-readable HTTP contract exists
- A13-Q7: No OpenAPI or runtime contract test exists

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `api/openapi/**`
- `docs/api/**`
- `internal/router/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Describe v1/v2 users, jobs, Profile, async mode, Range and SSE.
2. Describe JWT security, Problem variants, request ID, ETag and exposed headers.
3. Keep debug/internal endpoints out unless explicitly intended.
4. Use examples that match real responses.

## Acceptance criteria

- [ ] The spec validates.
- [ ] Every registered public HTTP route has an intentional disposition.
- [ ] Status/content types and security match tests.

## Required verification

- `test -f api/openapi/openapi.yaml`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
