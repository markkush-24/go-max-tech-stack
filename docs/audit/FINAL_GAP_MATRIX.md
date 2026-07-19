# Final Gap Matrix

Unique implementation gaps/outcomes: **98**. Raw findings were deduplicated into these bounded outcomes.

| Gap ID | Phase | Priority | Remediation group | Outcome | Source findings | Task |
|---|---|---|---|---|---:|---|
| GAP-REPRO-001 | P0 | P0 | RG-REPRO | Track CI, scripts, tools and generated-artifact policy | 4 | TASK-001 |
| GAP-REPRO-002 | P0 | P0 | RG-REPRO | Pin protobuf and quality-tool versions | 2 | TASK-002 |
| GAP-REPRO-003 | P0 | P0 | RG-REPRO | Add clean-checkout and generated-drift verification | 3 | TASK-003 |
| GAP-REPRO-004 | P0 | P2 | RG-REPRO | Define repository and archive hygiene rules | 1 | TASK-004 |
| GAP-DOCS-005 | P0 | P1 | RG-DOCS | Align runtime defaults, README and actual route surface | 3 | TASK-005 |
| GAP-QA-006 | P0 | P1 | RG-QA | Make configuration tests independent of host environment | 1 | TASK-006 |
| GAP-QA-007 | P0 | P0 | RG-QA | Establish the Go 1.25.8 verification baseline | 1 | TASK-007 |
| GAP-HTTP-CORRECTNESS-008 | P1 | P0 | RG-HTTP-CORRECTNESS | Fix first-status-only response recording with regression tests | 3 | TASK-008 |
| GAP-JOB-CORRECTNESS-009 | P1 | P0 | RG-JOB-CORRECTNESS | Define an explicit immutable-terminal Job state machine | 2 | TASK-009 |
| GAP-JOB-CORRECTNESS-010 | P1 | P0 | RG-JOB-CORRECTNESS | Enforce Job CAS transitions in memory and PostgreSQL repositories | 2 | TASK-010 |
| GAP-LIFECYCLE-011 | P1 | P0 | RG-LIFECYCLE | Redesign WorkerPool around an immutable worker generation | 5 | TASK-011 |
| GAP-LIFECYCLE-012 | P1 | P0 | RG-LIFECYCLE | Make worker Stop and terminal repair bounded and truly idempotent | 10 | TASK-012 |
| GAP-CONCURRENCY-013 | P1 | P0 | RG-CONCURRENCY | Define deterministic queue cancellation and StopAccepting semantics | 2 | TASK-013 |
| GAP-STREAM-CORRECTNESS-014 | P1 | P0 | RG-STREAM-CORRECTNESS | Correct async transition ordering and use one Event Hub in testkit | 4 | TASK-014 |
| GAP-LIFECYCLE-015 | P1 | P0 | RG-LIFECYCLE | Give the gRPC runtime single-start/single-stop ownership | 1 | TASK-015 |
| GAP-LIFECYCLE-016 | P1 | P0 | RG-LIFECYCLE | Introduce one application supervisor for HTTP, HTTPS, gRPC, workers and streams | 10 | TASK-016 |
| GAP-LIFECYCLE-017 | P1 | P0 | RG-LIFECYCLE | Implement one global shutdown budget and truthful outcome model | 6 | TASK-017 |
| GAP-GRPC-SECURITY-018 | P1 | P0 | RG-GRPC-SECURITY | Record the direct gRPC trust and exposure model | 1 | TASK-018 |
| GAP-GRPC-SECURITY-019 | P1 | P0 | RG-GRPC-SECURITY | Add gRPC TLS or mTLS and environment-controlled reflection | 2 | TASK-019 |
| GAP-GRPC-SECURITY-020 | P1 | P0 | RG-GRPC-SECURITY | Add gRPC authentication, RBAC and owner authorization | 1 | TASK-020 |
| GAP-GRPC-CONTRACT-021 | P1 | P0 | RG-GRPC-CONTRACT | Normalize gRPC metadata, request ID, status codes and bridge timeout | 8 | TASK-021 |
| GAP-SECURITY-022 | P1 | P0 | RG-SECURITY | Add environment guards for JWT, TLS, proxy and security-header defaults | 4 | TASK-022 |
| GAP-LOGGING-023 | P2 | P1 | RG-LOGGING | Normalize logger ownership, component attribution and field schema | 5 | TASK-023 |
| GAP-LOGGING-024 | P2 | P1 | RG-LOGGING | Add redaction policy and coherent security, retry, SSE and shutdown events | 8 | TASK-024 |
| GAP-OTEL-025 | P2 | P1 | RG-OTEL | Create telemetry configuration, Resource and bootstrap runtime | 6 | TASK-025 |
| GAP-OTEL-026 | P2 | P1 | RG-OTEL | Integrate telemetry fail-open, ForceFlush and Shutdown lifecycle | 5 | TASK-026 |
| GAP-OTEL-027 | P2 | P1 | RG-OTEL | Add broker-compatible async propagation envelope and per-job context | 8 | TASK-027 |
| GAP-OTEL-028 | P2 | P1 | RG-OTEL | Inject request ID, trace ID and span ID into contextual slog events | 3 | TASK-028 |
| GAP-OTEL-029 | P2 | P1 | RG-OTEL | Instrument inbound HTTP while preserving ServeMux route identity | 4 | TASK-029 |
| GAP-OTEL-030 | P2 | P1 | RG-OTEL | Instrument outbound HTTP and model logical retries | 4 | TASK-030 |
| GAP-OTEL-031 | P2 | P1 | RG-OTEL | Instrument gRPC client/server and stream propagation | 5 | TASK-031 |
| GAP-METRICS-032 | P2 | P1 | RG-METRICS | Replace process-global metric ownership with a DI-owned registry | 5 | TASK-032 |
| GAP-METRICS-033 | P2 | P1 | RG-METRICS | Fix queue/auth/bulkhead instrument semantics | 3 | TASK-033 |
| GAP-METRICS-034 | P2 | P1 | RG-METRICS | Add HTTP RED metrics, histograms and service-class attributes | 6 | TASK-034 |
| GAP-METRICS-035 | P2 | P1 | RG-METRICS | Add complete queue and Job lifecycle metrics | 4 | TASK-035 |
| GAP-METRICS-036 | P2 | P1 | RG-METRICS | Add logical Profile operation and retry metrics | 3 | TASK-036 |
| GAP-METRICS-037 | P2 | P1 | RG-METRICS | Add gRPC, DB, SSE, security and process metrics | 10 | TASK-037 |
| GAP-OBS-STACK-038 | P2 | P1 | RG-OBS-STACK | Provision Collector, Tempo, Prometheus and Grafana data sources | 7 | TASK-038 |
| GAP-OBS-STACK-039 | P2 | P1 | RG-OBS-STACK | Ship JSON slog logs through Alloy to Loki with trace correlation | 1 | TASK-039 |
| GAP-SLO-040 | P3 | P1 | RG-SLO | Publish metric, cardinality and SLI classification contracts | 6 | TASK-040 |
| GAP-SLO-041 | P3 | P1 | RG-SLO | Implement HTTP SLO recording rules | 2 | TASK-041 |
| GAP-SLO-042 | P3 | P1 | RG-SLO | Implement async, Profile, gRPC and SSE SLI rules | 4 | TASK-042 |
| GAP-SLO-043 | P3 | P1 | RG-SLO | Add multi-window burn-rate and diagnostic alerts | 3 | TASK-043 |
| GAP-SLO-044 | P3 | P1 | RG-SLO | Provision the ten planned Grafana dashboards as code | 2 | TASK-044 |
| GAP-SLO-045 | P3 | P2 | RG-SLO | Add alert runbooks and load-experiment annotation policy | 3 | TASK-045 |
| GAP-OTEL-046 | P3 | P1 | RG-OTEL | Add telemetry outage, flush and propagation tests | 6 | TASK-046 |
| GAP-ASYNC-DURABILITY-047 | P4 | P0 | RG-ASYNC-DURABILITY | Persist durable Job payload and correlation metadata | 2 | TASK-047 |
| GAP-ASYNC-DURABILITY-048 | P4 | P0 | RG-ASYNC-DURABILITY | Add Job ownership, lease, attempt and transition version metadata | 1 | TASK-048 |
| GAP-ASYNC-DURABILITY-049 | P4 | P0 | RG-ASYNC-DURABILITY | Choose transactional async acceptance and outbox semantics | 2 | TASK-049 |
| GAP-ASYNC-DURABILITY-050 | P4 | P0 | RG-ASYNC-DURABILITY | Implement durable async admission without insert-delete compensation | 3 | TASK-050 |
| GAP-ASYNC-DURABILITY-051 | P4 | P0 | RG-ASYNC-DURABILITY | Implement startup reconciliation and expired-work recovery | 2 | TASK-051 |
| GAP-ASYNC-DURABILITY-052 | P4 | P1 | RG-ASYNC-DURABILITY | Add retry, backoff, poison-job and dead-letter state model | 1 | TASK-052 |
| GAP-ASYNC-DURABILITY-053 | P4 | P1 | RG-ASYNC-DURABILITY | Execute durable retry, repair and reconciliation policies in workers | 3 | TASK-053 |
| GAP-ASYNC-DURABILITY-054 | P4 | P0 | RG-ASYNC-DURABILITY | Make user creation and Job completion atomic or idempotently recoverable | 2 | TASK-054 |
| GAP-RESILIENCE-055 | P4 | P1 | RG-RESILIENCE | Add durable Idempotency-Key behavior for create operations | 1 | TASK-055 |
| GAP-RESILIENCE-056 | P4 | P0 | RG-RESILIENCE | Contain or supervise worker panics with durable job outcome | 1 | TASK-056 |
| GAP-OUTBOUND-057 | P4 | P1 | RG-OUTBOUND | Strictly validate and byte-bound Profile responses | 2 | TASK-057 |
| GAP-OUTBOUND-058 | P4 | P1 | RG-OUTBOUND | Fix backoff overflow and honor bounded HTTP retry guidance | 2 | TASK-058 |
| GAP-OUTBOUND-059 | P4 | P1 | RG-OUTBOUND | Add Profile-specific connection bound, bulkhead, circuit breaker and retry budget | 3 | TASK-059 |
| GAP-RESILIENCE-060 | P4 | P1 | RG-RESILIENCE | Split rate limits and bulkheads by workload, including SSE connections | 5 | TASK-060 |
| GAP-ERROR-CONTRACT-061 | P4 | P1 | RG-ERROR-CONTRACT | Normalize DB, context, outbound and gRPC error taxonomy | 6 | TASK-061 |
| GAP-HEALTH-062 | P4 | P1 | RG-HEALTH | Add schema-aware, redacted and independently budgeted readiness checks | 5 | TASK-062 |
| GAP-HTTP-CONTRACT-063 | P5 | P1 | RG-HTTP-CONTRACT | Implement RFC-compliant Accept preference selection | 1 | TASK-063 |
| GAP-HTTP-CONTRACT-064 | P5 | P1 | RG-HTTP-CONTRACT | Unify ServeMux method registration, HEAD and Allow behavior | 1 | TASK-064 |
| GAP-HTTP-CONTRACT-065 | P5 | P1 | RG-HTTP-CONTRACT | Resolve v2 item surface, unknown-route precedence and strict query parsing | 5 | TASK-065 |
| GAP-CORS-066 | P5 | P1 | RG-CORS | Expose required browser headers and complete CORS Vary behavior | 2 | TASK-066 |
| GAP-CORS-067 | P5 | P2 | RG-CORS | Introduce route-aware CORS method/header policies | 1 | TASK-067 |
| GAP-SECURITY-068 | P5 | P1 | RG-SECURITY | Apply configured security headers and host-wide HSTS correctly | 2 | TASK-068 |
| GAP-SECURITY-069 | P5 | P1 | RG-SECURITY | Define and implement trusted X-Forwarded-For chain semantics | 1 | TASK-069 |
| GAP-ERROR-CONTRACT-070 | P5 | P1 | RG-ERROR-CONTRACT | Adopt an RFC 9457 Problem catalog and explicit cache policy | 6 | TASK-070 |
| GAP-SSE-071 | P5 | P0 | RG-SSE | Commit SSE immediately and handle post-commit errors without Problem corruption | 5 | TASK-071 |
| GAP-SSE-072 | P5 | P1 | RG-SSE | Add SSE snapshot, sequence and reconnect/resync semantics | 2 | TASK-072 |
| GAP-HTTP-CONTRACT-073 | P5 | P2 | RG-HTTP-CONTRACT | Complete Range validators, conditional requests and HEAD behavior | 2 | TASK-073 |
| GAP-PERFORMANCE-074 | P5 | P1 | RG-PERFORMANCE | Add bounded pagination to user-list endpoints | 1 | TASK-074 |
| GAP-PERFORMANCE-075 | P5 | P2 | RG-PERFORMANCE | Replace O(n) in-memory email lookup with an indexed structure | 2 | TASK-075 |
| GAP-QA-076 | P6 | P1 | RG-QA | Create production-faithful and explicitly scoped testkit fixtures | 4 | TASK-076 |
| GAP-QA-077 | P6 | P0 | RG-QA | Commit regression tests for queue, worker, Job state and metrics races | 3 | TASK-077 |
| GAP-QA-078 | P6 | P0 | RG-QA | Add PostgreSQL repository, migration and transaction integration tests | 1 | TASK-078 |
| GAP-OPENAPI-079 | P6 | P1 | RG-OPENAPI | Create the machine-readable OpenAPI source of truth | 2 | TASK-079 |
| GAP-OPENAPI-080 | P6 | P1 | RG-OPENAPI | Add pinned OpenAPI validation, codegen and runtime conformance checks | 4 | TASK-080 |
| GAP-QA-081 | P6 | P1 | RG-QA | Add fuzz/property tests for protocol parsers and state transitions | 1 | TASK-081 |
| GAP-QA-082 | P6 | P1 | RG-QA | Commit focused benchmarks for hot paths and allocations | 2 | TASK-082 |
| GAP-QA-083 | P6 | P1 | RG-QA | Add coverage reporting and risk-based CI gates | 1 | TASK-083 |
| GAP-QA-084 | P6 | P1 | RG-QA | Remove avoidable sleeps, port races and brittle representation assertions | 1 | TASK-084 |
| GAP-QA-085 | P6 | P1 | RG-QA | Provide cross-platform quality commands and drift/security gates | 6 | TASK-085 |
| GAP-QA-086 | P6 | P1 | RG-QA | Add integration CI for PostgreSQL, full binary, TLS/gRPC and observability smoke | 2 | TASK-086 |
| GAP-QA-087 | P6 | P2 | RG-QA | Harden configuration and representation tests | 2 | TASK-087 |
| GAP-PERF-088 | P7 | P1 | RG-PERF | Add explicit functional, baseline, observability and saturation config profiles | 5 | TASK-088 |
| GAP-PERF-089 | P7 | P1 | RG-PERF | Create versioned HTTP read/write/mixed load scenarios | 2 | TASK-089 |
| GAP-PERF-090 | P7 | P1 | RG-PERF | Add queue, worker and DB saturation experiments | 2 | TASK-090 |
| GAP-PERF-091 | P7 | P1 | RG-PERF | Add Profile latency, failure, retry and breaker experiments | 3 | TASK-091 |
| GAP-PERF-092 | P7 | P1 | RG-PERF | Add SSE connection, fan-out and slow-client experiments | 4 | TASK-092 |
| GAP-PERF-093 | P7 | P2 | RG-PERF | Compare direct gRPC, HTTP bridge, HTTP/1.1 and HTTP/2 | 2 | TASK-093 |
| GAP-PERF-094 | P7 | P1 | RG-PERF | Create reproducible PostgreSQL pool and query saturation experiments | 2 | TASK-094 |
| GAP-PERF-095 | P7 | P1 | RG-PERF | Test graceful and forced shutdown under active HTTP, gRPC, jobs and SSE | 2 | TASK-095 |
| GAP-PERF-096 | P7 | P2 | RG-PERF | Add lab-only mutex/block profiling and runtime capture protocol | 1 | TASK-096 |
| GAP-PERF-097 | P7 | P1 | RG-PERF | Measure telemetry overhead and Collector outage behavior | 4 | TASK-097 |
| GAP-PERF-098 | P7 | P2 | RG-PERF | Define benchmark/load baselines and regression review policy | 2 | TASK-098 |