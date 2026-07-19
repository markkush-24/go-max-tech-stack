# TASK-023 — Normalize logger ownership, component attribution and field schema

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-LOGGING`
- Gap: `GAP-LOGGING-023`
- Dependencies: TASK-016
- Initial status: `BACKLOG`

## Goal

Normalize logger ownership, component attribution and field schema. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A03-L12: Field naming and duration units are inconsistent
- A03-L3: Component attribution is duplicated or misleading
- A03-L4: Logger ownership is inconsistent and heavily dependent on global state
- A03-L5: Logging format and level are not configurable
- A09-F16: HTTP access logging can become a load bottleneck

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `cmd/api/main.go`
- `internal/middleware/middleware.go`
- `internal/outbound/profile_instrumentation.go`
- `internal/interceptors/**`
- `internal/config/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Create component-specific child loggers without duplicate keys.
2. Add configurable level, text/json format and optional source.
3. Define canonical duration and route/method field names.
4. Avoid hidden dependence on the global default logger.

## Acceptance criteria

- [ ] Each log event has one unambiguous component.
- [ ] Field names and duration units are consistent.
- [ ] Log configuration is tested.

## Required verification

- `go test ./internal/middleware/... ./internal/outbound/... ./internal/interceptors/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
