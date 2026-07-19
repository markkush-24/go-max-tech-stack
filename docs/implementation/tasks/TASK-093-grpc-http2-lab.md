# TASK-093 — Compare direct gRPC, HTTP bridge, HTTP/1.1 and HTTP/2

## Metadata

- Phase: `P7`
- Priority: `P2`
- Remediation group: `RG-PERF`
- Gap: `GAP-PERF-093`
- Dependencies: TASK-031, TASK-020, TASK-089
- Initial status: `BACKLOG`

## Goal

Compare direct gRPC, HTTP bridge, HTTP/1.1 and HTTP/2. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A09-F18: gRPC has no explicit load controls or measurements
- A12-F10: TLS and HTTP2 have no explicit production security profile

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `loadtest/**`
- `docs/performance/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Implement direct gRPC and bridge load clients.
2. Compare HTTP/1.1 and TLS/HTTP2 under matched payloads.
3. Record connection reuse, latency, codes and CPU/allocations.
4. Keep security enabled for production-like runs.

## Acceptance criteria

- [ ] Results distinguish protocol overhead from business/storage work.
- [ ] gRPC and HTTP metrics/traces are correlated.
- [ ] Test configuration is reproducible.

## Required verification

- `find loadtest -type f`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
