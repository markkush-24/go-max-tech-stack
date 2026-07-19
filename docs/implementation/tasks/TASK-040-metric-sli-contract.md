# TASK-040 — Publish metric, cardinality and SLI classification contracts

## Metadata

- Phase: `P3`
- Priority: `P1`
- Remediation group: `RG-SLO`
- Gap: `GAP-SLO-040`
- Dependencies: TASK-034, TASK-035, TASK-036, TASK-037
- Initial status: `BACKLOG`

## Goal

Publish metric, cardinality and SLI classification contracts. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A04-M18: No standard exporter, resource identity or cardinality policy exists
- A05-T16: Route-aware span naming must preserve the existing low-cardinality invariant
- A05-T20: There is no documented propagation and privacy contract
- A06-G12: Signal cardinality contract
- A07-SLO-1: Current data cannot support trustworthy SLOs
- A07-SLO-3: Service-class mapping is not formally encoded

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `docs/observability/METRIC_CONTRACT.md`
- `docs/observability/SLI_SLO.md`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. List each instrument, unit, type and bounded attributes.
2. Define eligible/good/bad/excluded events by service class.
3. Document identifier and cardinality prohibitions.
4. Mark initial SLO targets as provisional until baseline experiments.

## Acceptance criteria

- [ ] Every dashboard/alert metric has a documented contract.
- [ ] New routes cannot silently bypass service-class mapping.
- [ ] All identifiers have an explicit logs/traces/metrics policy.

## Required verification

- `test -f docs/observability/METRIC_CONTRACT.md`
- `test -f docs/observability/SLI_SLO.md`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
