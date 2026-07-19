# TASK-015 — Give the gRPC runtime single-start/single-stop ownership

## Metadata

- Phase: `P1`
- Priority: `P0`
- Remediation group: `RG-LIFECYCLE`
- Gap: `GAP-LIFECYCLE-015`
- Dependencies: TASK-007
- Initial status: `BACKLOG`

## Goal

Give the gRPC runtime single-start/single-stop ownership. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A10-F12: gRPC Runtime has no single-start/single-stop ownership

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/transport/grpcserver/runtime.go`
- `internal/transport/grpcserver/*test.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Represent runtime state explicitly.
2. Expose a stable done/error result to the application supervisor.
3. Make Shutdown idempotent with one graceful-stop attempt and forced fallback.
4. Reject or safely handle repeated Start.

## Acceptance criteria

- [ ] Fatal Serve errors are observable by the owner.
- [ ] Repeated Shutdown creates no extra goroutines.
- [ ] Runtime tests cover bind, serve failure and forced stop.

## Required verification

- `go test -race ./internal/transport/grpcserver/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
