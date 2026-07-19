# TASK-096 — Add lab-only mutex/block profiling and runtime capture protocol

## Metadata

- Phase: `P7`
- Priority: `P2`
- Remediation group: `RG-PERF`
- Gap: `GAP-PERF-096`
- Dependencies: TASK-088
- Initial status: `BACKLOG`

## Goal

Add lab-only mutex/block profiling and runtime capture protocol. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A09-F20: Block and mutex profiles are not explicitly enabled

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/runtimeinfo/**`
- `internal/config/**`
- `docs/performance/**`
- `scripts/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Add explicit lab-only controls for mutex and block profile rates.
2. Document CPU, heap, goroutine, mutex, block and trace capture windows.
3. Avoid always-on overhead without measurement.
4. Record profile metadata with each experiment.

## Acceptance criteria

- [ ] Contention profiles can be enabled intentionally.
- [ ] Default production-like mode remains low overhead.
- [ ] Capture commands are reproducible.

## Required verification

- `go test ./internal/runtimeinfo/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
