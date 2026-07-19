# TASK-055 — Add durable Idempotency-Key behavior for create operations

## Metadata

- Phase: `P4`
- Priority: `P1`
- Remediation group: `RG-RESILIENCE`
- Gap: `GAP-RESILIENCE-055`
- Dependencies: TASK-054
- Initial status: `BACKLOG`

## Goal

Add durable Idempotency-Key behavior for create operations. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A11-F17: POST operations have no idempotency contract

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/routes/users_handler*.go`
- `internal/service/**`
- `internal/store/**`
- `migrations/**`
- `internal/httputils/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Validate and bound Idempotency-Key.
2. Persist request fingerprint and original result/status.
3. Return the original result for safe retries.
4. Reject key reuse with a different request body.

## Acceptance criteria

- [ ] Client retry after lost response does not create a second operation.
- [ ] Conflicting key reuse is detected.
- [ ] Sync and async policy is documented.

## Required verification

- `go test ./internal/routes/... ./internal/service/... ./internal/store/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
