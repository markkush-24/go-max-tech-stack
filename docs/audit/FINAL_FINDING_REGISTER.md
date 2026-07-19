# Final Finding Register

Raw audit findings: **218**.

These are evidence entries, not one-to-one implementation tasks. Positive/invariant findings are retained with preservation dispositions.

| Finding ID | Source | Finding | Disposition | Task IDs |
|---|---|---|---|---|
| A01-R1 | `AUDIT_01_REPOSITORY_INVENTORY.md` | Clean checkout is not reproducible | FIX | TASK-001, TASK-002, TASK-003 |
| A01-R2 | `AUDIT_01_REPOSITORY_INVENTORY.md` | CI and local quality tooling are not committed | FIX | TASK-001, TASK-003, TASK-085 |
| A01-R3 | `AUDIT_01_REPOSITORY_INVENTORY.md` | Uploaded project contains a large uncommitted persistence change | ACCEPT_RISK_CURRENT_WORKTREE | TASK-005 |
| A01-R4 | `AUDIT_01_REPOSITORY_INVENTORY.md` | Archive contains sensitive/local artifacts | FIX | TASK-004 |
| A01-R5 | `AUDIT_01_REPOSITORY_INVENTORY.md` | README/config drift around storage defaults | FIX | TASK-005, TASK-065 |
| A01-R6 | `AUDIT_01_REPOSITORY_INVENTORY.md` | Current context/documentation and actual route surface differ | FIX | TASK-005, TASK-065 |
| A01-R7 | `AUDIT_01_REPOSITORY_INVENTORY.md` | Full automated verification unavailable in this audit environment | FIX | TASK-007 |
| A02-E1 | `AUDIT_02_EXECUTION_LIFECYCLE.md` | Startup ownership is fragmented | FIX | TASK-016, TASK-017 |
| A02-E2 | `AUDIT_02_EXECUTION_LIFECYCLE.md` | Shutdown paths are asymmetric | FIX | TASK-016, TASK-017 |
| A02-E3 | `AUDIT_02_EXECUTION_LIFECYCLE.md` | Async work loses request/trace correlation at enqueue | FIX | TASK-027 |
| A02-E4 | `AUDIT_02_EXECUTION_LIFECYCLE.md` | gRPC bridge propagation is request-id only | FIX | TASK-021, TASK-031 |
| A02-E5 | `AUDIT_02_EXECUTION_LIFECYCLE.md` | Worker terminal write uses unbounded background context | FIX | TASK-012 |
| A02-E6 | `AUDIT_02_EXECUTION_LIFECYCLE.md` | Observability route identity is intentionally mux-derived | PRESERVE_INVARIANT | TASK-029, TASK-034 |
| A02-E7 | `AUDIT_02_EXECUTION_LIFECYCLE.md` | Streaming has a separate telemetry path | PRESERVE_SEPARATE_STREAM_POLICY | TASK-071, TASK-037 |
| A02-E8 | `AUDIT_02_EXECUTION_LIFECYCLE.md` | Request context propagation is good on synchronous boundaries | PRESERVE_INVARIANT | TASK-029 |
| A02-E9 | `AUDIT_02_EXECUTION_LIFECYCLE.md` | Readiness has one shared sequential budget | FIX | TASK-062 |
| A03-L1 | `AUDIT_03_LOGGING.md` | Async worker logs cannot correlate to the initiating request | FIX | TASK-027, TASK-028 |
| A03-L2 | `AUDIT_03_LOGGING.md` | No trace ID or span ID correlation exists | FIX | TASK-025, TASK-028 |
| A03-L3 | `AUDIT_03_LOGGING.md` | Component attribution is duplicated or misleading | FIX | TASK-023 |
| A03-L4 | `AUDIT_03_LOGGING.md` | Logger ownership is inconsistent and heavily dependent on global state | FIX | TASK-023 |
| A03-L5 | `AUDIT_03_LOGGING.md` | Logging format and level are not configurable | FIX | TASK-023 |
| A03-L6 | `AUDIT_03_LOGGING.md` | HTTP status in logs and metrics can be incorrect after repeated `WriteHeader` | FIX | TASK-008 |
| A03-L7 | `AUDIT_03_LOGGING.md` | Privacy and redaction policy is not defined or enforced | FIX | TASK-024, TASK-022 |
| A03-L8 | `AUDIT_03_LOGGING.md` | Security decisions have counters but no safe audit events | FIX | TASK-024, TASK-022 |
| A03-L9 | `AUDIT_03_LOGGING.md` | Retry behavior is not visible as a coherent operation | FIX | TASK-024, TASK-036 |
| A03-L10 | `AUDIT_03_LOGGING.md` | SSE lifecycle is absent from logs | FIX | TASK-024 |
| A03-L11 | `AUDIT_03_LOGGING.md` | Lifecycle logs do not provide a complete shutdown narrative | FIX | TASK-024 |
| A03-L12 | `AUDIT_03_LOGGING.md` | Field naming and duration units are inconsistent | FIX | TASK-023 |
| A03-L13 | `AUDIT_03_LOGGING.md` | Access logging can become noisy for probes and long-lived streams | FIX | TASK-024 |
| A03-L14 | `AUDIT_03_LOGGING.md` | Unexpected errors may produce two records without an explicit relationship | FIX | TASK-024 |
| A04-M1 | `AUDIT_04_METRICS.md` | `queue_depth` is permanently bound to the first `Queue` instance | FIX | TASK-032 |
| A04-M2 | `AUDIT_04_METRICS.md` | Process-global metric singletons break isolation and blur instance ownership | FIX | TASK-032 |
| A04-M3 | `AUDIT_04_METRICS.md` | Recorded HTTP status can disagree with the wire status | FIX | TASK-008 |
| A04-M4 | `AUDIT_04_METRICS.md` | Latency metrics cannot describe tail latency | FIX | TASK-034 |
| A04-M5 | `AUDIT_04_METRICS.md` | HTTP metric inclusion policy is heuristic and inconsistent for streaming | FIX | TASK-034 |
| A04-M6 | `AUDIT_04_METRICS.md` | HTTP metrics omit important service signals | FIX | TASK-034 |
| A04-M7 | `AUDIT_04_METRICS.md` | `jobs_total` is a transition-event counter with incomplete terminal coverage | FIX | TASK-035 |
| A04-M8 | `AUDIT_04_METRICS.md` | Queue observability is too coarse even aside from the first-instance defect | FIX | TASK-035 |
| A04-M9 | `AUDIT_04_METRICS.md` | Outbound metrics describe physical attempts, not logical operations | FIX | TASK-036 |
| A04-M10 | `AUDIT_04_METRICS.md` | Authentication's first increment for a new kind can be lost under concurrency | FIX | TASK-033 |
| A04-M11 | `AUDIT_04_METRICS.md` | Security and admission-control metrics lack actionable dimensions | FIX | TASK-037 |
| A04-M12 | `AUDIT_04_METRICS.md` | SSE metric names and semantics are weak | FIX | TASK-037 |
| A04-M13 | `AUDIT_04_METRICS.md` | gRPC has logging but no metrics | FIX | TASK-037 |
| A04-M14 | `AUDIT_04_METRICS.md` | Database metrics stop at pool snapshots | FIX | TASK-037 |
| A04-M15 | `AUDIT_04_METRICS.md` | The runtime endpoint is diagnostic JSON, not an operational metrics pipeline | FIX | TASK-038 |
| A04-M16 | `AUDIT_04_METRICS.md` | Metric test coverage is narrow and process-global state makes it fragile | FIX | TASK-032 |
| A04-M17 | `AUDIT_04_METRICS.md` | `testkit` constructs disconnected stream hubs while metrics aggregate them globally | FIX | TASK-014, TASK-076 |
| A04-M18 | `AUDIT_04_METRICS.md` | No standard exporter, resource identity or cardinality policy exists | FIX | TASK-025, TASK-040 |
| A05-T1 | `AUDIT_05_TRACING_CONTEXT.md` | Distributed tracing is not implemented | FIX | TASK-025, TASK-029, TASK-030, TASK-031 |
| A05-T2 | `AUDIT_05_TRACING_CONTEXT.md` | Async enqueue is a hard propagation break | FIX | TASK-027 |
| A05-T3 | `AUDIT_05_TRACING_CONTEXT.md` | Workers use one shared lifecycle context rather than a per-job operation context | FIX | TASK-027 |
| A05-T4 | `AUDIT_05_TRACING_CONTEXT.md` | `markJobFailed(context.Background())` severs all operation context | FIX | TASK-027 |
| A05-T5 | `AUDIT_05_TRACING_CONTEXT.md` | Job persistence has no correlation or propagation metadata | FIX | TASK-027 |
| A05-T6 | `AUDIT_05_TRACING_CONTEXT.md` | Outbound correlation is duplicated between context and an explicit string parameter | FIX | TASK-028, TASK-030 |
| A05-T7 | `AUDIT_05_TRACING_CONTEXT.md` | Outbound HTTP propagates request ID but not standard trace context | FIX | TASK-030 |
| A05-T8 | `AUDIT_05_TRACING_CONTEXT.md` | Retry attempts have no explicit span model | FIX | TASK-030 |
| A05-T9 | `AUDIT_05_TRACING_CONTEXT.md` | The HTTP-to-gRPC bridge propagates only request ID | FIX | TASK-031, TASK-021 |
| A05-T10 | `AUDIT_05_TRACING_CONTEXT.md` | `metadata.NewOutgoingContext` replaces existing outgoing metadata | FIX | TASK-031, TASK-021 |
| A05-T11 | `AUDIT_05_TRACING_CONTEXT.md` | gRPC request-ID metadata is trusted without sanitization | FIX | TASK-021 |
| A05-T12 | `AUDIT_05_TRACING_CONTEXT.md` | A generated gRPC request ID is not returned to the client | FIX | TASK-021 |
| A05-T13 | `AUDIT_05_TRACING_CONTEXT.md` | There is no gRPC client interceptor and no stream interceptor baseline | FIX | TASK-031, TASK-021 |
| A05-T14 | `AUDIT_05_TRACING_CONTEXT.md` | Security identity remains request-local and stops at integration boundaries | FIX | TASK-027 |
| A05-T15 | `AUDIT_05_TRACING_CONTEXT.md` | SSE connection cancellation is correct, but event causality is absent | FIX | TASK-027 |
| A05-T16 | `AUDIT_05_TRACING_CONTEXT.md` | Route-aware span naming must preserve the existing low-cardinality invariant | FIX | TASK-029, TASK-040 |
| A05-T17 | `AUDIT_05_TRACING_CONTEXT.md` | Startup and shutdown contexts are separated from request cancellation, but telemetry lifecycle is absent | FIX | TASK-026 |
| A05-T18 | `AUDIT_05_TRACING_CONTEXT.md` | DB/startup operations are only partly connected to application cancellation | FIX | TASK-016, TASK-062 |
| A05-T19 | `AUDIT_05_TRACING_CONTEXT.md` | Propagation tests cover request ID and some cancellation, not distributed context | FIX | TASK-046 |
| A05-T20 | `AUDIT_05_TRACING_CONTEXT.md` | There is no documented propagation and privacy contract | FIX | TASK-040, TASK-024 |
| A06-G1 | `AUDIT_06_TELEMETRY_PIPELINE_ARCHITECTURE.md` | Telemetry bootstrap ownership | FIX | TASK-025 |
| A06-G2 | `AUDIT_06_TELEMETRY_PIPELINE_ARCHITECTURE.md` | Shared resource identity | FIX | TASK-025 |
| A06-G3 | `AUDIT_06_TELEMETRY_PIPELINE_ARCHITECTURE.md` | OTLP export | FIX | TASK-025, TASK-038 |
| A06-G4 | `AUDIT_06_TELEMETRY_PIPELINE_ARCHITECTURE.md` | Collector | FIX | TASK-038 |
| A06-G5 | `AUDIT_06_TELEMETRY_PIPELINE_ARCHITECTURE.md` | Tempo | FIX | TASK-038 |
| A06-G6 | `AUDIT_06_TELEMETRY_PIPELINE_ARCHITECTURE.md` | Prometheus operational pipeline | FIX | TASK-038 |
| A06-G7 | `AUDIT_06_TELEMETRY_PIPELINE_ARCHITECTURE.md` | Loki/log shipping | FIX | TASK-039 |
| A06-G8 | `AUDIT_06_TELEMETRY_PIPELINE_ARCHITECTURE.md` | Grafana provisioning | FIX | TASK-038, TASK-044 |
| A06-G9 | `AUDIT_06_TELEMETRY_PIPELINE_ARCHITECTURE.md` | Telemetry failure policy | FIX | TASK-026 |
| A06-G10 | `AUDIT_06_TELEMETRY_PIPELINE_ARCHITECTURE.md` | Telemetry shutdown/flush | FIX | TASK-026 |
| A06-G11 | `AUDIT_06_TELEMETRY_PIPELINE_ARCHITECTURE.md` | Collector self-observability | FIX | TASK-046, TASK-097 |
| A06-G12 | `AUDIT_06_TELEMETRY_PIPELINE_ARCHITECTURE.md` | Signal cardinality contract | FIX | TASK-040 |
| A07-SLO-1 | `AUDIT_07_SLI_SLO_DASHBOARDS_ALERTING.md` | Current data cannot support trustworthy SLOs | FIX | TASK-038, TASK-040 |
| A07-SLO-2 | `AUDIT_07_SLI_SLO_DASHBOARDS_ALERTING.md` | Current status attribution can corrupt availability | FIX | TASK-008, TASK-041 |
| A07-SLO-3 | `AUDIT_07_SLI_SLO_DASHBOARDS_ALERTING.md` | Service-class mapping is not formally encoded | FIX | TASK-040, TASK-034 |
| A07-SLO-4 | `AUDIT_07_SLI_SLO_DASHBOARDS_ALERTING.md` | Average-only latency cannot define latency objectives | FIX | TASK-034, TASK-041 |
| A07-SLO-5 | `AUDIT_07_SLI_SLO_DASHBOARDS_ALERTING.md` | Async acceptance and completion are conflated/incomplete | FIX | TASK-035, TASK-042 |
| A07-SLO-6 | `AUDIT_07_SLI_SLO_DASHBOARDS_ALERTING.md` | Outbound user outcome is not represented | FIX | TASK-036, TASK-042 |
| A07-SLO-7 | `AUDIT_07_SLI_SLO_DASHBOARDS_ALERTING.md` | gRPC is absent from SLO telemetry | FIX | TASK-037, TASK-042 |
| A07-SLO-8 | `AUDIT_07_SLI_SLO_DASHBOARDS_ALERTING.md` | SSE delivery reliability is not measurable | FIX | TASK-037, TASK-042 |
| A07-SLO-9 | `AUDIT_07_SLI_SLO_DASHBOARDS_ALERTING.md` | No alert/dashboard-as-code artifacts exist | FIX | TASK-043, TASK-044, TASK-045 |
| A07-SLO-10 | `AUDIT_07_SLI_SLO_DASHBOARDS_ALERTING.md` | Low traffic requires explicit guardrails | FIX | TASK-043, TASK-045 |
| A07-SLO-11 | `AUDIT_07_SLI_SLO_DASHBOARDS_ALERTING.md` | Operational and user-facing alerts must be separated | FIX | TASK-043, TASK-045 |
| A07-SLO-12 | `AUDIT_07_SLI_SLO_DASHBOARDS_ALERTING.md` | Numerical objectives are provisional until baseline audit | FIX | TASK-088, TASK-098 |
| A08-F1 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | There is no unified application supervisor | FIX | TASK-016 |
| A08-F2 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | Worker and gRPC components start before construction is complete | FIX | TASK-016 |
| A08-F3 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | `os.Exit(1)` makes deferred cleanup in `main` unusable | FIX | TASK-016 |
| A08-F4 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | Unexpected HTTP/HTTPS server error cleanup is asymmetric | FIX | TASK-016 |
| A08-F5 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | A gRPC serve failure can produce a successful process exit | FIX | TASK-016 |
| A08-F6 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | Forced HTTP shutdown is reported as success | FIX | TASK-017 |
| A08-F7 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | gRPC timeout is deliberately suppressed by `APIServer.Run` | FIX | TASK-017 |
| A08-F8 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | Shutdown has no global deadline | FIX | TASK-017 |
| A08-F9 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | Worker wait and terminal repair consume the same context budget | FIX | TASK-012 |
| A08-F10 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | Worker terminal fallback uses an unbounded background context | FIX | TASK-012 |
| A08-F11 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | A timed-out `WorkerPool.Stop` leaves a waiter goroutine behind | FIX | TASK-012 |
| A08-F12 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | Closing the hub before shutting down HTTP is useful but creates an admission race | FIX | TASK-016, TASK-060 |
| A08-F13 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | SSE timing has a boundary race with server `WriteTimeout` | FIX | TASK-071, TASK-088 |
| A08-F14 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | Component stop order is not explicitly tied to telemetry completion | FIX | TASK-026, TASK-017 |
| A08-F15 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | Collector/exporter outage policy is not implemented | FIX | TASK-026, TASK-046 |
| A08-F16 | `AUDIT_08_TELEMETRY_RELIABILITY_SHUTDOWN.md` | Current tests do not cover shutdown outcome truth | FIX | TASK-046, TASK-095 |
| A09-F1 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | No persistent benchmark or load-test suite exists | FIX | TASK-082, TASK-089 |
| A09-F2 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | Current defaults are not a high-load profile | FIX | TASK-088 |
| A09-F3 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | SSE occupies the shared bulkhead for the full stream lifetime | FIX | TASK-060 |
| A09-F4 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | One global bulkhead creates cross-route head-of-line and noisy-neighbor behavior | FIX | TASK-060 |
| A09-F5 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | One global rate limiter mixes different traffic classes | FIX | TASK-060 |
| A09-F6 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | The user list path is unbounded | FIX | TASK-074 |
| A09-F7 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | Memory repository email uniqueness is O(n) | FIX | TASK-075 |
| A09-F8 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | Worker, queue and DB pool sizing have no explicit capacity relationship | FIX | TASK-088, TASK-090, TASK-094 |
| A09-F9 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | Queue observability cannot support capacity tuning | FIX | TASK-035, TASK-090 |
| A09-F10 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | Outbound active connection count is unlimited per host | FIX | TASK-059, TASK-091 |
| A09-F11 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | Retry amplification is not bounded by a dependency-level concurrency policy | FIX | TASK-059, TASK-091 |
| A09-F12 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | Outbound response draining is not byte-bounded | FIX | TASK-057 |
| A09-F13 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | SSE fan-out uses one global mutex across all job IDs | FIX | TASK-092 |
| A09-F14 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | Every SSE connection owns a goroutine, ticker and socket | FIX | TASK-092 |
| A09-F15 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | SSE is excluded from ordinary HTTP metrics but not from resource accounting | FIX | TASK-037, TASK-092 |
| A09-F16 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | HTTP access logging can become a load bottleneck | FIX | TASK-023, TASK-097 |
| A09-F17 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | Current expvar metrics add per-request string work and shared-map contention | FIX | TASK-032, TASK-097 |
| A09-F18 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | gRPC has no explicit load controls or measurements | FIX | TASK-037, TASK-093 |
| A09-F19 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | The Range endpoint is not a meaningful large-object streaming workload | FIX | TASK-073 |
| A09-F20 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | Block and mutex profiles are not explicitly enabled | FIX | TASK-096 |
| A09-F21 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | No process/container/host metrics exist yet | FIX | TASK-037, TASK-097 |
| A09-F22 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | PostgreSQL baseline environment is not reproducibly constrained | FIX | TASK-088, TASK-094 |
| A09-F23 | `AUDIT_09_PERFORMANCE_LOAD_READINESS.md` | No performance regression policy exists | FIX | TASK-098 |
| A10-F1 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | `WorkerPool.Stop` is not idempotent | FIX | TASK-011, TASK-012 |
| A10-F2 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | Active worker can overwrite shutdown terminal state | FIX | TASK-009, TASK-010, TASK-053 |
| A10-F3 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | Timed-out Stop allows unsafe restart of the same pool | FIX | TASK-011, TASK-012 |
| A10-F4 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | `WaitGroup` lifecycle can be reused before previous Wait returns | FIX | TASK-011, TASK-012 |
| A10-F5 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | Stop timeout can leak waiter goroutines | FIX | TASK-011, TASK-012 |
| A10-F6 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | `running` does not represent actual worker health | FIX | TASK-011, TASK-012 |
| A10-F7 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | Canceled context can still enqueue work | FIX | TASK-013 |
| A10-F8 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | Queue stop is not a strict barrier for an already-entered Enqueue | FIX | TASK-013 |
| A10-F9 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | Queued events and metrics can appear after running/succeeded | FIX | TASK-014 |
| A10-F10 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | Job transitions consume work without guaranteed terminality | FIX | TASK-009, TASK-010, TASK-053 |
| A10-F11 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | Async enqueue rollback uses the request context | FIX | TASK-050, TASK-012 |
| A10-F12 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | gRPC Runtime has no single-start/single-stop ownership | FIX | TASK-015 |
| A10-F13 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | APIServer and component lifecycle remain path-dependent | FIX | TASK-016 |
| A10-F14 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | Testkit wires two different event hubs | FIX | TASK-014, TASK-076 |
| A10-F15 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | Global authentication first-increment update is not atomic | FIX | TASK-033 |
| A10-F16 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | Bulkhead gauge can transiently disagree with semaphore occupancy | FIX | TASK-033 |
| A10-F17 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | Stream Hub synchronization is currently sound for tested send/close paths | PRESERVE_LOCKING_INVARIANT | TASK-092 |
| A10-F18 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | Memory repositories have good locking and copy discipline | PRESERVE_COPY_LOCKING_INVARIANT | TASK-075 |
| A10-F19 | `AUDIT_10_CONCURRENCY_LIFECYCLE_CORRECTNESS.md` | Full concurrency proof is still unavailable | FIX | TASK-077 |
| A11-F1 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Accepted async work is not durable or crash-recoverable | FIX | TASK-047, TASK-051 |
| A11-F2 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | User creation and job terminal success are not atomic | FIX | TASK-054 |
| A11-F3 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Shared-database shutdown repair is unsafe for horizontal workers | FIX | TASK-048 |
| A11-F4 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Queue-full compensation can create orphan jobs and amplify DB load | FIX | TASK-049, TASK-050 |
| A11-F5 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | No poison-job, retry, DLQ or reconciliation model exists | FIX | TASK-052, TASK-053 |
| A11-F6 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Worker panics are not contained or supervised | FIX | TASK-056 |
| A11-F7 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | SSE can permanently miss the terminal event | FIX | TASK-072 |
| A11-F8 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Streaming errors are routed through Problem+JSON after the stream may be committed | FIX | TASK-071 |
| A11-F9 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Dependency/cancellation errors collapse into misleading HTTP 500 responses | FIX | TASK-061, TASK-070 |
| A11-F10 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | gRPC error mapping loses cancellation, deadline and availability semantics | FIX | TASK-061, TASK-070 |
| A11-F11 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | HTTP-to-gRPC bridge has no explicit call budget | FIX | TASK-021 |
| A11-F12 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | No circuit breaker or retry budget protects the Profile dependency | FIX | TASK-059 |
| A11-F13 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Backoff can overflow before applying its configured cap | FIX | TASK-058 |
| A11-F14 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Retry policy is too coarse for HTTP status and server guidance | FIX | TASK-058 |
| A11-F15 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Profile response validation is not strict enough | FIX | TASK-057 |
| A11-F16 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Profile 4xx semantics are flattened to HTTP 502 | FIX | TASK-061, TASK-070 |
| A11-F17 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | POST operations have no idempotency contract | FIX | TASK-055 |
| A11-F18 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Startup readiness validates connectivity, not schema or migrations | FIX | TASK-062 |
| A11-F19 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Readiness checks are sequential under one shared 200 ms deadline | FIX | TASK-062 |
| A11-F20 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Global admission controls enable cross-route noisy-neighbor failure | FIX | TASK-060 |
| A11-F21 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Queue overload response lacks durable admission semantics | FIX | TASK-049, TASK-050 |
| A11-F22 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Job model lacks recovery metadata | FIX | TASK-047, TASK-051 |
| A11-F23 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | `WithinTransaction` exists but is not integrated into application operations | FIX | TASK-054 |
| A11-F24 | `AUDIT_11_RESILIENCE_FAULT_HANDLING.md` | Fault-injection coverage is insufficient | FIX | TASK-091, TASK-095, TASK-086 |
| A12-F1 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | Direct gRPC is an authentication and authorization bypass | FIX | TASK-018, TASK-019, TASK-020 |
| A12-F2 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | gRPC status and metadata contracts are incomplete | FIX | TASK-021 |
| A12-F3 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | HTTP content negotiation violates Accept quality semantics | FIX | TASK-063 |
| A12-F4 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | HEAD and Allow contracts are inconsistent | FIX | TASK-064 |
| A12-F5 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | Browser clients cannot read important response headers | FIX | TASK-066 |
| A12-F6 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | CORS Vary coverage is incomplete on denial paths | FIX | TASK-066 |
| A12-F7 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | CORS policy is global rather than route-specific | FIX | TASK-067 |
| A12-F8 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | Security-header configuration is partly ignored | FIX | TASK-068 |
| A12-F9 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | HSTS is scoped to the API subtree instead of the HTTPS host | FIX | TASK-068 |
| A12-F10 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | TLS and HTTP2 have no explicit production security profile | FIX | TASK-022, TASK-019, TASK-093 |
| A12-F11 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | Development JWT defaults are unsafe without an environment guard | FIX | TASK-022 |
| A12-F12 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | Trusted X-Forwarded-For semantics depend on undocumented proxy sanitization | FIX | TASK-069 |
| A12-F13 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | Public readiness responses expose raw internal errors | FIX | TASK-062 |
| A12-F14 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | Problem Details source and machine contract are outdated/incomplete | FIX | TASK-070, TASK-061 |
| A12-F15 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | Some HTTP status mappings are semantically misleading | FIX | TASK-070, TASK-061 |
| A12-F16 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | Problem responses containing request-specific data have no cache policy | FIX | TASK-070, TASK-061 |
| A12-F17 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | SSE does not establish the stream immediately | FIX | TASK-071 |
| A12-F18 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | SSE delivery has no resumable contract | FIX | TASK-072 |
| A12-F19 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | Streaming errors can corrupt an already committed response | FIX | TASK-071 |
| A12-F20 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | Range works, but the export representation lacks validators and consistent HEAD | FIX | TASK-073 |
| A12-F21 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | v2 contains dead item-handler code but no item endpoint contract | FIX | TASK-065 |
| A12-F22 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | Unknown-route behavior contradicts the deny-by-default policy comment | FIX | TASK-065 |
| A12-F23 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | Query parameter contract is permissive and undocumented | FIX | TASK-065 |
| A12-F24 | `AUDIT_12_API_SECURITY_PROTOCOL_CONTRACTS.md` | No machine-readable HTTP contract exists | FIX | TASK-079, TASK-080 |
| A13-Q1 | `AUDIT_13_TESTING_CI_QUALITY.md` | The quality system is not part of the tracked repository | FIX | TASK-001, TASK-085 |
| A13-Q2 | `AUDIT_13_TESTING_CI_QUALITY.md` | Generated protobuf code is required but untracked | FIX | TASK-001, TASK-002, TASK-080 |
| A13-Q3 | `AUDIT_13_TESTING_CI_QUALITY.md` | `testkit.NewServer` is not production full-stack | FIX | TASK-076 |
| A13-Q4 | `AUDIT_13_TESTING_CI_QUALITY.md` | `testkit` wires producers and SSE consumers to different Hubs | FIX | TASK-014, TASK-076 |
| A13-Q5 | `AUDIT_13_TESTING_CI_QUALITY.md` | The highest-risk concurrency packages have no direct tests | FIX | TASK-077 |
| A13-Q6 | `AUDIT_13_TESTING_CI_QUALITY.md` | PostgreSQL and migrations are untested | FIX | TASK-078 |
| A13-Q7 | `AUDIT_13_TESTING_CI_QUALITY.md` | No OpenAPI or runtime contract test exists | FIX | TASK-079, TASK-080 |
| A13-Q8 | `AUDIT_13_TESTING_CI_QUALITY.md` | There are no fuzz targets | FIX | TASK-081 |
| A13-Q9 | `AUDIT_13_TESTING_CI_QUALITY.md` | There are no committed benchmarks or load scripts | FIX | TASK-082, TASK-089 |
| A13-Q10 | `AUDIT_13_TESTING_CI_QUALITY.md` | No coverage report or coverage policy exists | FIX | TASK-083 |
| A13-Q11 | `AUDIT_13_TESTING_CI_QUALITY.md` | Global `expvar` state prevents test isolation | FIX | TASK-032, TASK-046 |
| A13-Q12 | `AUDIT_13_TESTING_CI_QUALITY.md` | Timing-based tests contain avoidable flakiness | FIX | TASK-084 |
| A13-Q13 | `AUDIT_13_TESTING_CI_QUALITY.md` | Several tests assert brittle representation details | FIX | TASK-087 |
| A13-Q14 | `AUDIT_13_TESTING_CI_QUALITY.md` | Configuration tests are environment-sensitive and incomplete | FIX | TASK-006, TASK-087 |
| A13-Q15 | `AUDIT_13_TESTING_CI_QUALITY.md` | CI does not verify generated or dependency state | FIX | TASK-003, TASK-085 |
| A13-Q16 | `AUDIT_13_TESTING_CI_QUALITY.md` | CI does not exercise deployment/integration topology | FIX | TASK-086 |
| A13-Q17 | `AUDIT_13_TESTING_CI_QUALITY.md` | Quality scripts are Windows-only | FIX | TASK-085 |
| A13-Q18 | `AUDIT_13_TESTING_CI_QUALITY.md` | CI tool installation and workflow integrity can be hardened | FIX | TASK-085 |
| A13-Q19 | `AUDIT_13_TESTING_CI_QUALITY.md` | Vulnerability scanning covers only the built application binary | FIX | TASK-085 |
| A13-Q20 | `AUDIT_13_TESTING_CI_QUALITY.md` | Previously discovered defects lack regression tests | FIX | TASK-077, TASK-046, TASK-080 |