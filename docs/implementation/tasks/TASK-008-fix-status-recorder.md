# TASK-008 — Fix first-status-only response recording with regression tests

## Metadata

- Phase: `P1`
- Priority: `P0`
- Remediation group: `RG-HTTP-CORRECTNESS`
- Gap: `GAP-HTTP-CORRECTNESS-008`
- Dependencies: TASK-007
- Initial status: `BACKLOG`

## Goal

Fix first-status-only response recording with regression tests. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A03-L6: HTTP status in logs and metrics can be incorrect after repeated `WriteHeader`
- A04-M3: Recorded HTTP status can disagree with the wire status
- A07-SLO-2: Current status attribution can corrupt availability

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/middleware/status_recorder.go`
- `internal/middleware/metrics.go`
- `internal/middleware/middleware.go`
- `internal/middleware/*test.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Record only the first effective `WriteHeader` status.
2. Implicit first `Write` must record 200.
3. Preserve optional interfaces needed by streaming or delegation.
4. Assert logs and metrics observe the same status sent on the wire.

## Acceptance criteria

- [ ] A second `WriteHeader` cannot overwrite the recorded status.
- [ ] Implicit 200 and explicit error statuses are correct.
- [ ] Existing middleware behavior remains compatible.

## Required verification

- `go test ./internal/middleware/...`
- `go test -race ./internal/middleware/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
