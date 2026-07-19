# Audit 06 — Telemetry Pipeline Architecture

## Scope

This pass defines the target observability architecture for the uploaded `pet-study` working tree. It does **not** add OpenTelemetry, Prometheus, Grafana, Tempo or Loki yet.

The goal is to prevent a fragmented implementation in which each subsystem chooses its own exporter, labels, lifecycle and configuration.

## Evidence from the current project

### Application composition and lifecycle

- `cmd/api/main.go:37-45` creates a fixed text `slog` logger before configuration is loaded.
- `cmd/api/main.go:61-64` loads application configuration.
- `cmd/api/main.go:71-74` obtains the process-global `expvar` metrics singleton and creates the SSE hub.
- `cmd/api/main.go:111-117` starts the worker pool before HTTP listeners are bound.
- `cmd/api/main.go:119-131` creates the shared outbound HTTP transport and retry/client wrapper chain.
- `cmd/api/main.go:140-163` creates and starts the gRPC runtime and loopback client.
- `cmd/api/main.go:243-259` assembles the global HTTP middleware chain.
- `cmd/api/main.go:261-264` transfers lifecycle control to `APIServer.Run`.

### Existing signal implementations

- Logs: `log/slog`, fixed text handler, process stderr.
- Metrics: package-global `expvar` variables in `internal/metrics`, middleware packages, queue, stream and DB helpers.
- Runtime diagnostics: `expvar`, `runtime/metrics`, `runtime.MemStats`, `pprof` behind the debug/admin boundary.
- Correlation: request ID through HTTP and the loopback unary gRPC bridge.
- Distributed tracing: absent.
- Standard operational metrics endpoint: absent.
- Telemetry collector/backend configuration: absent.

### Existing infrastructure

`docker-compose.yml` currently contains PostgreSQL only. There are no services or configuration files for:

- OpenTelemetry Collector;
- Prometheus;
- Grafana;
- Tempo;
- Loki;
- Grafana Alloy or another log agent.

`go.mod` has no direct OpenTelemetry modules.

## Architectural decision

### Target local-laboratory pipeline

```text
pet-study process
  ├─ traces ─────────── OTLP ───────────────┐
  ├─ operational metrics ─ OTLP ────────────┤
  └─ JSON slog stdout ─ log collector ──────┤
                                             v
                              OpenTelemetry Collector / Alloy
                                  ├─ traces  -> Tempo
                                  ├─ metrics -> Prometheus endpoint
                                  └─ logs    -> Loki

Prometheus scrapes Collector-exported application metrics
Grafana queries Prometheus + Tempo + Loki
```

### Concrete first production-like version

1. **Traces:** application OTel SDK -> OTLP/gRPC -> OpenTelemetry Collector -> Tempo.
2. **Metrics:** application OTel SDK -> OTLP/gRPC -> OpenTelemetry Collector -> Prometheus exporter endpoint -> Prometheus scrape.
3. **Logs:** existing `slog` changes to structured JSON stdout; a log agent sends logs to Loki.
4. **Grafana:** provisioned data sources for Prometheus, Tempo and Loki, including trace-to-logs and logs-to-traces correlation.
5. **Existing `expvar`:** retained temporarily as a debug/compatibility surface, not the long-term operational source of truth.

## Why the application should send to a Collector

The application should not contain backend-specific clients for Tempo, Loki and Prometheus.

The Collector provides a vendor-neutral boundary for:

- OTLP reception;
- batching;
- memory limiting and backpressure;
- filtering/redaction;
- retries and sending queues;
- backend replacement;
- Collector self-monitoring.

This also allows the same application binary to run with:

- telemetry disabled;
- a local debug exporter;
- a local Collector;
- a remote Collector endpoint.

## Signal-by-signal decisions

### 1. Traces

Status: **must be implemented first**.

The Go trace API/SDK is stable and directly addresses the largest current gap.

Required path:

```text
HTTP server span
  -> service/internal spans
  -> SQL/repository spans
  -> outbound logical operation
       -> physical HTTP attempt spans
  -> enqueue producer span
       -> worker consumer/process span

HTTP bridge span
  -> gRPC client span
       -> gRPC server span
```

Required instrumentation points:

- incoming `net/http` requests;
- outbound `http.Transport`;
- gRPC client and server;
- queue enqueue/process boundary;
- worker job processing;
- selected service/repository operations;
- retry attempts and backoff events;
- SSE connection lifecycle only where useful.

Route names must use the matched `r.Pattern`. Concrete user IDs/job IDs must not be placed in span names.

### 2. Metrics

Status: **must migrate from process-global expvar to DI-owned instruments**.

The OTel MeterProvider is owned by the application composition root. Components receive narrow observer interfaces or instrument groups rather than using package globals.

Operational metrics should include:

- HTTP RED metrics and histograms;
- gRPC RED metrics;
- outbound physical-attempt and logical-operation metrics;
- queue accepted/rejected/depth/capacity/wait-age metrics;
- job queue wait, processing and end-to-end histograms;
- worker activity;
- bulkhead/rate limiter state;
- SSE connection lifecycle and delivery failures;
- DB pool and selected DB-operation metrics;
- runtime/process metrics.

`expvar` can remain during migration for `/debug/vars`, but the implementation must avoid permanent dual-write divergence.

### 3. Logs

Status: **keep `slog`, normalize it, then aggregate**.

The Go OpenTelemetry logs signal is less mature than traces and metrics. The first implementation should not require application-direct OTLP logs.

Recommended initial path:

```text
slog.JSONHandler -> stdout/stderr -> Grafana Alloy (or equivalent) -> Loki
```

The application must add context-aware fields:

- `service.name` / equivalent stable service field;
- `service.version`;
- `deployment.environment`;
- `component` exactly once;
- `request_id`;
- `trace_id`;
- `span_id`;
- stable `http.route` / gRPC method / upstream name;
- normalized error kind.

Do not index request ID, trace ID, user ID, job ID or client IP as Loki stream labels. They should remain log fields/structured metadata.

A later experiment may evaluate the OTel slog bridge and direct OTLP log export, but it is not a prerequisite for the first reliable pipeline.

## Application package ownership

Introduce a composition-root-owned package, conceptually:

```text
internal/telemetry/
  config.go
  resource.go
  runtime.go
  traces.go
  metrics.go
  logging.go
  propagation.go
  shutdown.go
```

Recommended public shape:

```go
runtime, err := telemetry.New(ctx, cfg.Telemetry, telemetry.BuildInfo{...})
if err != nil { ... }
defer runtime.Shutdown(shutdownCtx)

logger := runtime.Logger()
meterProvider := runtime.MeterProvider()
tracerProvider := runtime.TracerProvider()
propagator := runtime.Propagator()
```

The exact API should remain small. Components should not import Collector, Tempo, Loki or Grafana-specific code.

## Bootstrap order

Current logging is initialized before config. The target bootstrap order should be:

```text
1. create minimal bootstrap logger
2. load and validate config
3. initialize telemetry resource/providers/exporters
4. install final context-aware logger and OTel global providers/propagator
5. create repositories/clients/queue/workers/gRPC/HTTP
6. start runtime components
7. run application lifecycle
8. stop admissions and runtime components
9. flush and shut down telemetry last
```

If telemetry initialization fails:

- invalid telemetry configuration: startup error;
- Collector temporarily unreachable: application should normally still start and exporters retry/drop according to bounded policy;
- local explicit strict mode may treat exporter initialization failure as fatal for integration tests.

The application `/readyz` should **not** depend on Collector availability. Observability failure must not make the business service unavailable.

## Telemetry runtime resource

Every signal must share the same resource attributes.

Minimum stable set:

```text
service.name=pet-study
service.version=<build version>
service.instance.id=<runtime instance>
deployment.environment.name=<local|test|stage|prod-like>
```

Optional bounded attributes:

```text
service.namespace=pet-study
host.name / container.id (detector-provided)
```

Do not hard-code an instance ID shared by all replicas.

Prefer standard OpenTelemetry environment variables for exporter/resource settings rather than inventing duplicate application-specific names.

## Configuration decision

Use standard OTel variables where possible:

```text
OTEL_SERVICE_NAME=pet-study
OTEL_RESOURCE_ATTRIBUTES=service.version=...,deployment.environment.name=local
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
OTEL_TRACES_EXPORTER=otlp
OTEL_METRICS_EXPORTER=otlp
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=1.0
```

Application-specific toggles may be limited to:

```text
TELEMETRY_ENABLED=true|false
TELEMETRY_STRICT_STARTUP=false
LOG_FORMAT=json|text
LOG_LEVEL=debug|info|warn|error
LOG_ADD_SOURCE=false|true
```

Avoid maintaining two independent endpoint/protocol/sampler configuration systems.

## Sampling policy

Initial laboratory policy:

- local functional development: `ParentBased(AlwaysSample)` or ratio `1.0`;
- load tests: configurable parent-based ratio, starting around `0.1` or lower depending on traffic;
- preserve upstream sampling decisions;
- never implement random per-span sampling independently inside handlers.

Tail sampling in the Collector is a later experiment, not the first implementation. It adds state, latency and operational complexity.

## Propagation policy

Set a composite global propagator containing:

- W3C Trace Context;
- W3C Baggage only if a concrete bounded use case is defined.

Do not put secrets, JWTs, user PII or arbitrary claims in baggage.

### HTTP

- extract incoming trace headers at the trust boundary;
- inject into outbound Profile requests;
- preserve request ID as a separate correlation identifier.

### gRPC

Use current `otelgrpc` stats handlers on both client and server. Existing request-ID logging interceptor remains separate and must be fixed for validation/response metadata.

### Queue/worker

Do not store `context.Context` in the queue.

Extend the work envelope with explicit metadata:

```text
request_id
traceparent
tracestate
optional bounded baggage
actor/tenant only if required
created/enqueued timestamp
```

Workers create a new operation context from the pool lifecycle context, extract the carrier, and start a process/consumer span. The initial HTTP cancellation must not cancel accepted async work after `202`.

For later durable brokers, the same metadata maps naturally to message headers.

## HTTP instrumentation placement

A generic outer HTTP wrapper cannot be allowed to replace the existing `ServeMux` path or lose `r.Pattern`.

Required invariant:

- `ServeMux.ServeHTTP` remains the matcher;
- the completed server span receives `http.route` from the matched pattern;
- unmatched requests use a bounded fallback;
- raw URL/query values do not become span names or metric labels.

Implementation should either use route-aware OTel helpers at registration time or update the span after `ServeMux` has populated `r.Pattern`.

## Outbound instrumentation placement

Current custom transport is created in `internal/outbound/httpclient/client.go:8-19`.

The target chain should preserve transport ownership:

```text
configured base *http.Transport
  -> OTel instrumented RoundTripper
  -> http.Client
```

`CloseIdleConnections` must still close the underlying base transport.

Retry instrumentation must distinguish:

- one logical profile fetch;
- N physical HTTP attempts;
- final result;
- retry/backoff events.

## gRPC instrumentation placement

Current client: `internal/transport/grpcclient/grpcclient.go:18-21`.

Current server: `internal/transport/grpcserver/runtime.go:36-40`.

Add OTel gRPC stats handlers:

```text
grpc.WithStatsHandler(otelgrpc.NewClientHandler(...))
grpc.StatsHandler(otelgrpc.NewServerHandler(...))
```

Do not replace existing business/auth/request-ID interceptors with tracing interceptors. They serve different responsibilities.

## Collector architecture

Initial local Collector pipelines:

```text
receivers:
  otlp:
    grpc: 4317
    http: 4318

processors:
  memory_limiter
  optional redaction/filter/resource normalization
  batch

exporters:
  otlp/tempo
  prometheus
  debug (local troubleshooting only)

service.pipelines:
  traces:  [otlp -> memory_limiter -> batch -> otlp/tempo]
  metrics: [otlp -> memory_limiter -> batch -> prometheus]
```

For every configured component, confirm it is enabled in a pipeline. Processor order is operationally significant.

The Collector's own telemetry must also be scraped/observed. Collector health and dropped/export-failed telemetry belong on the observability dashboard.

## Logs pipeline decision

Recommended first local path:

```text
pet-study JSON stdout
  -> Grafana Alloy Docker/file source
  -> Loki
```

Reasons:

- preserves the stable existing `slog` surface;
- avoids making the Go OTel logs SDK a prerequisite;
- separates application failures from log-shipping failures;
- aligns with Loki label/cardinality guidance.

Alternative later path:

```text
slog -> OTel log bridge -> OTLP -> Collector -> Loki native OTLP endpoint
```

This alternative should be evaluated only after traces/metrics and log schema are stable.

## Prometheus topology

Recommended initial topology:

```text
application OTel metrics
  -> Collector OTLP receiver
  -> Collector Prometheus exporter :8889/metrics
  -> Prometheus scrape
```

Prometheus also scrapes:

- Collector internal telemetry (`:8888/metrics` or configured endpoint);
- Prometheus itself;
- Tempo/Loki internal metrics where useful.

Do not expose the application debug `/debug/vars` as the main Prometheus source.

## Grafana provisioning

Repository-owned provisioning should create:

- Prometheus data source;
- Tempo data source;
- Loki data source;
- Tempo trace-to-logs link;
- Loki derived field/log-to-trace link using `trace_id`;
- later trace-to-metrics/exemplar links.

Provisioning files are preferable to manual UI clicks because they are reproducible and reviewable.

## Target repository layout

Suggested layout:

```text
deploy/observability/
  compose.yml
  otel-collector/
    config.yaml
  prometheus/
    prometheus.yml
    rules/
  tempo/
    tempo.yaml
  loki/
    loki.yaml
  alloy/
    config.alloy
  grafana/
    provisioning/
      datasources/
      dashboards/
    dashboards/

docs/
  observability.md
  runbooks/
```

The existing PostgreSQL Compose configuration may be merged into one development stack or referenced as a Compose include/profile, but the result must have one documented entrypoint.

## Network and security boundaries

Local defaults:

- expose Grafana to the host;
- expose Prometheus/Tempo/Loki UIs only when useful for the lab;
- keep Collector OTLP and backend ingestion ports on the Compose network where possible;
- do not expose Collector debug/config endpoints publicly;
- never commit real auth headers, certificates or backend tokens.

Non-local/prod-like mode:

- TLS and authentication between application/Collector/backend;
- bounded exporter queues and retry policy;
- Collector redaction/filtering for sensitive attributes;
- secrets through environment/secret storage, not YAML committed values;
- telemetry failure is fail-open for request serving unless explicitly operating in a diagnostic strict mode.

## Cardinality contract

Allowed bounded dimensions include:

- HTTP method;
- matched route pattern;
- HTTP status/status class;
- gRPC service/method/code;
- upstream service name and normalized route;
- job state;
- normalized error/rejection reason;
- storage backend;
- deployment environment.

Forbidden as metric labels/Loki stream labels:

- request ID;
- trace/span ID;
- user ID;
- job ID;
- raw URL/query;
- email;
- client IP;
- arbitrary error strings;
- JWT claims/tokens.

High-cardinality correlation belongs in traces and structured log metadata, not metrics or indexed Loki labels.

## Lifecycle and shutdown ownership

Telemetry must be created in `run()` after configuration and shut down after application components.

Target order:

```text
1. readiness false
2. stop HTTP/job admissions
3. stop HTTP/HTTPS accepting and wait handlers
4. stop gRPC gracefully
5. stop/finish worker pool according to policy
6. close SSE hub/subscriptions
7. close outbound idle connections and DB
8. force-flush telemetry
9. shut down meter/tracer/log providers
10. return from run; main may call os.Exit for non-zero status
```

Current lifecycle is not yet sufficient because the unexpected HTTP server error path does not explicitly stop all components. Audit 08 and Audit 10 will define the required lifecycle refactor before exporters are introduced.

Telemetry shutdown must be bounded by its own context and must not use an already-cancelled request/server context.

## Readiness policy

The following must **not** make `/readyz` fail:

- Collector unavailable;
- Tempo unavailable;
- Loki unavailable;
- Grafana unavailable;
- temporary exporter backpressure.

Instead expose/monitor:

- export failures;
- dropped spans/metrics/logs;
- exporter queue usage;
- Collector refused/dropped data;
- last successful export where available.

Application readiness remains about the service's ability to serve its business contract.

## Migration policy

### Phase 0 — prerequisites

Before telemetry rollout:

- fix first-write HTTP status recording;
- define logger field schema and one `component` value;
- establish DI-owned metric registry/observers;
- define lifecycle owner and error-path cleanup;
- define async propagation envelope.

### Phase 1 — traces only

- add telemetry bootstrap/resource/propagator;
- OTLP trace exporter and Collector+Tempo;
- HTTP/outbound/gRPC tracing;
- trace IDs in JSON logs;
- deterministic in-memory trace tests.

### Phase 2 — metrics migration

- OTel MeterProvider;
- HTTP/outbound/job/queue/gRPC/DB instruments;
- histograms and bounded attributes;
- Collector Prometheus exporter and Prometheus;
- keep expvar only for transitional debug parity.

### Phase 3 — logs and Grafana correlation

- normalized `slog` JSON;
- Alloy -> Loki;
- provision Grafana data sources;
- trace-to-logs/logs-to-traces.

### Phase 4 — operational hardening

- Collector own telemetry;
- sampling profiles;
- bounded queues/retries;
- TLS/auth for OTLP in prod-like mode;
- failure injection and shutdown flush tests.

### Phase 5 — SLO/dashboards/alerts

Defined in Audit 07.

## Rejected alternatives

### Direct application -> every backend

Rejected because it couples the application to Tempo/Loki/Prometheus details and duplicates batching/retry/security configuration.

### Replace all `slog` calls with direct OTel logs immediately

Rejected for the initial rollout because the existing logging surface is useful and the Go logs signal is less mature. Normalize `slog` first.

### Prometheus scrape of `/debug/vars` as the final solution

Rejected because the current metrics lack histogram semantics, isolated registry ownership and standard labels/types.

### Make Collector availability part of application readiness

Rejected because an observability outage should not create a business-service outage.

### Put `context.Context` directly in queued work

Rejected because accepted async work must outlive the originating HTTP request and future brokers need a serializable carrier.

## New/confirmed gaps

| Capability | Status | Evidence / decision | Risk |
|---|---|---|---|
| Telemetry bootstrap ownership | MISSING | no composition-root telemetry runtime | providers/exporters would be scattered |
| Shared resource identity | MISSING | no service/version/environment resource | signals cannot correlate reliably |
| OTLP export | MISSING | no OTel dependencies/config | no standard pipeline |
| Collector | MISSING | compose contains PostgreSQL only | no central processing/retry/redaction |
| Tempo | MISSING | no backend/config | no trace storage/query |
| Prometheus operational pipeline | MISSING | expvar only | no histograms/SLO queries |
| Loki/log shipping | MISSING | process stderr only | no centralized log search |
| Grafana provisioning | MISSING | no datasource/dashboard files | local stack is not reproducible |
| Telemetry failure policy | MISSING | no config/lifecycle semantics | collector failures may be handled inconsistently |
| Telemetry shutdown/flush | MISSING | no provider/exporter owner | buffered signals would be lost |
| Collector self-observability | MISSING | no Collector | silent telemetry loss risk |
| Signal cardinality contract | MISSING/PARTIAL | good route choices exist, no shared policy | backend cost/performance risk |

## Audit 06 verdict

**Telemetry pipeline architecture: MISSING, but the project has strong insertion points.**

The existing composition root, custom HTTP transport, gRPC runtime, middleware chain, request context and debug diagnostics make incremental instrumentation feasible. The implementation should not begin by adding dashboards. It should begin with lifecycle/registry prerequisites, a single application telemetry runtime, and trace propagation.

## Official reference anchors

Verified during this audit (July 2026):

- OpenTelemetry Go status and instrumentation:
  - https://opentelemetry.io/docs/languages/go/
  - https://opentelemetry.io/docs/languages/go/instrumentation/
  - https://opentelemetry.io/docs/languages/go/exporters/
  - https://opentelemetry.io/docs/languages/go/resources/
  - https://opentelemetry.io/docs/languages/go/sampling/
- OpenTelemetry Collector:
  - https://opentelemetry.io/docs/collector/
  - https://opentelemetry.io/docs/collector/configuration/
  - https://opentelemetry.io/docs/collector/troubleshooting/
  - https://opentelemetry.io/docs/collector/internal-telemetry/
  - https://opentelemetry.io/docs/security/
- HTTP and gRPC instrumentation:
  - https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
  - https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc
  - https://opentelemetry.io/docs/specs/semconv/http/http-spans/
- Prometheus:
  - https://prometheus.io/docs/practices/naming/
  - https://prometheus.io/docs/practices/instrumentation/
  - https://prometheus.io/docs/prometheus/latest/configuration/configuration/
- Grafana stack:
  - https://grafana.com/docs/tempo/latest/set-up-for-tracing/instrument-send/set-up-collector/
  - https://grafana.com/docs/loki/latest/send-data/otel/
  - https://grafana.com/docs/loki/latest/get-started/labels/
  - https://grafana.com/docs/grafana/latest/datasources/tempo/configure-tempo-data-source/configure-trace-to-logs/
