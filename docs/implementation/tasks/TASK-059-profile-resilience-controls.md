# TASK-059 — Add Profile-specific connection bound, bulkhead, circuit breaker and retry budget

## Metadata

- Phase: `P4`
- Priority: `P1`
- Remediation group: `RG-OUTBOUND`
- Gap: `GAP-OUTBOUND-059`
- Dependencies: TASK-058
- Initial status: `BACKLOG`

## Goal

Add Profile-specific connection bound, bulkhead, circuit breaker and retry budget. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A09-F10: Outbound active connection count is unlimited per host
- A09-F11: Retry amplification is not bounded by a dependency-level concurrency policy
- A11-F12: No circuit breaker or retry budget protects the Profile dependency

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/outbound/httpclient/**`
- `internal/outbound/**`
- `internal/config/**`
- `cmd/api/main.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Set a finite MaxConnsPerHost aligned with dependency concurrency.
2. Add a dependency-specific bulkhead separate from global API admission.
3. Implement a configurable circuit-breaker state machine with bounded half-open probes.
4. Expose breaker/retry-budget metrics and logs.

## Acceptance criteria

- [ ] Slow upstream cannot consume unbounded connections.
- [ ] Open breaker fails fast with a typed error.
- [ ] Retries cannot exceed configured global/dependency budget.

## Required verification

- `go test -race ./internal/outbound/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
