# Task Index

Total implementation tasks: **98**.

Status values: `BACKLOG`, `READY`, `IN_PROGRESS`, `REVIEW`, `DONE`, `BLOCKED`, `DEFERRED`, `ACCEPTED_RISK`.

## P0 — Repository reproducibility and execution baseline

Make a clean checkout buildable, testable and safe to hand to agents.

| Task | Priority | Dependencies | Initial status | Title |
|---|---|---|---|---|
| [TASK-001](tasks/TASK-001-repo-track-quality-assets.md) | P0 | — | DONE | Track CI, scripts, tools and generated-artifact policy |
| [TASK-002](tasks/TASK-002-pin-codegen-tools.md) | P0 | TASK-001 | DONE | Pin protobuf and quality-tool versions |
| [TASK-003](tasks/TASK-003-clean-checkout-gate.md) | P0 | TASK-001, TASK-002 | DONE | Add clean-checkout and generated-drift verification |
| [TASK-004](tasks/TASK-004-archive-hygiene.md) | P2 | — | DONE | Define repository and archive hygiene rules |
| [TASK-005](tasks/TASK-005-align-config-docs.md) | P1 | — | DONE | Align runtime defaults, README and actual route surface |
| [TASK-006](tasks/TASK-006-isolate-config-tests.md) | P1 | — | DONE | Make configuration tests independent of host environment |
| [TASK-007](tasks/TASK-007-target-toolchain-baseline.md) | P0 | TASK-003 | DONE | Establish the Go 1.25.12 verification baseline |

## P0 Corrections — Review follow-up cards

Correction cards are tracked separately from the 98 implementation tasks.

| Task | Priority | Dependencies | Initial status | Title |
|---|---|---|---|---|
| [P0-CORR-001](corrections/P0-CORR-001-quality-system-consistency.md) | P0 | TASK-007 | DONE | Make codegen, clean-checkout and toolchain checks exact |
| [P0-CORR-002](corrections/P0-CORR-002-archive-export-hardening.md) | P0 | — | DONE | Make repository export fail safely and scan all key risks |
| [P0-CORR-003](corrections/P0-CORR-003-operational-docs-provenance.md) | P0 | — | DONE | Complete operational documentation and P0 provenance |
| [P0-CORR-004](corrections/P0-CORR-004-final-closure-guards.md) | P0 | — | DONE | Add final regression guards and close P0 bookkeeping |

## P1 — Critical correctness, lifecycle and security

Remove defects that can corrupt state, bypass security or make shutdown results untrustworthy.

| Task | Priority | Dependencies | Initial status | Title |
|---|---|---|---|---|
| [TASK-008](tasks/TASK-008-fix-status-recorder.md) | P0 | TASK-007 | READY | Fix first-status-only response recording with regression tests |
| [TASK-009](tasks/TASK-009-define-job-state-machine.md) | P0 | TASK-007 | READY | Define an explicit immutable-terminal Job state machine |
| [TASK-010](tasks/TASK-010-enforce-job-cas.md) | P0 | TASK-009 | BLOCKED | Enforce Job CAS transitions in memory and PostgreSQL repositories |
| [TASK-011](tasks/TASK-011-worker-generation-lifecycle.md) | P0 | TASK-009 | BLOCKED | Redesign WorkerPool around an immutable worker generation |
| [TASK-012](tasks/TASK-012-worker-stop-repair.md) | P0 | TASK-011, TASK-010 | BLOCKED | Make worker Stop and terminal repair bounded and truly idempotent |
| [TASK-013](tasks/TASK-013-queue-admission-contract.md) | P0 | TASK-007 | READY | Define deterministic queue cancellation and StopAccepting semantics |
| [TASK-014](tasks/TASK-014-async-event-topology.md) | P0 | TASK-010, TASK-013 | BLOCKED | Correct async transition ordering and use one Event Hub in testkit |
| [TASK-015](tasks/TASK-015-grpc-runtime-owner.md) | P0 | TASK-007 | READY | Give the gRPC runtime single-start/single-stop ownership |
| [TASK-016](tasks/TASK-016-application-supervisor.md) | P0 | TASK-011, TASK-015 | BLOCKED | Introduce one application supervisor for HTTP, HTTPS, gRPC, workers and streams |
| [TASK-017](tasks/TASK-017-shutdown-budget-outcome.md) | P0 | TASK-016, TASK-012 | BLOCKED | Implement one global shutdown budget and truthful outcome model |
| [TASK-018](tasks/TASK-018-grpc-security-decision.md) | P0 | TASK-005 | READY | Record the direct gRPC trust and exposure model |
| [TASK-019](tasks/TASK-019-grpc-transport-security.md) | P0 | TASK-018, TASK-015 | BLOCKED | Add gRPC TLS or mTLS and environment-controlled reflection |
| [TASK-020](tasks/TASK-020-grpc-authz.md) | P0 | TASK-018, TASK-019 | BLOCKED | Add gRPC authentication, RBAC and owner authorization |
| [TASK-021](tasks/TASK-021-grpc-metadata-errors.md) | P0 | TASK-020 | BLOCKED | Normalize gRPC metadata, request ID, status codes and bridge timeout |
| [TASK-022](tasks/TASK-022-production-security-guards.md) | P0 | TASK-005 | READY | Add environment guards for JWT, TLS, proxy and security-header defaults |

## P2 — Observability foundation

Introduce stable logging, tracing and metric ownership without changing business behavior.

| Task | Priority | Dependencies | Initial status | Title |
|---|---|---|---|---|
| [TASK-023](tasks/TASK-023-logging-schema-config.md) | P1 | TASK-016 | BLOCKED | Normalize logger ownership, component attribution and field schema |
| [TASK-024](tasks/TASK-024-logging-policy-events.md) | P1 | TASK-023 | BLOCKED | Add redaction policy and coherent security, retry, SSE and shutdown events |
| [TASK-025](tasks/TASK-025-telemetry-bootstrap.md) | P1 | TASK-016 | BLOCKED | Create telemetry configuration, Resource and bootstrap runtime |
| [TASK-026](tasks/TASK-026-telemetry-lifecycle.md) | P1 | TASK-025, TASK-017 | BLOCKED | Integrate telemetry fail-open, ForceFlush and Shutdown lifecycle |
| [TASK-027](tasks/TASK-027-async-propagation-envelope.md) | P1 | TASK-013, TASK-011, TASK-025 | BLOCKED | Add broker-compatible async propagation envelope and per-job context |
| [TASK-028](tasks/TASK-028-log-trace-correlation.md) | P1 | TASK-023, TASK-025 | BLOCKED | Inject request ID, trace ID and span ID into contextual slog events |
| [TASK-029](tasks/TASK-029-http-tracing.md) | P1 | TASK-025, TASK-008 | BLOCKED | Instrument inbound HTTP while preserving ServeMux route identity |
| [TASK-030](tasks/TASK-030-outbound-tracing.md) | P1 | TASK-025 | BLOCKED | Instrument outbound HTTP and model logical retries |
| [TASK-031](tasks/TASK-031-grpc-tracing.md) | P1 | TASK-025, TASK-021 | BLOCKED | Instrument gRPC client/server and stream propagation |
| [TASK-032](tasks/TASK-032-metrics-registry.md) | P1 | TASK-025, TASK-008 | BLOCKED | Replace process-global metric ownership with a DI-owned registry |
| [TASK-033](tasks/TASK-033-metric-correctness.md) | P1 | TASK-032 | BLOCKED | Fix queue/auth/bulkhead instrument semantics |
| [TASK-034](tasks/TASK-034-http-red-metrics.md) | P1 | TASK-032, TASK-008 | BLOCKED | Add HTTP RED metrics, histograms and service-class attributes |
| [TASK-035](tasks/TASK-035-job-queue-metrics.md) | P1 | TASK-032, TASK-009 | BLOCKED | Add complete queue and Job lifecycle metrics |
| [TASK-036](tasks/TASK-036-outbound-logical-metrics.md) | P1 | TASK-032 | BLOCKED | Add logical Profile operation and retry metrics |
| [TASK-037](tasks/TASK-037-subsystem-metrics.md) | P1 | TASK-032 | BLOCKED | Add gRPC, DB, SSE, security and process metrics |
| [TASK-038](tasks/TASK-038-observability-stack.md) | P1 | TASK-026, TASK-034 | BLOCKED | Provision Collector, Tempo, Prometheus and Grafana data sources |
| [TASK-039](tasks/TASK-039-log-pipeline.md) | P1 | TASK-024, TASK-028, TASK-038 | BLOCKED | Ship JSON slog logs through Alloy to Loki with trace correlation |

## P3 — SLO, dashboards and telemetry hardening

Turn reliable signals into operational views, alerts and runbooks.

| Task | Priority | Dependencies | Initial status | Title |
|---|---|---|---|---|
| [TASK-040](tasks/TASK-040-metric-sli-contract.md) | P1 | TASK-034, TASK-035, TASK-036, TASK-037 | BLOCKED | Publish metric, cardinality and SLI classification contracts |
| [TASK-041](tasks/TASK-041-http-slo-rules.md) | P1 | TASK-040, TASK-038 | BLOCKED | Implement HTTP SLO recording rules |
| [TASK-042](tasks/TASK-042-subsystem-sli-rules.md) | P1 | TASK-040, TASK-038 | BLOCKED | Implement async, Profile, gRPC and SSE SLI rules |
| [TASK-043](tasks/TASK-043-alert-rules.md) | P1 | TASK-041, TASK-042 | BLOCKED | Add multi-window burn-rate and diagnostic alerts |
| [TASK-044](tasks/TASK-044-grafana-dashboards.md) | P1 | TASK-038, TASK-041, TASK-042 | BLOCKED | Provision the ten planned Grafana dashboards as code |
| [TASK-045](tasks/TASK-045-observability-runbooks.md) | P2 | TASK-043, TASK-044 | BLOCKED | Add alert runbooks and load-experiment annotation policy |
| [TASK-046](tasks/TASK-046-telemetry-reliability-tests.md) | P1 | TASK-026, TASK-031, TASK-027, TASK-038 | BLOCKED | Add telemetry outage, flush and propagation tests |

## P4 — Async durability and resilience

Make accepted asynchronous work recoverable and dependencies safely degradable.

| Task | Priority | Dependencies | Initial status | Title |
|---|---|---|---|---|
| [TASK-047](tasks/TASK-047-durable-job-payload.md) | P0 | TASK-010, TASK-027 | BLOCKED | Persist durable Job payload and correlation metadata |
| [TASK-048](tasks/TASK-048-job-lease-recovery.md) | P0 | TASK-047 | BLOCKED | Add Job ownership, lease, attempt and transition version metadata |
| [TASK-049](tasks/TASK-049-async-acceptance-adr.md) | P0 | TASK-048 | BLOCKED | Choose transactional async acceptance and outbox semantics |
| [TASK-050](tasks/TASK-050-durable-async-admission.md) | P0 | TASK-049 | BLOCKED | Implement durable async admission without insert-delete compensation |
| [TASK-051](tasks/TASK-051-startup-reconciliation.md) | P0 | TASK-050, TASK-048 | BLOCKED | Implement startup reconciliation and expired-work recovery |
| [TASK-052](tasks/TASK-052-job-retry-dlq-model.md) | P1 | TASK-048 | BLOCKED | Add retry, backoff, poison-job and dead-letter state model |
| [TASK-053](tasks/TASK-053-worker-retry-repair.md) | P1 | TASK-052, TASK-012 | BLOCKED | Execute durable retry, repair and reconciliation policies in workers |
| [TASK-054](tasks/TASK-054-atomic-user-job-completion.md) | P0 | TASK-050, TASK-010 | BLOCKED | Make user creation and Job completion atomic or idempotently recoverable |
| [TASK-055](tasks/TASK-055-post-idempotency.md) | P1 | TASK-054 | BLOCKED | Add durable Idempotency-Key behavior for create operations |
| [TASK-056](tasks/TASK-056-worker-panic-policy.md) | P0 | TASK-053 | BLOCKED | Contain or supervise worker panics with durable job outcome |
| [TASK-057](tasks/TASK-057-profile-response-validation.md) | P1 | — | BACKLOG | Strictly validate and byte-bound Profile responses |
| [TASK-058](tasks/TASK-058-safe-retry-policy.md) | P1 | — | BACKLOG | Fix backoff overflow and honor bounded HTTP retry guidance |
| [TASK-059](tasks/TASK-059-profile-resilience-controls.md) | P1 | TASK-058 | BLOCKED | Add Profile-specific connection bound, bulkhead, circuit breaker and retry budget |
| [TASK-060](tasks/TASK-060-workload-admission-controls.md) | P1 | TASK-032 | BLOCKED | Split rate limits and bulkheads by workload, including SSE connections |
| [TASK-061](tasks/TASK-061-dependency-error-taxonomy.md) | P1 | TASK-021 | BLOCKED | Normalize DB, context, outbound and gRPC error taxonomy |
| [TASK-062](tasks/TASK-062-readiness-reliability.md) | P1 | TASK-061 | BLOCKED | Add schema-aware, redacted and independently budgeted readiness checks |

## P5 — HTTP and streaming contracts

Normalize API, browser, SSE and caching behavior.

| Task | Priority | Dependencies | Initial status | Title |
|---|---|---|---|---|
| [TASK-063](tasks/TASK-063-accept-negotiation.md) | P1 | — | BACKLOG | Implement RFC-compliant Accept preference selection |
| [TASK-064](tasks/TASK-064-uniform-method-routing.md) | P1 | — | BACKLOG | Unify ServeMux method registration, HEAD and Allow behavior |
| [TASK-065](tasks/TASK-065-strict-route-query-contract.md) | P1 | TASK-064 | BLOCKED | Resolve v2 item surface, unknown-route precedence and strict query parsing |
| [TASK-066](tasks/TASK-066-cors-response-contract.md) | P1 | — | BACKLOG | Expose required browser headers and complete CORS Vary behavior |
| [TASK-067](tasks/TASK-067-route-cors-policy.md) | P2 | TASK-066, TASK-064 | BLOCKED | Introduce route-aware CORS method/header policies |
| [TASK-068](tasks/TASK-068-host-security-headers.md) | P1 | TASK-022 | BLOCKED | Apply configured security headers and host-wide HSTS correctly |
| [TASK-069](tasks/TASK-069-trusted-xff-chain.md) | P1 | TASK-022 | BLOCKED | Define and implement trusted X-Forwarded-For chain semantics |
| [TASK-070](tasks/TASK-070-problem-catalog.md) | P1 | TASK-061 | BLOCKED | Adopt an RFC 9457 Problem catalog and explicit cache policy |
| [TASK-071](tasks/TASK-071-sse-handshake-errors.md) | P0 | TASK-060, TASK-008 | BLOCKED | Commit SSE immediately and handle post-commit errors without Problem corruption |
| [TASK-072](tasks/TASK-072-sse-resume-contract.md) | P1 | TASK-071, TASK-010 | BLOCKED | Add SSE snapshot, sequence and reconnect/resync semantics |
| [TASK-073](tasks/TASK-073-range-contract.md) | P2 | TASK-064 | BLOCKED | Complete Range validators, conditional requests and HEAD behavior |
| [TASK-074](tasks/TASK-074-user-list-pagination.md) | P1 | — | BACKLOG | Add bounded pagination to user-list endpoints |
| [TASK-075](tasks/TASK-075-memory-email-index.md) | P2 | — | BACKLOG | Replace O(n) in-memory email lookup with an indexed structure |

## P6 — OpenAPI and quality system

Make contracts and high-risk behavior reproducibly verifiable in CI.

| Task | Priority | Dependencies | Initial status | Title |
|---|---|---|---|---|
| [TASK-076](tasks/TASK-076-production-testkit.md) | P1 | TASK-016, TASK-014 | BLOCKED | Create production-faithful and explicitly scoped testkit fixtures |
| [TASK-077](tasks/TASK-077-concurrency-regression-suite.md) | P0 | TASK-012, TASK-010, TASK-033 | BLOCKED | Commit regression tests for queue, worker, Job state and metrics races |
| [TASK-078](tasks/TASK-078-postgres-integration-tests.md) | P0 | TASK-007 | READY | Add PostgreSQL repository, migration and transaction integration tests |
| [TASK-079](tasks/TASK-079-openapi-spec.md) | P1 | TASK-063, TASK-065, TASK-070, TASK-066 | BLOCKED | Create the machine-readable OpenAPI source of truth |
| [TASK-080](tasks/TASK-080-openapi-codegen-contract.md) | P1 | TASK-079, TASK-002 | BLOCKED | Add pinned OpenAPI validation, codegen and runtime conformance checks |
| [TASK-081](tasks/TASK-081-fuzz-suite.md) | P1 | — | BACKLOG | Add fuzz/property tests for protocol parsers and state transitions |
| [TASK-082](tasks/TASK-082-benchmark-suite.md) | P1 | TASK-008, TASK-032 | BLOCKED | Commit focused benchmarks for hot paths and allocations |
| [TASK-083](tasks/TASK-083-coverage-ci-policy.md) | P1 | TASK-077, TASK-078 | BLOCKED | Add coverage reporting and risk-based CI gates |
| [TASK-084](tasks/TASK-084-deterministic-tests.md) | P1 | TASK-076 | BLOCKED | Remove avoidable sleeps, port races and brittle representation assertions |
| [TASK-085](tasks/TASK-085-quality-entrypoint.md) | P1 | TASK-003, TASK-080 | BLOCKED | Provide cross-platform quality commands and drift/security gates |
| [TASK-086](tasks/TASK-086-integration-ci-topology.md) | P1 | TASK-038, TASK-078, TASK-019 | BLOCKED | Add integration CI for PostgreSQL, full binary, TLS/gRPC and observability smoke |
| [TASK-087](tasks/TASK-087-test-cleanup.md) | P2 | TASK-006, TASK-084 | BLOCKED | Harden configuration and representation tests |

## P7 — Performance and failure laboratory

Build repeatable high-load, saturation and fault experiments.

| Task | Priority | Dependencies | Initial status | Title |
|---|---|---|---|---|
| [TASK-088](tasks/TASK-088-capacity-profiles.md) | P1 | TASK-060, TASK-062 | BLOCKED | Add explicit functional, baseline, observability and saturation config profiles |
| [TASK-089](tasks/TASK-089-http-load-harness.md) | P1 | TASK-088, TASK-082, TASK-038 | BLOCKED | Create versioned HTTP read/write/mixed load scenarios |
| [TASK-090](tasks/TASK-090-async-saturation-lab.md) | P1 | TASK-089, TASK-035, TASK-051 | BLOCKED | Add queue, worker and DB saturation experiments |
| [TASK-091](tasks/TASK-091-profile-fault-lab.md) | P1 | TASK-089, TASK-059, TASK-036 | BLOCKED | Add Profile latency, failure, retry and breaker experiments |
| [TASK-092](tasks/TASK-092-sse-scale-lab.md) | P1 | TASK-072, TASK-037, TASK-088 | BLOCKED | Add SSE connection, fan-out and slow-client experiments |
| [TASK-093](tasks/TASK-093-grpc-http2-lab.md) | P2 | TASK-031, TASK-020, TASK-089 | BLOCKED | Compare direct gRPC, HTTP bridge, HTTP/1.1 and HTTP/2 |
| [TASK-094](tasks/TASK-094-postgres-saturation-lab.md) | P1 | TASK-078, TASK-088, TASK-037 | BLOCKED | Create reproducible PostgreSQL pool and query saturation experiments |
| [TASK-095](tasks/TASK-095-shutdown-load-lab.md) | P1 | TASK-017, TASK-046, TASK-089 | BLOCKED | Test graceful and forced shutdown under active HTTP, gRPC, jobs and SSE |
| [TASK-096](tasks/TASK-096-runtime-profile-controls.md) | P2 | TASK-088 | BLOCKED | Add lab-only mutex/block profiling and runtime capture protocol |
| [TASK-097](tasks/TASK-097-telemetry-overhead-lab.md) | P1 | TASK-038, TASK-046, TASK-089 | BLOCKED | Measure telemetry overhead and Collector outage behavior |
| [TASK-098](tasks/TASK-098-performance-regression-policy.md) | P2 | TASK-082, TASK-089, TASK-097 | BLOCKED | Define benchmark/load baselines and regression review policy |
