# TASK-076 — Create production-faithful and explicitly scoped testkit fixtures

## Metadata

- Phase: `P6`
- Priority: `P1`
- Remediation group: `RG-QA`
- Gap: `GAP-QA-076`
- Dependencies: TASK-016, TASK-014
- Initial status: `BACKLOG`

## Goal

Create production-faithful and explicitly scoped testkit fixtures. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A04-M17: `testkit` constructs disconnected stream hubs while metrics aggregate them globally
- A10-F14: Testkit wires two different event hubs
- A13-Q3: `testkit.NewServer` is not production full-stack
- A13-Q4: `testkit` wires producers and SSE consumers to different Hubs

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/testkit/**`
- `internal/router/**/*_test.go`
- `internal/routes/**/*_test.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Separate router-only, middleware-stack and full-runtime fixtures clearly.
2. Do not inject admin authorization implicitly unless requested.
3. Use the same Hub and dependency graph as production.
4. Offer options for JWT, proxy, worker pool, gRPC, TLS and PostgreSQL where appropriate.

## Acceptance criteria

- [ ] A test cannot accidentally bypass auth because of helper defaults.
- [ ] Full-runtime fixture exercises APIServer lifecycle.
- [ ] Fixture names communicate their fidelity.

## Required verification

- `go test ./internal/testkit/... ./internal/router/... ./internal/routes/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
