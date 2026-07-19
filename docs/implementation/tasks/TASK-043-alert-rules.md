# TASK-043 — Add multi-window burn-rate and diagnostic alerts

## Metadata

- Phase: `P3`
- Priority: `P1`
- Remediation group: `RG-SLO`
- Gap: `GAP-SLO-043`
- Dependencies: TASK-041, TASK-042
- Initial status: `BACKLOG`

## Goal

Add multi-window burn-rate and diagnostic alerts. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A07-SLO-10: Low traffic requires explicit guardrails
- A07-SLO-11: Operational and user-facing alerts must be separated
- A07-SLO-9: No alert/dashboard-as-code artifacts exist

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `observability/prometheus/rules/**`
- `observability/grafana/provisioning/alerting/**`
- `docs/observability/ALERT_POLICY.md`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Add multi-window burn-rate alerts with low-traffic guards.
2. Separate user-symptom pages from component diagnostics.
3. Add queue, DB, outbound, SSE, gRPC and telemetry-pipeline alerts.
4. Attach stable labels, severity and runbook references.

## Acceptance criteria

- [ ] Rules are syntactically valid.
- [ ] A low-volume single error does not page without the guard condition.
- [ ] Every page-level alert maps to an SLO symptom.

## Required verification

- `promtool check rules observability/prometheus/rules/*.yml`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
