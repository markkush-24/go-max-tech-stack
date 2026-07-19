# TASK-082 — Commit focused benchmarks for hot paths and allocations

## Metadata

- Phase: `P6`
- Priority: `P1`
- Remediation group: `RG-QA`
- Gap: `GAP-QA-082`
- Dependencies: TASK-008, TASK-032
- Initial status: `BACKLOG`

## Goal

Commit focused benchmarks for hot paths and allocations. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A09-F1: No persistent benchmark or load-test suite exists
- A13-Q9: There are no committed benchmarks or load scripts

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/**/*_test.go`
- `benchmarks/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Benchmark queue, Hub fan-out, JSON/Protobuf, Problem writing, auth middleware, ETag and representative handlers.
2. Report allocations.
3. Include telemetry enabled/disabled variants where feasible.
4. Do not optimize before recording a baseline.

## Acceptance criteria

- [ ] Benchmarks run with one documented command.
- [ ] Results include ns/op and allocs/op.
- [ ] Critical hot paths have stable names for benchstat comparison.

## Required verification

- `go test -bench=. -benchmem ./internal/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
