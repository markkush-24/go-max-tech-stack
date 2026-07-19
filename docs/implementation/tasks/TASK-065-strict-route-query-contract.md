# TASK-065 — Resolve v2 item surface, unknown-route precedence and strict query parsing

## Metadata

- Phase: `P5`
- Priority: `P1`
- Remediation group: `RG-HTTP-CONTRACT`
- Gap: `GAP-HTTP-CONTRACT-065`
- Dependencies: TASK-064
- Initial status: `BACKLOG`

## Goal

Resolve v2 item surface, unknown-route precedence and strict query parsing. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A01-R5: README/config drift around storage defaults
- A01-R6: Current context/documentation and actual route surface differ
- A12-F21: v2 contains dead item-handler code but no item endpoint contract
- A12-F22: Unknown-route behavior contradicts the deny-by-default policy comment
- A12-F23: Query parameter contract is permissive and undocumented

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/router/**`
- `internal/routes/users_handler_v2.go`
- `internal/routes/users_handler.go`
- `README.md`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Choose whether v2 item endpoint exists and remove dead drift.
2. Document route-existence/authentication precedence.
3. Validate `async` values and unknown query parameters according to policy.
4. Update tests and docs to the chosen contract.

## Acceptance criteria

- [ ] No dead handler remains without a contract.
- [ ] Malformed query values do not silently change behavior.
- [ ] Unknown/unsupported route behavior is consistent and tested.

## Required verification

- `go test ./internal/router/... ./internal/routes/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
