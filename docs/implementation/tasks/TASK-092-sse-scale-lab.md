# TASK-092 — Add SSE connection, fan-out and slow-client experiments

## Metadata

- Phase: `P7`
- Priority: `P1`
- Remediation group: `RG-PERF`
- Gap: `GAP-PERF-092`
- Dependencies: TASK-072, TASK-037, TASK-088
- Initial status: `BACKLOG`

## Goal

Add SSE connection, fan-out and slow-client experiments. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A09-F13: SSE fan-out uses one global mutex across all job IDs
- A09-F14: Every SSE connection owns a goroutine, ticker and socket
- A09-F15: SSE is excluded from ordinary HTTP metrics but not from resource accounting
- A10-F17: Stream Hub synchronization is currently sound for tested send/close paths

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `loadtest/**`
- `docs/performance/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Sweep concurrent connections and subscribers per job.
2. Include slow/non-reading clients and reconnects.
3. Measure goroutines, FDs, heap, drops, write timeouts and Hub contention.
4. Verify finite API capacity remains isolated.

## Acceptance criteria

- [ ] One SSE workload cannot starve ordinary API requests.
- [ ] Drop/resync behavior matches the contract.
- [ ] The practical connection limit is documented.

## Required verification

- `test -d loadtest`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
