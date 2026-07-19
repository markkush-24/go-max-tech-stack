# TASK-034 — Add HTTP RED metrics, histograms and service-class attributes

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-METRICS`
- Gap: `GAP-METRICS-034`
- Dependencies: TASK-032, TASK-008
- Initial status: `BACKLOG`

## Goal

Add HTTP RED metrics, histograms and service-class attributes. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A02-E6: Observability route identity is intentionally mux-derived
- A04-M4: Latency metrics cannot describe tail latency
- A04-M5: HTTP metric inclusion policy is heuristic and inconsistent for streaming
- A04-M6: HTTP metrics omit important service signals
- A07-SLO-3: Service-class mapping is not formally encoded
- A07-SLO-4: Average-only latency cannot define latency objectives

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/middleware/metrics.go`
- `internal/metrics/**`
- `internal/router/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Emit request count, active requests and duration histogram.
2. Use bounded route/method/status/error attributes.
3. Encode reviewed service classes for core, async, dependency, export and stream.
4. Define streaming inclusion separately from finite-request latency.

## Acceptance criteria

- [ ] p95/p99 can be calculated.
- [ ] No raw path or identifier becomes an attribute.
- [ ] Wire status and metric status match.

## Required verification

- `go test ./internal/middleware/... ./internal/metrics/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
