# TASK-014 — Correct async transition ordering and use one Event Hub in testkit

## Metadata

- Phase: `P1`
- Priority: `P0`
- Remediation group: `RG-STREAM-CORRECTNESS`
- Gap: `GAP-STREAM-CORRECTNESS-014`
- Dependencies: TASK-010, TASK-013
- Initial status: `BACKLOG`

## Goal

Correct async transition ordering and use one Event Hub in testkit. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A04-M17: `testkit` constructs disconnected stream hubs while metrics aggregate them globally
- A10-F14: Testkit wires two different event hubs
- A10-F9: Queued events and metrics can appear after running/succeeded
- A13-Q4: `testkit` wires producers and SSE consumers to different Hubs

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/routes/users_handler*.go`
- `internal/workerpool/**`
- `internal/stream/**`
- `internal/testkit/**`
- `internal/routes/*test.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Ensure queued cannot be published after running/terminal transitions.
2. Use the same Hub for producers and SSE consumers in testkit.
3. Add a true async-create-to-SSE full-path test without manual publish.
4. Keep worker publication non-blocking.

## Acceptance criteria

- [ ] Observed transition order is monotonic.
- [ ] Testkit topology matches production topology.
- [ ] The end-to-end test fails if producer and consumer hubs differ.

## Required verification

- `go test -race ./internal/routes/... ./internal/stream/... ./internal/testkit/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
