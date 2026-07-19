# TASK-086 — Add integration CI for PostgreSQL, full binary, TLS/gRPC and observability smoke

## Metadata

- Phase: `P6`
- Priority: `P1`
- Remediation group: `RG-QA`
- Gap: `GAP-QA-086`
- Dependencies: TASK-038, TASK-078, TASK-019
- Initial status: `BACKLOG`

## Goal

Add integration CI for PostgreSQL, full binary, TLS/gRPC and observability smoke. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A11-F24: Fault-injection coverage is insufficient
- A13-Q16: CI does not exercise deployment/integration topology

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `.github/workflows/**`
- `docker-compose*.yml`
- `scripts/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Start PostgreSQL and apply migrations.
2. Run the actual binary and exercise HTTP, HTTPS/HTTP2, direct gRPC and SSE smoke paths.
3. Start a minimal Collector/Prometheus/Tempo smoke stack.
4. Verify graceful shutdown and non-zero fatal component exit.

## Acceptance criteria

- [ ] CI proves the deployable topology, not only handlers.
- [ ] A broken migration/TLS/gRPC/OTLP configuration fails the integration job.
- [ ] Artifacts/logs are retained on failure.

## Required verification

- `docker compose config`
- `go test ./...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
