# TASK-006 — Make configuration tests independent of host environment

## Metadata

- Phase: `P0`
- Priority: `P1`
- Remediation group: `RG-QA`
- Gap: `GAP-QA-006`
- Dependencies: None
- Initial status: `BACKLOG`

## Goal

Make configuration tests independent of host environment. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A13-Q14: Configuration tests are environment-sensitive and incomplete

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/config/config_test.go`
- `internal/config/**`
- `internal/testkit/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Clear or explicitly set every environment variable read by configuration tests.
2. Use `t.Setenv` and table-driven cases.
3. Cover invalid duration, integer, URL, JWT, CORS, proxy, DB and telemetry values.
4. Do not rely on developer machine defaults.

## Acceptance criteria

- [ ] Config tests pass with hostile unrelated environment variables set.
- [ ] Defaults and validation branches are explicitly covered.
- [ ] Tests can run in any order.

## Required verification

- `go test -count=20 ./internal/config/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
