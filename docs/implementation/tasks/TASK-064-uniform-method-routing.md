# TASK-064 — Unify ServeMux method registration, HEAD and Allow behavior

## Metadata

- Phase: `P5`
- Priority: `P1`
- Remediation group: `RG-HTTP-CONTRACT`
- Gap: `GAP-HTTP-CONTRACT-064`
- Dependencies: None
- Initial status: `BACKLOG`

## Goal

Unify ServeMux method registration, HEAD and Allow behavior. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A12-F4: HEAD and Allow contracts are inconsistent

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/router/**`
- `internal/routes/**`
- `internal/httpapi/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Use one consistent method-routing approach.
2. Define HEAD support for GET resources.
3. Ensure generated Allow matches actual capabilities.
4. Preserve `r.Pattern` and PathValue.

## Acceptance criteria

- [ ] Collection and item routes have consistent HEAD semantics.
- [ ] 405 responses expose correct Allow.
- [ ] RBAC remains based on matched patterns.

## Required verification

- `go test ./internal/router/... ./internal/routes/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
