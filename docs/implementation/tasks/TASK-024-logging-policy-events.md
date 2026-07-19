# TASK-024 — Add redaction policy and coherent security, retry, SSE and shutdown events

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-LOGGING`
- Gap: `GAP-LOGGING-024`
- Dependencies: TASK-023
- Initial status: `BACKLOG`

## Goal

Add redaction policy and coherent security, retry, SSE and shutdown events. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A03-L10: SSE lifecycle is absent from logs
- A03-L11: Lifecycle logs do not provide a complete shutdown narrative
- A03-L13: Access logging can become noisy for probes and long-lived streams
- A03-L14: Unexpected errors may produce two records without an explicit relationship
- A03-L7: Privacy and redaction policy is not defined or enforced
- A03-L8: Security decisions have counters but no safe audit events
- A03-L9: Retry behavior is not visible as a coherent operation
- A05-T20: There is no documented propagation and privacy contract

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/middleware/**`
- `internal/outbound/**`
- `internal/routes/job_handler.go`
- `internal/api/**`
- `docs/observability/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Document allowed/forbidden log fields and PII handling.
2. Emit normalized authn/authz/CORS decision reasons without secrets.
3. Log logical retry operation with attempt/backoff/final outcome.
4. Log SSE connection lifecycle and one shutdown summary without per-event noise.

## Acceptance criteria

- [ ] No token, secret or body is logged.
- [ ] Retry attempts can be grouped into one operation.
- [ ] Shutdown outcome and duration are visible.

## Required verification

- `go test ./internal/middleware/... ./internal/outbound/... ./internal/api/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
