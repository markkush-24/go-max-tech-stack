# Decisions Required

Most decisions are implemented as explicit ADR tasks rather than informal chat choices.

| Decision | Blocking task | Recommended default |
|---|---|---|
| Direct gRPC trust model | TASK for `grpc-security-decision` | Internal authenticated boundary with TLS/mTLS; reflection only in development |
| Async durable acceptance | TASK for `async-acceptance-adr` | PostgreSQL durable job payload + transactional outbox/reconciliation before adding a broker |
| Generated artifact policy | TASK for `repo-track-quality-assets` | Commit generated Go protobuf code and verify regeneration produces no diff |
| Telemetry topology | `telemetry-bootstrap` / `observability-stack` | OTLP to Collector; Tempo for traces; Prometheus for metrics; JSON slog via Alloy to Loki; Grafana provisioning |
| OpenAPI codegen use | `openapi-codegen-contract` | Generate a typed client/models; do not replace hand-written handler architecture wholesale |
| SLO numbers | Performance phase | Treat Audit 07 numbers as hypotheses until calibrated by repeatable baselines |

When a decision task is reached, Codex must implement only the chosen ADR or ask for the missing choice. It must not silently choose a different architecture.
