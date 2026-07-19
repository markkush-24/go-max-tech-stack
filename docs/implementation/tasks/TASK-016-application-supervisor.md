# TASK-016 — Introduce one application supervisor for HTTP, HTTPS, gRPC, workers and streams

## Metadata

- Phase: `P1`
- Priority: `P0`
- Remediation group: `RG-LIFECYCLE`
- Gap: `GAP-LIFECYCLE-016`
- Dependencies: TASK-011, TASK-015
- Initial status: `BACKLOG`

## Goal

Introduce one application supervisor for HTTP, HTTPS, gRPC, workers and streams. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A02-E1: Startup ownership is fragmented
- A02-E2: Shutdown paths are asymmetric
- A05-T18: DB/startup operations are only partly connected to application cancellation
- A08-F1: There is no unified application supervisor
- A08-F12: Closing the hub before shutting down HTTP is useful but creates an admission race
- A08-F2: Worker and gRPC components start before construction is complete
- A08-F3: `os.Exit(1)` makes deferred cleanup in `main` unusable
- A08-F4: Unexpected HTTP/HTTPS server error cleanup is asymmetric
- A08-F5: A gRPC serve failure can produce a successful process exit
- A10-F13: APIServer and component lifecycle remain path-dependent

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `cmd/api/main.go`
- `internal/api/**`
- `internal/workerpool/**`
- `internal/transport/grpcserver/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Construct components before starting background goroutines.
2. Collect fatal component errors through one supervisor.
3. Use the same cleanup path for signal and component failure.
4. Return non-zero failure information for required component crashes.

## Acceptance criteria

- [ ] A gRPC Serve failure cannot result in successful process exit.
- [ ] Unexpected HTTP failure stops all owned components.
- [ ] No component-specific cleanup remains reachable only on one path.

## Required verification

- `go test ./internal/api/... ./internal/transport/grpcserver/...`
- `go test -race ./internal/api/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
