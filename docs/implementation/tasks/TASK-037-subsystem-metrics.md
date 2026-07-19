# TASK-037 — Add gRPC, DB, SSE, security and process metrics

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-METRICS`
- Gap: `GAP-METRICS-037`
- Dependencies: TASK-032
- Initial status: `BACKLOG`

## Goal

Add gRPC, DB, SSE, security and process metrics. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A02-E7: Streaming has a separate telemetry path
- A04-M11: Security and admission-control metrics lack actionable dimensions
- A04-M12: SSE metric names and semantics are weak
- A04-M13: gRPC has logging but no metrics
- A04-M14: Database metrics stop at pool snapshots
- A07-SLO-7: gRPC is absent from SLO telemetry
- A07-SLO-8: SSE delivery reliability is not measurable
- A09-F15: SSE is excluded from ordinary HTTP metrics but not from resource accounting
- A09-F18: gRPC has no explicit load controls or measurements
- A09-F21: No process/container/host metrics exist yet

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/transport/grpcserver/**`
- `internal/db/**`
- `internal/stream/**`
- `internal/middleware/**`
- `internal/runtimeinfo/**`
- `internal/metrics/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Add gRPC calls, codes, duration and in-flight.
2. Add DB operation duration/error/transaction outcomes in addition to pool stats.
3. Add SSE connection duration, outcomes, delivery success/drop and write timeout.
4. Add bounded security reasons and process/runtime resource metrics.

## Acceptance criteria

- [ ] Each subsystem has a RED/saturation view.
- [ ] Metrics use normalized operation names and reasons.
- [ ] No user/job/request ID labels exist.

## Required verification

- `go test ./internal/transport/grpcserver/... ./internal/db/... ./internal/stream/... ./internal/middleware/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
