# TASK-039 — Ship JSON slog logs through Alloy to Loki with trace correlation

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-OBS-STACK`
- Gap: `GAP-OBS-STACK-039`
- Dependencies: TASK-024, TASK-028, TASK-038
- Initial status: `BACKLOG`

## Goal

Ship JSON slog logs through Alloy to Loki with trace correlation. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A06-G7: Loki/log shipping

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `observability/alloy/**`
- `observability/loki/**`
- `observability/grafana/**`
- `internal/config/**`
- `cmd/api/main.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Use JSON logs for the central pipeline.
2. Ship stdout/container logs through a bounded agent pipeline.
3. Configure trace-to-logs and logs-to-traces correlation.
4. Do not index high-cardinality request/trace/user/job IDs as labels.

## Acceptance criteria

- [ ] A trace can navigate to related logs in Grafana.
- [ ] Loki labels remain bounded.
- [ ] Collector/Loki outage does not block request handling.

## Required verification

- `docker compose config`
- `go test ./internal/config/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
