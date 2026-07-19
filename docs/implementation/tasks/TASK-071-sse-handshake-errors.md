# TASK-071 — Commit SSE immediately and handle post-commit errors without Problem corruption

## Metadata

- Phase: `P5`
- Priority: `P0`
- Remediation group: `RG-SSE`
- Gap: `GAP-SSE-071`
- Dependencies: TASK-060, TASK-008
- Initial status: `BACKLOG`

## Goal

Commit SSE immediately and handle post-commit errors without Problem corruption. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A02-E7: Streaming has a separate telemetry path
- A08-F13: SSE timing has a boundary race with server `WriteTimeout`
- A11-F8: Streaming errors are routed through Problem+JSON after the stream may be committed
- A12-F17: SSE does not establish the stream immediately
- A12-F19: Streaming errors can corrupt an already committed response

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/routes/job_handler.go`
- `internal/httputils/apphandler.go`
- `internal/routes/streaming_range_test.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Send an initial comment/snapshot and Flush after successful authorization/subscription.
2. Distinguish pre-commit handler errors from post-commit stream failures.
3. Never append Problem JSON to an active SSE stream.
4. Record disconnect/write/flush outcome through logs and metrics.

## Acceptance criteria

- [ ] Client receives stream confirmation immediately.
- [ ] Write/flush failure closes the stream without a second HTTP response.
- [ ] Wire status, logs and metrics remain consistent.

## Required verification

- `go test ./internal/routes/... ./internal/httputils/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
