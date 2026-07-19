# Audit 05 — Tracing and Context Propagation Audit

## Scope

This pass audits propagation of cancellation, deadlines, request identity, security identity and future distributed-tracing state across the uploaded working tree. It covers inbound HTTP, service/repository calls, outbound HTTP, retries, the HTTP-to-gRPC bridge, gRPC server processing, the bounded queue/worker pool, SSE delivery, startup and shutdown contexts, and related tests.

This pass does not add OpenTelemetry or change runtime behavior. Recommendations below describe the target propagation model; they are not implemented yet.

## Method and environment

Reviewed in depth:

- `cmd/api/main.go`
- `internal/requestid/requestid.go`
- `internal/middleware/trustproxy_requestid.go`
- `internal/middleware/trustproxy.go`
- `internal/security/principal.go`
- `internal/security/requestinfo.go`
- `internal/routes/users_handler.go`
- `internal/routes/users_handler_v2.go`
- `internal/routes/user_profile_handler.go`
- `internal/routes/job_handler.go`
- `internal/service/service.go`
- `internal/service/jobService.go`
- `internal/service/user_profile_service.go`
- `internal/outbound/profile.go`
- `internal/outbound/profile_retry.go`
- `internal/outbound/profile_instrumentation.go`
- `internal/transport/grpcclient/grpcclient.go`
- `internal/interceptors/interceptors.go`
- `internal/transport/grpcserver/runtime.go`
- `internal/transport/grpcserver/grpc_job_service.go`
- `internal/queue/queue.go`
- `internal/workerpool/workerpool.go`
- `internal/stream/stream.go`
- SQLX and memory repositories
- propagation/cancellation-related tests.

The target remains Go `1.25.8`. Full dynamic tests are unavailable because external modules are not cached and the sandbox has no network. Findings in this pass are based on static control-flow and data-flow inspection plus the already established stdlib-only compatibility checks.

## Current propagation matrix

| Boundary | Cancellation/deadline | Request ID | Principal/request info | Trace context | Current result |
|---|---|---|---|---|---|
| inbound HTTP -> global middleware | inherited from `net/http` request | generated or trusted-proxy value, stored in context | trusted proxy info stored; principal added later by auth | none | synchronous baseline is sound |
| HTTP handler -> service -> repository | same request context | available in context, but generally not consumed below handler | principal remains in context | none | preserved |
| profile service -> outbound HTTP | parent context plus profile timeout | passed separately as string and written to header | not propagated | none | cancellation works; correlation is manual |
| retry wrapper -> physical attempts | same budget context for all attempts | same explicit string | not propagated | none | deadline/cancel aware |
| HTTP -> local gRPC bridge | original request context | copied to `request-id` metadata | not copied | none | cancellation/deadline works; identity is partial |
| gRPC server interceptor -> service/repository | gRPC request context | extracted/generated and stored in context | no auth principal | none | request ID works for server logs only |
| HTTP async create -> queue | request context controls only the enqueue attempt | not placed in `WorkItem` | not placed in `WorkItem` | none | correlation ends at enqueue |
| queue -> worker | shared worker-pool lifecycle context | absent | absent | absent | all jobs share app lifecycle context |
| worker -> SSE hub event | no context argument | absent | absent | absent | event has job ID/time/data only |
| SSE connection | request context ends on client disconnect | request ID remains in handler context | principal used for initial authorization | none | connection cancellation is handled |
| shutdown cleanup | new background timeout contexts | no operation correlation | none | none | cancellation-independent cleanup, no telemetry flush context yet |

## Strengths already present

### T-S1 — Synchronous HTTP context is consistently threaded through the application

Handlers pass `r.Context()` into services. Services pass it into repositories. SQLX repositories derive query timeouts from the parent context rather than replacing it. This preserves client cancellation and upstream deadlines for normal synchronous operations.

### T-S2 — Outbound HTTP requests use `http.NewRequestWithContext`

`internal/outbound/profile.go:33` binds the outgoing request to the supplied context. A canceled HTTP request or expired profile budget therefore reaches the transport.

### T-S3 — Retry backoff respects one shared operation budget

`internal/outbound/profile_retry.go` checks `ctx.Err()`, verifies time remaining before sleeping, and uses a context-aware timer. Retries do not create independent unlimited contexts.

### T-S4 — The profile timeout is derived from the request context

`internal/service/user_profile_service.go:34-36` uses `context.WithTimeout(ctx, profileTimeout)`. It narrows the budget while preserving parent cancellation and future trace state.

### T-S5 — HTTP-to-gRPC calls retain cancellation and deadlines

`internal/routes/job_handler.go:63-75` calls the gRPC client with a context derived from `r.Context()`. gRPC can therefore observe client disconnect and deadline cancellation.

### T-S6 — gRPC handlers pass their context into the service/repository layer

`JobGRPCService.GetJob` calls `jobService.GetByID(ctx, ...)`, so server-side cancellation/deadline can reach PostgreSQL.

### T-S7 — SSE exits on request cancellation

`JobHandler.Events` selects on `r.Context().Done()`. The subscription is removed by a deferred idempotent unsubscribe, and closing the hub closes subscriber channels.

### T-S8 — HTTP request-ID trust is explicitly bounded

The outer proxy middleware removes client-supplied `X-Request-Id` unless the remote address is a configured trusted proxy. The request-ID middleware then sanitizes the retained value or creates a new one.

## Findings

### T1 — Distributed tracing is not implemented

Severity: **HIGH / expected primary gap**

No application code imports or configures:

- an OpenTelemetry tracer provider;
- W3C Trace Context extraction/injection;
- HTTP server/client instrumentation;
- gRPC server/client tracing interceptors;
- a batch span processor or exporter;
- service/resource attributes;
- sampling configuration;
- baggage propagation.

OpenTelemetry modules appear in `go.sum` as transitive checksums, but they are not direct application dependencies and no trace API is used.

Consequences:

- no `trace_id` or `span_id` for logs;
- no end-to-end HTTP -> DB/outbound/gRPC trace;
- no async queue/worker causality;
- no visualization in Tempo or another backend;
- no trace exemplars for future metrics.

### T2 — Async enqueue is a hard propagation break

Severity: **HIGH**

`queue.WorkItem` contains only:

```go
JobID   int64
Payload entity.CreateUserInput
```

The HTTP request context is used only while trying to send the item into the channel. Once `Enqueue` succeeds, the worker has no access to:

- request ID;
- remote trace parent/state;
- principal/tenant identity;
- enqueue time;
- causation/correlation fields;
- operation-specific deadline policy.

This is the most important propagation defect for the planned observability work.

The correct remediation is **not** to store `context.Context` in the queue item. The target should be an explicit, bounded, serializable work envelope, for example conceptually:

```text
WorkItem
  JobID
  Payload
  Correlation.RequestID
  Correlation.TraceCarrier
  Correlation.ActorID/Role if policy permits
  EnqueuedAt
```

That envelope can later be carried through Kafka/NATS/RabbitMQ headers without redesigning the application model.

### T3 — Workers use one shared lifecycle context rather than a per-job operation context

Severity: **HIGH**

`WorkerPool.Start` creates one `wp.ctx` from the application signal context. Every worker and every job uses that same context for:

- `SetRunning`;
- user creation;
- `SetSucceeded`;
- most failure handling.

This is appropriate as a lifecycle cancellation source, but insufficient as an operation context. There is no per-job span, per-job timeout, correlation metadata, or cancellation cause.

The target design should derive a fresh per-work context from the pool lifecycle context, then extract/link the serialized trace carrier from the `WorkItem`. Job execution should remain independent of the originating HTTP client's cancellation after `202 Accepted`, while still preserving causality.

### T4 — `markJobFailed(context.Background())` severs all operation context

Severity: **MEDIUM/HIGH**

`internal/workerpool/workerpool.go:194-206` uses `context.Background()` to write the terminal failed state after worker cancellation.

The intention is valid: terminal repair must not be immediately canceled by the pool lifecycle context. However, this also removes:

- request/job correlation values;
- future span context;
- cancellation cause;
- any outer cleanup deadline.

PostgreSQL repositories currently add their own query timeout, so the DB operation is bounded in that implementation. The service-level contract is still unbounded and the memory implementation ignores context.

The future cleanup context should be cancellation-independent but bounded and should preserve safe correlation values. A dedicated shutdown/repair context is preferable to an unqualified background context.

### T5 — Job persistence has no correlation or propagation metadata

Severity: **HIGH for restart/broker evolution**

`entity.Job` and both job repositories store status, owner, result and problem only. They do not store request ID, trace carrier, enqueue time or producer identity.

Even if an in-memory `WorkItem` is extended, correlation would still disappear when:

- the process restarts;
- queued work is moved to a durable broker;
- a job is replayed/requeued;
- another service consumes the job.

A later design decision must explicitly separate:

- operational trace propagation fields, which may be stored with the message/job envelope;
- business data, which should not be polluted with arbitrary context values;
- privacy-sensitive actor fields, which need a retention policy.

### T6 — Outbound correlation is duplicated between context and an explicit string parameter

Severity: **MEDIUM**

The HTTP request ID already exists in `context.Context`, but `FetchProfile` also requires `requestID string`. The handler extracts the value and passes it through service/client layers separately.

This allows divergence: a caller can provide a context containing one request ID and a string containing another. Logs and outbound headers use the explicit string, while future tracing would use the context.

Before adding OTel, choose one canonical model:

- read request ID from context in infrastructure code; or
- pass a typed request metadata object containing explicit fields.

Avoid parallel implicit and explicit sources of truth.

### T7 — Outbound HTTP propagates request ID but not standard trace context

Severity: **HIGH**

`ClientImpl` sets `X-Request-Id` manually. There is no instrumented `RoundTripper` and no injection of `traceparent`/`tracestate`.

Cancellation and deadline propagation are correct, but a future upstream trace cannot be connected to the inbound server operation.

Instrumentation should be attached to the shared transport/client once in the composition root, not recreated per request or retry.

### T8 — Retry attempts have no explicit span model

Severity: **MEDIUM**

Each retry uses the same context, which is a good budget model. There is no decision yet whether traces should contain:

- one logical outbound span with retry events; or
- one logical operation span plus a child span for each physical attempt.

Without an explicit model, automatic HTTP client instrumentation may produce attempt-level spans while application metrics/logs describe a different concept. Audit 06 must align trace, metric and log semantics for logical operations versus physical attempts.

### T9 — The HTTP-to-gRPC bridge propagates only request ID

Severity: **HIGH**

`GetByIDViaGRPC` places `request-id` into outgoing metadata and calls the client with the HTTP request context. It does not propagate:

- trace context;
- principal/authentication metadata;
- tenant/baggage;
- explicit caller/service identity.

The direct gRPC service therefore cannot apply the same security or correlation model as the HTTP layer.

The security consequence will be handled in Audit 12. For observability, both gRPC client and server instrumentation are required.

### T10 — `metadata.NewOutgoingContext` replaces existing outgoing metadata

Severity: **MEDIUM / future compatibility risk**

The bridge uses:

```go
metadata.NewOutgoingContext(ctx, metadata.Pairs("request-id", reqID))
```

This creates a context with the supplied outgoing metadata rather than intentionally appending to an existing carrier. The current HTTP context normally has no gRPC metadata, so it works today. It is fragile once auth, baggage or other middleware starts adding metadata.

A composition-safe propagation helper should preserve existing metadata and have tests for coexistence with trace/auth fields.

### T11 — gRPC request-ID metadata is trusted without sanitization

Severity: **HIGH for correlation integrity**

The unary server interceptor accepts the first incoming `request-id` value directly. Unlike HTTP, it does not:

- apply length/character validation;
- distinguish trusted internal callers;
- normalize multiple values;
- enforce the HTTP trusted-proxy policy equivalent.

The listener may bind to a non-loopback configured address, and it currently uses insecure transport. A direct gRPC caller can therefore influence log correlation identifiers.

This does not expose a secret by itself, but it makes correlation and log queries untrustworthy. A shared request-ID validation function and an explicit gRPC trust policy are needed.

### T12 — A generated gRPC request ID is not returned to the client

Severity: **MEDIUM**

When incoming metadata has no request ID, the interceptor generates one and stores it in the server context. It logs the value, but does not send it as gRPC response header/trailer metadata.

A direct caller cannot know which correlation ID belongs to its call. The HTTP bridge avoids this only because HTTP already generated an ID before invoking gRPC.

### T13 — There is no gRPC client interceptor and no stream interceptor baseline

Severity: **HIGH for future protocol growth**

The client connection is created without interceptors. The server installs only one unary interceptor for request ID/logging.

Current proto exposes one unary method, so stream interceptors are not exercised today. The project roadmap explicitly targets richer gRPC and broker/high-load experiments; client/server unary and stream propagation should be treated as one instrumentation package rather than added ad hoc later.

### T14 — Security identity remains request-local and stops at integration boundaries

Severity: **MEDIUM**

`Principal` and trusted `RequestInfo` are stored in the inbound HTTP context. They reach synchronous application code, but are not transferred into:

- gRPC metadata;
- queue work envelopes;
- worker operation contexts;
- audit events.

Not all identity should be propagated everywhere. The missing piece is an explicit policy defining which bounded actor fields are required for authorization, audit and trace attributes, and which fields must never become baggage/metric labels.

### T15 — SSE connection cancellation is correct, but event causality is absent

Severity: **MEDIUM**

The SSE handler is tied to the client request context and cleanly exits on disconnect. The event hub itself accepts only `jobID` and `Event`, with no context or correlation carrier.

Consequences:

- a `queued/running/succeeded/failed` event cannot be linked to the producer/worker trace;
- terminal events cannot carry a stable causation reference;
- SSE logs cannot correlate connection, job transition and originating create request.

The target should avoid creating a child span for every heartbeat or holding unbounded per-event trace state. Audit 06 should define a bounded connection/event model, likely using connection-level telemetry plus job/trace correlation on meaningful state transitions.

### T16 — Route-aware span naming must preserve the existing low-cardinality invariant

Severity: **HIGH design requirement**

Current metrics obtain `r.Pattern` only after `ServeMux` has matched and handled the request. A naïve HTTP tracing wrapper may start/name a server span before the route pattern is known and fall back to the raw path.

The tracing implementation must update the final route attribute/span name from `r.Pattern` without bypassing `ServeMux.ServeHTTP`. Concrete user/job IDs must not become span names or unbounded attributes.

### T17 — Startup and shutdown contexts are separated from request cancellation, but telemetry lifecycle is absent

Severity: **MEDIUM now / HIGH once exporters exist**

Startup uses the root signal context for the worker pool and server lifecycle. Shutdown creates background timeout contexts for HTTP, gRPC and worker cleanup, which is generally the correct direction because the signal context is already canceled.

Missing:

- a telemetry provider owned by the composition root;
- a guaranteed flush/shutdown call on every exit path;
- a dedicated timeout/config for telemetry flush;
- a clear order between stopping admissions, servers, workers and exporters;
- correlation/cause fields on shutdown operations.

The asymmetric unexpected-server-error path found in Audit 02 would also risk losing buffered spans.

### T18 — DB/startup operations are only partly connected to application cancellation

Severity: **LOW/MEDIUM**

Normal repository calls derive from request/gRPC/worker contexts and are well structured. `db.Open` performs its startup ping from `context.Background()` with a timeout, so an application signal cannot cancel the startup ping early. This is bounded but not connected to the root application context.

This is not a production blocker, but the future composition root should pass an explicit startup context and trace startup phases separately from request traces.

### T19 — Propagation tests cover request ID and some cancellation, not distributed context

Severity: **HIGH for the future implementation**

Existing useful tests include:

- request ID generation/sanitization in HTTP;
- request ID propagation to the profile upstream;
- outbound timeout and upstream cancellation behavior;
- gRPC request-ID logging;
- SSE response/cancellation scenarios.

Missing test contracts:

- incoming `traceparent` is extracted and continued;
- outbound HTTP injects trace context;
- gRPC client/server propagation preserves the same trace;
- queue envelope preserves/links causality;
- worker spans have job ID but no high-cardinality metric labels;
- generated gRPC request ID is returned;
- malformed gRPC request ID is rejected/replaced;
- cancellation/deadline status is recorded consistently in spans;
- telemetry shutdown flushes buffered data;
- SSE disconnect and write-timeout spans/events terminate without leaks.

### T20 — There is no documented propagation and privacy contract

Severity: **MEDIUM**

The project has implicit conventions but no document defining:

- request ID versus trace ID responsibilities;
- accepted external trace headers;
- baggage allowlist;
- actor/tenant propagation rules;
- async parent versus span-link policy;
- retry span policy;
- attributes prohibited because of PII or cardinality;
- behavior across trust boundaries.

This contract should be written before instrumentation is spread across packages.

## Target trace topology for Audit 06 design

The following is a proposed topology to validate in the telemetry architecture pass.

### Synchronous profile request

```text
HTTP server span: GET /api/v1/users/{id}/profile
  service span or event: load user
    DB client span(s)
  logical outbound profile operation
    physical HTTP attempt 1
    retry/backoff event
    physical HTTP attempt 2
```

### HTTP-to-gRPC bridge

```text
HTTP server span: GET /api/v1/jobs/{id}/grpc
  gRPC client span: /pb.JobsService/GetJob
    gRPC server span: /pb.JobsService/GetJob
      DB/repository span
```

### Async create

```text
HTTP server span: POST /api/v1/users?async=1
  producer/enqueue span

worker consumer/process span
  link or extracted relationship to producer context
  job transition events
  DB spans
```

The consumer should not depend on the HTTP request cancellation after the service has returned `202 Accepted`.

### SSE

```text
HTTP server/stream connection telemetry
  meaningful events: subscribed, terminal job event, disconnect/write failure
```

Heartbeat writes should not create high-volume child spans.

## Required design decisions for Audit 06

1. OpenTelemetry SDK/exporter ownership in `cmd/api/main.go`.
2. Resource attributes: service name/version/environment/instance.
3. W3C propagator and bounded baggage policy.
4. HTTP server instrumentation placement while preserving `r.Pattern`.
5. Shared instrumented outbound `RoundTripper`.
6. gRPC client/server unary and stream interceptors.
7. Serializable async propagation envelope.
8. Producer/consumer parent-versus-link policy.
9. Logical operation versus physical retry span model.
10. SSE connection/event telemetry policy.
11. Log correlation helper for request ID, trace ID and span ID.
12. Shutdown/flush order and timeout.
13. In-memory exporters/readers for deterministic tests.

## Audit result

Status: **PARTIAL synchronous propagation / MISSING distributed and async propagation**.

The synchronous request paths already provide a strong context-cancellation foundation. The critical work is not to rewrite that foundation, but to add one coherent propagation system around it. The queue/worker boundary, gRPC boundary, route-aware span naming and telemetry lifecycle must be designed together; implementing isolated `trace_id` fields in logs would not solve the underlying causality gaps.
