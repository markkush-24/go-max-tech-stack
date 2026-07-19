# TASK-051 — Implement startup reconciliation and expired-work recovery

## Metadata

- Phase: `P4`
- Priority: `P0`
- Remediation group: `RG-ASYNC-DURABILITY`
- Gap: `GAP-ASYNC-DURABILITY-051`
- Dependencies: TASK-050, TASK-048
- Initial status: `BACKLOG`

## Goal

Implement startup reconciliation and expired-work recovery. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A11-F1: Accepted async work is not durable or crash-recoverable
- A11-F22: Job model lacks recovery metadata

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/service/**`
- `internal/store/jobrepo/**`
- `internal/workerpool/**`
- `cmd/api/main.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Scan recoverable queued and expired running jobs at startup.
2. Claim work using lease/version CAS.
3. Avoid processing terminal or actively leased jobs.
4. Expose reconciliation outcomes in logs and metrics.

## Acceptance criteria

- [ ] A crash-restart test completes previously accepted work.
- [ ] Two instances do not both claim the same job.
- [ ] Poison/repeated failures are not retried forever.

## Required verification

- `go test -race ./internal/store/jobrepo/... ./internal/workerpool/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
