# TASK-063 — Implement RFC-compliant Accept preference selection

## Metadata

- Phase: `P5`
- Priority: `P1`
- Remediation group: `RG-HTTP-CONTRACT`
- Gap: `GAP-HTTP-CONTRACT-063`
- Dependencies: None
- Initial status: `BACKLOG`

## Goal

Implement RFC-compliant Accept preference selection. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A12-F3: HTTP content negotiation violates Accept quality semantics

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/httputils/accept.go`
- `internal/routes/negotiation_test.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Parse media ranges and q weights.
2. Treat q=0 as unacceptable.
3. Respect specificity and deterministic preference ordering.
4. Return 406 when no representation is acceptable.

## Acceptance criteria

- [ ] JSON q=1 beats Protobuf q=0.
- [ ] Unsupported and all-q=0 requests return Problem 406.
- [ ] Wildcard behavior is documented and tested.

## Required verification

- `go test ./internal/httputils/... ./internal/routes/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
