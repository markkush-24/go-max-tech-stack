# TASK-095 — Test graceful and forced shutdown under active HTTP, gRPC, jobs and SSE

## Metadata

- Phase: `P7`
- Priority: `P1`
- Remediation group: `RG-PERF`
- Gap: `GAP-PERF-095`
- Dependencies: TASK-017, TASK-046, TASK-089
- Initial status: `BACKLOG`

## Goal

Test graceful and forced shutdown under active HTTP, gRPC, jobs and SSE. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A08-F16: Current tests do not cover shutdown outcome truth
- A11-F24: Fault-injection coverage is insufficient

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `loadtest/**`
- `docs/performance/**`
- `internal/api/**/*_test.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Initiate shutdown during mixed traffic.
2. Measure drain duration, forced terminations, job repair and telemetry flush.
3. Test fatal component failure as well as signal shutdown.
4. Verify exit code and final state.

## Acceptance criteria

- [ ] Shutdown completes within global budget.
- [ ] Forced outcomes are reported truthfully.
- [ ] No accepted durable job is lost.

## Required verification

- `go test ./internal/api/...`
- `test -d loadtest`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
