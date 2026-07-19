# TASK-091 — Add Profile latency, failure, retry and breaker experiments

## Metadata

- Phase: `P7`
- Priority: `P1`
- Remediation group: `RG-PERF`
- Gap: `GAP-PERF-091`
- Dependencies: TASK-089, TASK-059, TASK-036
- Initial status: `BACKLOG`

## Goal

Add Profile latency, failure, retry and breaker experiments. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A09-F10: Outbound active connection count is unlimited per host
- A09-F11: Retry amplification is not bounded by a dependency-level concurrency policy
- A11-F24: Fault-injection coverage is insufficient

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `loadtest/**`
- `testservices/profile/**`
- `docs/performance/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Provide controllable success, latency, 4xx, 5xx, malformed and reset modes.
2. Measure logical success, attempts, connection use and breaker state.
3. Test retry amplification and recovery.
4. Correlate representative traces and logs.

## Acceptance criteria

- [ ] The breaker protects upstream during sustained failure.
- [ ] Retry budget behavior is visible.
- [ ] Malformed/oversized responses do not destabilize the service.

## Required verification

- `find testservices loadtest -type f`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
