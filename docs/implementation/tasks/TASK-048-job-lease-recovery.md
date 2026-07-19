# TASK-048 — Add Job ownership, lease, attempt and transition version metadata

## Metadata

- Phase: `P4`
- Priority: `P0`
- Remediation group: `RG-ASYNC-DURABILITY`
- Gap: `GAP-ASYNC-DURABILITY-048`
- Dependencies: TASK-047
- Initial status: `BACKLOG`

## Goal

Add Job ownership, lease, attempt and transition version metadata. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A11-F3: Shared-database shutdown repair is unsafe for horizontal workers

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/entity/job.go`
- `internal/store/jobrepo/**`
- `migrations/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Add worker instance/lease owner and lease expiry.
2. Add attempt count and transition/version field.
3. Ensure one instance cannot repair another instance’s active lease.
4. Provide queries for claim, heartbeat and expired-work recovery.

## Acceptance criteria

- [ ] Two service instances cannot process the same active lease.
- [ ] Shutdown repair only affects locally owned jobs.
- [ ] Expired leases are discoverable.

## Required verification

- `go test -race ./internal/store/jobrepo/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
