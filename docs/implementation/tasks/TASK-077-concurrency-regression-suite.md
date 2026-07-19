# TASK-077 — Commit regression tests for queue, worker, Job state and metrics races

## Metadata

- Phase: `P6`
- Priority: `P0`
- Remediation group: `RG-QA`
- Gap: `GAP-QA-077`
- Dependencies: TASK-012, TASK-010, TASK-033
- Initial status: `BACKLOG`

## Goal

Commit regression tests for queue, worker, Job state and metrics races. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A10-F19: Full concurrency proof is still unavailable
- A13-Q20: Previously discovered defects lack regression tests
- A13-Q5: The highest-risk concurrency packages have no direct tests

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/queue/**/*_test.go`
- `internal/workerpool/**/*_test.go`
- `internal/store/jobrepo/**/*_test.go`
- `internal/metrics/**/*_test.go`
- `internal/stream/**/*_test.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Cover canceled enqueue, Stop barrier, repeated Stop, timeout/restart, terminal overwrite and queue gauge ownership.
2. Use synchronization hooks/channels instead of sleeps.
3. Run under race detector.
4. Preserve positive Hub/repository locking invariants.

## Acceptance criteria

- [ ] Every critical concurrency finding has a committed regression test.
- [ ] Tests fail on the audited broken behavior.
- [ ] Race run is deterministic.

## Required verification

- `go test -race ./internal/queue/... ./internal/workerpool/... ./internal/store/jobrepo/... ./internal/stream/... ./internal/metrics/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
