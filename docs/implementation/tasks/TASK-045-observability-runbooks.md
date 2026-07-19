# TASK-045 — Add alert runbooks and load-experiment annotation policy

## Metadata

- Phase: `P3`
- Priority: `P2`
- Remediation group: `RG-SLO`
- Gap: `GAP-SLO-045`
- Dependencies: TASK-043, TASK-044
- Initial status: `BACKLOG`

## Goal

Add alert runbooks and load-experiment annotation policy. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A07-SLO-10: Low traffic requires explicit guardrails
- A07-SLO-11: Operational and user-facing alerts must be separated
- A07-SLO-9: No alert/dashboard-as-code artifacts exist

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `docs/observability/RUNBOOKS.md`
- `docs/observability/ALERT_POLICY.md`
- `scripts/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Document first checks, likely causes and safe mitigations for each alert family.
2. Define annotation and silence procedure for intentional load/failure tests.
3. Do not weaken SLO thresholds merely because a chaos test is intentional.
4. Record evidence to preserve after experiments.

## Acceptance criteria

- [ ] Every page alert has a runbook anchor.
- [ ] Load sessions can be annotated reproducibly.
- [ ] Silence policy distinguishes expected lab alarms from real failures.

## Required verification

- `git grep -n "runbook" observability docs/observability`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
