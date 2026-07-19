# TASK-041 — Implement HTTP SLO recording rules

## Metadata

- Phase: `P3`
- Priority: `P1`
- Remediation group: `RG-SLO`
- Gap: `GAP-SLO-041`
- Dependencies: TASK-040, TASK-038
- Initial status: `BACKLOG`

## Goal

Implement HTTP SLO recording rules. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A07-SLO-2: Current status attribution can corrupt availability
- A07-SLO-4: Average-only latency cannot define latency objectives

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `observability/prometheus/rules/**`
- `docs/observability/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Record core request rate, bad-event ratio and latency good ratio.
2. Add p95/p99 histogram queries by service class.
3. Exclude reviewed client errors while counting overload/dependency failures as bad.
4. Validate rules with promtool or equivalent.

## Acceptance criteria

- [ ] 30-day availability and latency SLI queries are computable.
- [ ] 304 and client-error classification match the contract.
- [ ] Rules load without error.

## Required verification

- `promtool check rules observability/prometheus/rules/*.yml`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
