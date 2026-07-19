# Audit 03 — Logging Audit

## Scope

This pass audits the uploaded working tree's application logging only. It reviews logger construction, event coverage, field consistency, correlation, privacy exposure, testability and readiness for future OpenTelemetry correlation. It does not yet implement logging changes or define the final telemetry backend.

## Method and environment

Reviewed production logging and adjacent context/correlation code in:

- `cmd/api/main.go`
- `internal/api/server.go`
- `internal/middleware/middleware.go`
- `internal/middleware/status_recorder.go`
- `internal/httputils/apphandler.go`
- `internal/outbound/profile_instrumentation.go`
- `internal/outbound/profile_retry.go`
- `internal/interceptors/interceptors.go`
- `internal/transport/grpcserver/runtime.go`
- `internal/workerpool/workerpool.go`
- `internal/requestid/requestid.go`
- `internal/queue/queue.go`
- `internal/security/principal.go`
- `internal/security/requestinfo.go`
- relevant logging tests and `server.log`

The repository target remains Go `1.25.8`. A disposable copy was modified to declare Go `1.23.0` with no `toolchain` directive. The original audit tree was not modified. Test execution still could not proceed because required modules are not cached and network access is disabled. Therefore this pass is a static audit.

## Current logging architecture

### Root logger

`cmd/api/main.go:38-41` creates one process-wide logger:

- `slog.NewTextHandler(os.Stderr, ...)`;
- fixed `INFO` minimum level;
- no source location;
- no configurable format or level;
- installed through `slog.SetDefault`.

### Logger usage models

The project currently mixes two models:

1. Global lookup through `slog.Default()`:
   - HTTP access logger;
   - HTTP recovery logger;
   - `AppHandler` unexpected-error logger;
   - API server lifecycle logger;
   - worker-pool logger.

2. Constructor injection of `*slog.Logger`:
   - outbound profile instrumentation;
   - gRPC runtime;
   - gRPC unary interceptor.

This split is functional but produces inconsistent ownership and fields.

## Existing event coverage

| Area | Existing events | Main fields |
|---|---|---|
| Process startup/exit | configured, fatal exit | addr, debug, err |
| HTTP access | one completion log per request | method, raw path, mux pattern, status, bytes, latency, request ID, client IP, scheme |
| HTTP panic | recovered panic | request ID, error, stack |
| Unexpected handler error | mapped unexpected error | method, raw path, request ID, error |
| Outbound profile call | success/failure per physical attempt | request ID, host, stable route, user ID, status, latency, error |
| gRPC unary call | completion | request ID, full method, code, duration |
| HTTP/HTTPS lifecycle | listen/readiness/shutdown timeout/server error | addr, ready, server, timeout, error |
| gRPC lifecycle | listen/failure/forced stop | addr, error |
| Worker pool | transition/storage failures and shutdown repair | job ID, error, count |
| Auth/RBAC/CORS | no dedicated logs | access log and expvar metrics only |
| Queue/bulkhead/rate limiting | no dedicated logs | access log and expvar metrics only |
| SSE | no connection/disconnect/drop logs | expvar counters only |
| DB/repositories | no application logs | readiness and DB expvar stats only |

## Strengths already present

### S1 — Structured logging baseline exists

The application uses `log/slog`; production code contains no `fmt.Printf` or legacy `log.Printf` application logging. Fields are represented as structured key/value attributes.

### S2 — Synchronous HTTP request correlation is present

The request-ID middleware executes before access/error/recovery logging. HTTP access logs, unexpected handler errors and panic logs can include the same request ID used in the response.

### S3 — Access logs include both raw path and normalized mux pattern

`internal/middleware/middleware.go:59-70` logs both `r.URL.Path` and `r.Pattern`. The pattern is read after `ServeMux` execution, providing a stable route identity for future telemetry.

### S4 — Panic logs contain stack traces

`internal/middleware/middleware.go:74-94` logs `debug.Stack()` together with request ID. This addresses an older review finding recorded in `docs/code-review-2026-03-08.md`.

### S5 — Outbound operation identity is normalized

`internal/outbound/profile_instrumentation.go:30-46` uses the stable route `GET /profiles/{user_id}` instead of a concrete URL path for metrics and operation identity.

### S6 — Secrets are not directly logged in reviewed production calls

No reviewed log call emits:

- Authorization headers;
- JWT tokens;
- HMAC secrets;
- database passwords/DSN;
- request or response bodies;
- user email/name/profile content.

This is a good baseline, but there is no enforced redaction policy.

## Findings

### L1 — Async worker logs cannot correlate to the initiating request

Severity: **HIGH**

Evidence:

- `internal/queue/queue.go:21-24` defines `WorkItem` with only `JobID` and payload.
- `internal/workerpool/workerpool.go:114-176` runs all work on the worker-pool lifecycle context and logs only `job_id`.

Lost at the queue boundary:

- request ID;
- future trace/span context;
- enqueue timestamp;
- principal/tenant identity;
- causation metadata.

A client can receive a request ID for the `202 Accepted` response, but worker failures cannot be searched by that request ID. This is the highest-priority logging correlation gap and also blocks complete distributed tracing.

### L2 — No trace ID or span ID correlation exists

Severity: **HIGH**

No OpenTelemetry SDK, span lookup, `trace_id` or `span_id` log attributes were found. Existing request ID correlation is local and does not replace distributed trace context across HTTP, gRPC, outbound HTTP and queue boundaries.

### L3 — Component attribution is duplicated or misleading

Severity: **HIGH for log query reliability**

Evidence:

- `cmd/api/main.go:50` creates `logger := slog.Default().With("component", "main")`.
- This logger is passed to outbound instrumentation at `cmd/api/main.go:124`.
- `internal/outbound/profile_instrumentation.go:50-70` adds another record attribute `component=outbound_profile`.

`slog.TextHandler` preserves both attributes, producing duplicate keys:

```text
component=main component=outbound_profile
```

The same main-scoped logger is passed to gRPC runtime/interceptors (`cmd/api/main.go:141`), so gRPC records are labeled `component=main`. The checked `server.log` confirms `gRPC server listening ... component=main`.

Queries and dashboards cannot reliably group records by component until logger ownership is normalized.

### L4 — Logger ownership is inconsistent and heavily dependent on global state

Severity: **MEDIUM**

HTTP middleware, API server, `AppHandler` and worker pool call `slog.Default()` internally, while outbound/gRPC accept injected loggers. Consequences:

- component fields are applied differently;
- tests that replace `slog.Default()` modify process-global state;
- parallel tests can interfere if global logger mutation expands;
- request-scoped logger enrichment cannot be applied consistently;
- replacing handlers/exporters requires global coordination.

The project should eventually choose one model, preferably constructor injection for lifecycle components and context-derived correlation attributes for request work.

### L5 — Logging format and level are not configurable

Severity: **MEDIUM**

`cmd/api/main.go:38-40` hard-codes:

- text output;
- `INFO` level;
- stderr destination;
- no source field.

There is no logging config section or environment support for development vs production-like behavior. JSON ingestion, temporary debug levels and deterministic timestamp/source policy cannot be selected without code changes.

### L6 — HTTP status in logs and metrics can be incorrect after repeated `WriteHeader`

Severity: **HIGH for telemetry correctness**

`internal/middleware/middleware.go:19-22` assigns `w.status = code` on every `WriteHeader` call before delegating to the underlying writer. `net/http` honors the first status and ignores later calls, but the recorder keeps the last supplied status.

Example:

```text
handler writes 201
handler later writes 500
wire status remains 201
access log and metrics can report 500
```

Because Logger and Metrics share this recorder, the defect affects both observability surfaces. The recorder should preserve the first committed status and track duplicate header attempts separately if desired.

### L7 — Privacy and redaction policy is not defined or enforced

Severity: **MEDIUM**

Current logs include:

- raw `path`, which may contain user/job identifiers;
- `client_ip`;
- outbound `user_id`;
- raw error strings and panic stack traces.

They do not currently include tokens or payloads, which is good. However, no policy defines:

- allowed identifiers;
- hashing/redaction rules;
- retention sensitivity;
- which error types may expose URLs or driver details;
- whether client IP is required outside security diagnostics.

Future Loki/central collection would increase the impact of this gap.

### L8 — Security decisions have counters but no safe audit events

Severity: **MEDIUM**

Authentication, RBAC and CORS failures increment metrics, while the HTTP access log records only final status. It does not capture a safe normalized reason such as:

- `authn_kind=expired`;
- `authz_kind=admin_required`;
- `cors_denial=method`.

This makes incident investigation difficult. Any future security log must avoid tokens and high-cardinality/error-detail fields and should distinguish operational access logs from true audit logs.

### L9 — Retry behavior is not visible as a coherent operation

Severity: **MEDIUM**

The retry wrapper surrounds the instrumented one-shot client:

```text
RetryingProfileClient -> InstrumentedProfileClient -> ClientImpl
```

Therefore each physical attempt is logged, but records do not include:

- attempt number;
- maximum attempts;
- retryability decision;
- selected backoff;
- final outcome after all attempts;
- remaining deadline budget.

A transient failure followed by success appears as independent warning and info records. Retry amplification cannot be reconstructed reliably without counting records heuristically.

### L10 — SSE lifecycle is absent from logs

Severity: **MEDIUM**

The stream hub has counters for subscribers/events/drops, but no logs for:

- connection opened/closed;
- job ID and request ID correlation;
- disconnect reason;
- write deadline/flush failure;
- connection duration;
- subscriber drop threshold events.

Logging every event would be too noisy; connection-level and exceptional-event logging is the appropriate future boundary.

### L11 — Lifecycle logs do not provide a complete shutdown narrative

Severity: **MEDIUM**

Present:

- shutdown started;
- readiness false;
- forced HTTP/gRPC shutdown warnings;
- selected errors.

Missing:

- shutdown trigger/reason;
- per-component shutdown start/end;
- total shutdown duration;
- successful completion;
- queue depth/active jobs/subscribers at shutdown;
- explicit cleanup records on unexpected server failure.

This will matter when OTel exporters require flush/shutdown and when shutdown is tested under load.

### L12 — Field naming and duration units are inconsistent

Severity: **LOW/MEDIUM**

Examples:

- HTTP/outbound: `latency_ms` as integer;
- gRPC: `duration` as a `time.Duration` string;
- metrics: nanosecond sums;
- HTTP uses `pattern`; outbound uses `route`; gRPC uses `method` for full RPC method;
- errors are generally `err`, but operation/result fields are not normalized.

This complicates cross-protocol dashboards and log queries. A common semantic schema should be defined before exporting logs centrally.

### L13 — Access logging can become noisy for probes and long-lived streams

Severity: **LOW/MEDIUM**

The global HTTP logger logs health, debug and SSE requests at `INFO` like ordinary API traffic. Metrics intentionally skip debug/SSE latency, but logging has no equivalent policy. Consequences:

- frequent health probes can dominate logs;
- an SSE access record appears only when the long-lived connection ends;
- debug access can add noise.

A route-aware logging policy is needed, not blanket removal.

### L14 — Unexpected errors may produce two records without an explicit relationship

Severity: **LOW**

An unexpected handler error is logged by `AppHandler` at `ERROR`, then the access logger records completion at `INFO`. The common request ID permits manual correlation, but there is no stable event name/error kind tying the two records together. This is acceptable as a baseline but should be documented and normalized.

## Correlation matrix

| Boundary | request_id | trace context | principal | stable operation name |
|---|---:|---:|---:|---:|
| HTTP middleware/handler | yes | no | yes after AuthN | yes via `r.Pattern` |
| HTTP -> outbound Profile | yes, explicit parameter/header | no | no | yes |
| HTTP -> gRPC bridge | yes via metadata | no | no | yes |
| HTTP -> queue -> worker | no after enqueue | no | no | job ID only |
| Worker -> SSE publish | job ID only | no | no | event type |
| gRPC server | generated/restored request ID | no | no auth principal | full method |

## Required follow-up capabilities

These are verified gaps, not yet implementation tasks:

1. Define one logger ownership/injection model.
2. Define a canonical log field schema for HTTP, gRPC, outbound, worker, SSE and lifecycle events.
3. Fix `statusRecorder` first-status semantics before trusting status-based telemetry.
4. Introduce an async work envelope or persisted job correlation fields.
5. Add trace/span correlation after the tracing design audit.
6. Add safe security/audit event taxonomy.
7. Add privacy/redaction policy and tests.
8. Add retry-operation summary and attempt fields.
9. Add SSE connection lifecycle and exceptional-event logging.
10. Add configurable format/level and production JSON mode.
11. Add lifecycle completion/duration logs and telemetry flush visibility.
12. Add logging contract tests that assert fields, not text formatting only.

## Audit conclusion

Status: **PARTIAL**

The project has a useful structured logging baseline and good synchronous request-ID coverage. It is not yet production-observable because correlation breaks at async boundaries, component identity is unreliable, trace correlation is absent, and one response-recorder defect can make status-based logs incorrect. These issues should be resolved before centralizing logs in Loki/Grafana, otherwise the central platform will preserve misleading or unqueryable data at greater scale.
