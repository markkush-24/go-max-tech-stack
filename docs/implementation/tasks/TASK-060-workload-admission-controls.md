# TASK-060 — Split rate limits and bulkheads by workload, including SSE connections

## Metadata

- Phase: `P4`
- Priority: `P1`
- Remediation group: `RG-RESILIENCE`
- Gap: `GAP-RESILIENCE-060`
- Dependencies: TASK-032
- Initial status: `BACKLOG`

## Goal

Split rate limits and bulkheads by workload, including SSE connections. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A08-F12: Closing the hub before shutting down HTTP is useful but creates an admission race
- A09-F3: SSE occupies the shared bulkhead for the full stream lifetime
- A09-F4: One global bulkhead creates cross-route head-of-line and noisy-neighbor behavior
- A09-F5: One global rate limiter mixes different traffic classes
- A11-F20: Global admission controls enable cross-route noisy-neighbor failure

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/middleware/**`
- `internal/router/**`
- `internal/config/**`
- `internal/routes/job_handler.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Remove long-lived SSE from finite-request bulkhead.
2. Define separate controls for edge abuse, writes, Profile, gRPC bridge, async admission and SSE connections.
3. Choose limiter placement relative to authentication deliberately.
4. Emit policy-specific rejection metrics.

## Acceptance criteria

- [ ] One SSE client cannot reject unrelated API traffic.
- [ ] A slow Profile call cannot consume read capacity.
- [ ] Policies are independently configurable and tested.

## Required verification

- `go test -race ./internal/middleware/... ./internal/router/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
