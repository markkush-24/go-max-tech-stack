# TASK-098 — Define benchmark/load baselines and regression review policy

## Metadata

- Phase: `P7`
- Priority: `P2`
- Remediation group: `RG-PERF`
- Gap: `GAP-PERF-098`
- Dependencies: TASK-082, TASK-089, TASK-097
- Initial status: `BACKLOG`

## Goal

Define benchmark/load baselines and regression review policy. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A07-SLO-12: Numerical objectives are provisional until baseline audit
- A09-F23: No performance regression policy exists

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `docs/performance/POLICY.md`
- `.github/workflows/**`
- `scripts/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Define stable benchmark environment and statistical comparison method.
2. Set review thresholds rather than brittle universal pass/fail numbers.
3. Store representative baseline artifacts.
4. Require explanation for material throughput/latency/allocation changes.

## Acceptance criteria

- [ ] Benchstat or equivalent comparison is documented.
- [ ] CI can run smoke benchmarks without pretending to be a production capacity test.
- [ ] Baseline changes require explicit review.

## Required verification

- `test -f docs/performance/POLICY.md`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
