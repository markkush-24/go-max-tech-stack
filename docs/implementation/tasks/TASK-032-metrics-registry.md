# TASK-032 — Replace process-global metric ownership with a DI-owned registry

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-METRICS`
- Gap: `GAP-METRICS-032`
- Dependencies: TASK-025, TASK-008
- Initial status: `BACKLOG`

## Goal

Replace process-global metric ownership with a DI-owned registry. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A04-M1: `queue_depth` is permanently bound to the first `Queue` instance
- A04-M16: Metric test coverage is narrow and process-global state makes it fragile
- A04-M2: Process-global metric singletons break isolation and blur instance ownership
- A09-F17: Current expvar metrics add per-request string work and shared-map contention
- A13-Q11: Global `expvar` state prevents test isolation

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/metrics/**`
- `internal/queue/**`
- `internal/middleware/**`
- `internal/stream/**`
- `internal/testkit/**`
- `cmd/api/main.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Create per-application metric instruments through DI.
2. Allow isolated test readers/registries.
3. Keep `/debug/vars` only as a temporary compatibility/debug adapter.
4. Remove first-instance capture and global cross-test coupling.

## Acceptance criteria

- [ ] Two application instances expose independent metrics.
- [ ] Tests need no before/after global deltas.
- [ ] Queue depth observes the correct queue instance.

## Required verification

- `go test -race ./internal/metrics/... ./internal/queue/... ./internal/testkit/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
