# TASK-089 — Create versioned HTTP read/write/mixed load scenarios

## Metadata

- Phase: `P7`
- Priority: `P1`
- Remediation group: `RG-PERF`
- Gap: `GAP-PERF-089`
- Dependencies: TASK-088, TASK-082, TASK-038
- Initial status: `BACKLOG`

## Goal

Create versioned HTTP read/write/mixed load scenarios. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A09-F1: No persistent benchmark or load-test suite exists
- A13-Q9: There are no committed benchmarks or load scripts

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `loadtest/**`
- `docs/performance/**`
- `scripts/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Choose and pin k6, Vegeta or equivalent tooling.
2. Implement read, sync write and mixed workloads with warm-up and steady phases.
3. Record environment, payload, concurrency, duration and result artifacts.
4. Emit experiment annotations.

## Acceptance criteria

- [ ] A clean checkout can reproduce the same scenario.
- [ ] Results include throughput, error classes and latency distribution.
- [ ] Load generator location and limits are documented.

## Required verification

- `find loadtest -type f`
- `docker compose config`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
