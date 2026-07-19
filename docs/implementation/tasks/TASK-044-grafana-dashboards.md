# TASK-044 — Provision the ten planned Grafana dashboards as code

## Metadata

- Phase: `P3`
- Priority: `P1`
- Remediation group: `RG-SLO`
- Gap: `GAP-SLO-044`
- Dependencies: TASK-038, TASK-041, TASK-042
- Initial status: `BACKLOG`

## Goal

Provision the ten planned Grafana dashboards as code. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A06-G8: Grafana provisioning
- A07-SLO-9: No alert/dashboard-as-code artifacts exist

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `observability/grafana/provisioning/**`
- `observability/grafana/dashboards/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Create Service Overview, HTTP, Jobs, Outbound, gRPC, SSE, PostgreSQL, Runtime, Security and Telemetry dashboards.
2. Use recording rules for SLO panels.
3. Add trace/log links and useful drilldowns.
4. Keep dashboard files version-controlled and provisioned.

## Acceptance criteria

- [ ] Grafana starts with all dashboards present.
- [ ] Panels use bounded labels and correct units.
- [ ] At least one exemplar/trace drilldown works.

## Required verification

- `docker compose up -d grafana prometheus tempo loki`
- `find observability/grafana -type f`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
