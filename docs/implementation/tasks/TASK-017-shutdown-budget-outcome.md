# TASK-017 — Implement one global shutdown budget and truthful outcome model

## Metadata

- Phase: `P1`
- Priority: `P0`
- Remediation group: `RG-LIFECYCLE`
- Gap: `GAP-LIFECYCLE-017`
- Dependencies: TASK-016, TASK-012
- Initial status: `BACKLOG`

## Goal

Implement one global shutdown budget and truthful outcome model. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A02-E1: Startup ownership is fragmented
- A02-E2: Shutdown paths are asymmetric
- A08-F14: Component stop order is not explicitly tied to telemetry completion
- A08-F6: Forced HTTP shutdown is reported as success
- A08-F7: gRPC timeout is deliberately suppressed by `APIServer.Run`
- A08-F8: Shutdown has no global deadline

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/api/**`
- `cmd/api/main.go`
- `internal/workerpool/**`
- `internal/stream/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Create one global shutdown deadline and derive component sub-budgets.
2. Stop independent HTTP listeners concurrently where safe.
3. Distinguish graceful, forced, timed-out and failed component outcomes.
4. Ensure cleanup and final result complete before `main` can call `os.Exit`.

## Acceptance criteria

- [ ] Total shutdown respects the configured global budget.
- [ ] Forced HTTP/gRPC termination is not reported as graceful success.
- [ ] Signal and fatal-error shutdown produce deterministic exit results.

## Required verification

- `go test ./internal/api/...`
- `go test -race ./internal/api/... ./internal/workerpool/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
