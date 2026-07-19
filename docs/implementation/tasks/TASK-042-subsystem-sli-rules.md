# TASK-042 — Implement async, Profile, gRPC and SSE SLI rules

## Metadata

- Phase: `P3`
- Priority: `P1`
- Remediation group: `RG-SLO`
- Gap: `GAP-SLO-042`
- Dependencies: TASK-040, TASK-038
- Initial status: `BACKLOG`

## Goal

Implement async, Profile, gRPC and SSE SLI rules. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A07-SLO-5: Async acceptance and completion are conflated/incomplete
- A07-SLO-6: Outbound user outcome is not represented
- A07-SLO-7: gRPC is absent from SLO telemetry
- A07-SLO-8: SSE delivery reliability is not measurable

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

1. Record async acceptance, terminality and system outcome.
2. Record logical Profile availability/latency across retries.
3. Record gRPC availability/latency by eligible code.
4. Record SSE connection and delivery reliability.

## Acceptance criteria

- [ ] Each declared non-HTTP SLO has a numerator and denominator.
- [ ] Physical attempts do not inflate logical Profile denominators.
- [ ] SSE drops have a successful-delivery denominator.

## Required verification

- `promtool check rules observability/prometheus/rules/*.yml`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
