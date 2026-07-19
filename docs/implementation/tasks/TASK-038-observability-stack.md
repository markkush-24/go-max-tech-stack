# TASK-038 — Provision Collector, Tempo, Prometheus and Grafana data sources

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-OBS-STACK`
- Gap: `GAP-OBS-STACK-038`
- Dependencies: TASK-026, TASK-034
- Initial status: `BACKLOG`

## Goal

Provision Collector, Tempo, Prometheus and Grafana data sources. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A04-M15: The runtime endpoint is diagnostic JSON, not an operational metrics pipeline
- A06-G3: OTLP export
- A06-G4: Collector
- A06-G5: Tempo
- A06-G6: Prometheus operational pipeline
- A06-G8: Grafana provisioning
- A07-SLO-1: Current data cannot support trustworthy SLOs

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `docker-compose*.yml`
- `observability/collector/**`
- `observability/tempo/**`
- `observability/prometheus/**`
- `observability/grafana/**`
- `README.md`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Add OTLP receiver, memory limiter and batch processor.
2. Export traces to Tempo and metrics for Prometheus scraping.
3. Provision Grafana data sources from files.
4. Keep telemetry backend failure outside application readiness.

## Acceptance criteria

- [ ] Local stack starts reproducibly.
- [ ] A request appears in Tempo and HTTP metrics appear in Prometheus.
- [ ] Collector internal telemetry is scrapeable.

## Required verification

- `docker compose config`
- `docker compose up -d`
- `docker compose ps`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
