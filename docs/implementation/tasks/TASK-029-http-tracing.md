# TASK-029 — Instrument inbound HTTP while preserving ServeMux route identity

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-OTEL`
- Gap: `GAP-OTEL-029`
- Dependencies: TASK-025, TASK-008
- Initial status: `BACKLOG`

## Goal

Instrument inbound HTTP while preserving ServeMux route identity. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A02-E6: Observability route identity is intentionally mux-derived
- A02-E8: Request context propagation is good on synchronous boundaries
- A05-T1: Distributed tracing is not implemented
- A05-T16: Route-aware span naming must preserve the existing low-cardinality invariant

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/middleware/**`
- `internal/router/**`
- `cmd/api/main.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Create server spans for inbound requests.
2. Set normalized span names/route attributes from final `r.Pattern`.
3. Do not put raw user/job IDs or query values in span names.
4. Record cancellation and server error status correctly.

## Acceptance criteria

- [ ] `GET /api/v1/users/123` is traced as the matched pattern.
- [ ] Incoming trace context is continued.
- [ ] 404/unmatched and streaming routes have defined naming behavior.

## Required verification

- `go test ./internal/middleware/... ./internal/router/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
