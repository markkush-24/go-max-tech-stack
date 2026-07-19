# TASK-049 — Choose transactional async acceptance and outbox semantics

## Metadata

- Phase: `P4`
- Priority: `P0`
- Remediation group: `RG-ASYNC-DURABILITY`
- Gap: `GAP-ASYNC-DURABILITY-049`
- Dependencies: TASK-048
- Initial status: `BACKLOG`

## Goal

Choose transactional async acceptance and outbox semantics. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A11-F21: Queue overload response lacks durable admission semantics
- A11-F4: Queue-full compensation can create orphan jobs and amplify DB load

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `docs/adr/**`
- `docs/implementation/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Decide how DB job persistence and queue/broker publication become reliable.
2. Define 202 acceptance point and failure semantics.
3. Define future broker migration boundary.
4. Define duplicate publication/consumption behavior.

## Acceptance criteria

- [ ] The ADR resolves insert-then-delete compensation.
- [ ] Accepted means recoverable after crash.
- [ ] The design supports at-least-once delivery.

## Required verification

- `test -f docs/adr/*async* || true`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
