# TASK-056 — Contain or supervise worker panics with durable job outcome

## Metadata

- Phase: `P4`
- Priority: `P0`
- Remediation group: `RG-RESILIENCE`
- Gap: `GAP-RESILIENCE-056`
- Dependencies: TASK-053
- Initial status: `BACKLOG`

## Goal

Contain or supervise worker panics with durable job outcome. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A11-F6: Worker panics are not contained or supervised

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/workerpool/**`
- `internal/service/jobService.go`
- `internal/metrics/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Define whether a job panic is recovered per item or crashes the supervised process.
2. If recovered, record stack, terminal/retry outcome and metrics.
3. Do not silently continue after unknown corruption.
4. Add panic injection tests.

## Acceptance criteria

- [ ] A worker panic has a deterministic documented outcome.
- [ ] The job is not left permanently running without evidence.
- [ ] Supervisor behavior is tested.

## Required verification

- `go test -race ./internal/workerpool/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
