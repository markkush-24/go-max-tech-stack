# TASK-025 — Create telemetry configuration, Resource and bootstrap runtime

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-OTEL`
- Gap: `GAP-OTEL-025`
- Dependencies: TASK-016
- Initial status: `BACKLOG`

## Goal

Create telemetry configuration, Resource and bootstrap runtime. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A03-L2: No trace ID or span ID correlation exists
- A04-M18: No standard exporter, resource identity or cardinality policy exists
- A05-T1: Distributed tracing is not implemented
- A06-G1: Telemetry bootstrap ownership
- A06-G2: Shared resource identity
- A06-G3: OTLP export

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/telemetry/**`
- `internal/config/**`
- `cmd/api/main.go`
- `go.mod`
- `go.sum`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Create one composition-root-owned telemetry runtime.
2. Configure service name, version, instance and environment Resource attributes.
3. Support standard OTLP environment configuration.
4. Allow telemetry-disabled local/test mode without scattered conditionals.

## Acceptance criteria

- [ ] TracerProvider and MeterProvider share the same Resource.
- [ ] Bootstrap has one owner and is dependency-injectable.
- [ ] Disabled mode is deterministic in tests.

## Required verification

- `go test ./internal/telemetry/... ./internal/config/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
