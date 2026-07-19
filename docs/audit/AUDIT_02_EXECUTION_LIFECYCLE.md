# Audit 02 — Execution and Lifecycle Map

## Scope

This pass maps how the uploaded working tree starts, handles work, crosses asynchronous/protocol boundaries, and shuts down. It records lifecycle and propagation facts needed for later logging, metrics and tracing audits.

It is not yet a full concurrency, security or correctness audit.

## Component ownership map

### `cmd/api/main.go`

Creates and wires:

- root signal context;
- config;
- trusted proxy middleware;
- metrics singletons;
- stream hub;
- memory or PostgreSQL repositories;
- services;
- queue and worker pool;
- outbound HTTP client, instrumentation and retries;
- optional gRPC runtime and loopback client;
- readiness checks;
- JWT/auth/RBAC/CORS/security middleware;
- HTTP routers and global middleware chain;
- `APIServer`.

Important ownership fact: several runtime components are started in `main` before `APIServer.Run` owns the HTTP lifecycle:

- worker pool is started at `cmd/api/main.go:112-117`;
- gRPC runtime is created and started at `cmd/api/main.go:140-163`;
- HTTP/HTTPS listeners are opened later inside `APIServer.Run`.

### `internal/api/APIServer`

Owns:

- HTTP server;
- optional HTTPS server;
- readiness lifecycle flag;
- queue stop-accepting transition;
- stream hub close on signal shutdown;
- gRPC shutdown on signal shutdown;
- deferred worker-pool stop after the main Run path returns.

## Startup sequence

Observed startup order:

```text
main
  -> create signal root context
  -> load config
  -> create metrics and stream hub
  -> open memory/PostgreSQL repositories
       -> PostgreSQL mode performs startup Ping
  -> create services
  -> create queue
  -> start worker pool using root context
  -> create outbound HTTP client
  -> optionally bind/start gRPC runtime
  -> create readiness checks
  -> create handlers/routers/middleware
  -> APIServer.Run(root context)
       -> bind HTTP listener
       -> optionally load TLS keypair and bind HTTPS listener
       -> start HTTP/HTTPS Serve goroutines
       -> set readiness=true
       -> wait for root cancellation or server error
```

### Startup propagation behavior

- A gRPC `Serve` failure calls the root cancel function (`grpcserver/runtime.go:61-64`).
- HTTP/HTTPS listener binding failure returns directly from `APIServer.Run`.
- PostgreSQL is a startup dependency when `STORAGE_BACKEND=postgres`; `db.Open` performs `PingContext` before handlers are created.

## HTTP request execution map

### Global chain

Construction in `cmd/api/main.go` produces this execution order:

```text
Proxy.SanitizeRequestIDHeader
  -> RequestIDMiddleware
    -> Recover (outer)
      -> TrustProxy
        -> Metrics
          -> Logger
            -> Recover (inner)
              -> Root ServeMux
```

Effects:

- untrusted request-id is removed before request-id generation;
- request-id exists for recovery, logs and Problem responses;
- trusted proxy information exists before access logging;
- metrics and logger share a `statusRecorder`;
- `r.Pattern` is read after the nested mux has executed;
- the inner recover allows logger/metrics to observe a generated 500;
- the outer recover protects middleware outside the inner boundary.

### Root mux

```text
Root ServeMux
  /api...  -> API subtree
  /livez   -> health mux
  /readyz  -> health mux
  /debug/* -> optional authenticated debug mux
```

### API subtree

Outer wrappers configured in `main`:

```text
SecurityHeaders
  -> CORS
    -> API ServeMux
```

Per-route wrapper configured in `internal/router/router.go`:

```text
Bulkhead
  -> RateLimiter
    -> Authenticate
      -> RBAC
        -> AppHandler
```

CORS can short-circuit preflight before API authentication.

### AppHandler error path

```text
handler returns error
  -> MapError
  -> apply mapped headers
  -> log only when mapping marks error as unexpected
  -> WriteProblem
       -> request_id from context
       -> response request-id header
       -> application/problem+json body
```

## Synchronous user request flow

Example `POST /api/v1/users` in synchronous mode:

```text
HTTP request context
  -> middleware/auth/RBAC
  -> strict body/content validation
  -> UserService.CreateUser(ctx)
  -> UserRepository.ExistsByEmail(ctx)
  -> UserRepository.Save(ctx)
  -> 201 JSON response
```

The request context is propagated into service and repository calls.

## Outbound profile flow

```text
GET /api/v1/users/{id}/profile
  -> request context + request-id
  -> UserProfileService
       -> load user with request context
       -> child context.WithTimeout(request context)
       -> RetryingProfileClient
       -> InstrumentedProfileClient
       -> ClientImpl/http.Client
            -> NewRequestWithContext
            -> X-Request-Id propagated
```

Properties:

- cancellation/deadline propagate from HTTP request;
- retry sleep respects context;
- request-id is carried explicitly as a method parameter and outbound header;
- outbound metrics/logs wrap each one-shot client attempt beneath the retry wrapper.

## Async job flow

### Producer path

```text
POST /api/v1|v2/users?async=1
  -> authenticate principal
  -> create queued Job in repository
  -> Queue.Enqueue(request context, WorkItem{JobID, Payload})
  -> increment queued metric
  -> publish queued SSE event
  -> 202 + Location
```

If enqueue fails, the handler attempts to delete the previously persisted Job.

### Queue boundary

`queue.WorkItem` currently contains only:

- `JobID`;
- user payload.

It does not carry:

- request-id;
- trace context;
- principal/tenant identity;
- enqueue timestamp;
- causation/correlation metadata.

The worker therefore starts from the application lifecycle context, not the originating request context.

### Consumer path

```text
WorkerPool worker
  -> select(worker lifecycle ctx.Done, queue receive)
  -> JobService.SetRunning(worker ctx)
  -> metrics running + SSE running
  -> UserService.CreateUser(worker ctx)
  -> JobService.SetSucceeded/SetFailed
  -> metrics terminal state + processing latency
  -> SSE terminal event
```

For terminal failure persistence, `markJobFailed` deliberately uses `context.Background()` rather than the canceled worker context.

## SSE flow

```text
GET /api/v1/jobs/{id}/events
  -> normal API auth/RBAC
  -> resource-level owner/admin check
  -> set text/event-stream headers
  -> subscribe to Hub(jobID)
  -> loop:
       request context cancellation -> return
       heartbeat ticker -> write + flush with write deadline
       event channel -> write + flush with write deadline
       closed subscription -> return
```

Hub behavior:

- subscriptions use bounded channels;
- publish is non-blocking per subscriber;
- full subscriber buffers increment drop metrics;
- hub close atomically rejects new work and closes all subscriber channels;
- ordinary HTTP latency metrics explicitly skip the SSE route.

## HTTP-to-gRPC bridge flow

```text
GET /api/v1/jobs/{id}/grpc
  -> API auth/RBAC
  -> extract request-id from HTTP context
  -> outgoing gRPC metadata: request-id
  -> loopback JobsService.GetJob
  -> unary interceptor restores/generates request-id
  -> JobService.GetByID(ctx)
  -> gRPC status mapping
  -> HTTP DTO / Problem mapping
```

Current propagation:

- request-id: yes;
- HTTP request cancellation/deadline: yes, via the gRPC call context;
- W3C trace context: no instrumentation detected;
- authenticated principal/authorization context: not propagated to gRPC.

The gRPC listener itself is created with insecure transport credentials and has no authentication interceptor in the current working tree.

## Health and readiness flow

- `/livez` always returns process liveness JSON.
- `/readyz` first checks the atomic lifecycle flag.
- Then checks run sequentially under one shared 200 ms request-derived timeout:
  - repository/DB;
  - worker pool;
  - stream hub;
  - gRPC runtime when enabled.
- Individual error strings are returned in readiness details.

## Shutdown sequence on signal context cancellation

Root signal cancellation also immediately cancels the worker pool child context.

Observed `APIServer.Run` sequence:

```text
root ctx canceled
  -> readiness=false
  -> queue.StopAccepting
  -> stream Hub.Close
  -> HTTPS Shutdown (if enabled)
  -> HTTP Shutdown
  -> gRPC GracefulStop with timeout fallback Stop
  -> wait HTTP/HTTPS Serve goroutines
  -> return from Run
  -> deferred WorkerPool.Stop
       -> cancel again (idempotent lifecycle)
       -> wait workers
       -> mark queued/running jobs failed
```

Other `run()` defers then close:

- gRPC loopback client connection;
- outbound idle connections;
- DB connection.

## Server-error path

When an HTTP/HTTPS Serve goroutine sends an unexpected error:

```text
readiness=false
  -> queue.StopAccepting
  -> close HTTP/HTTPS servers
  -> wait HTTP/HTTPS goroutines
  -> return error
  -> deferred WorkerPool.Stop
```

In this branch, the code does **not** explicitly:

- close the stream hub;
- shut down the gRPC server runtime.

The process normally exits afterward from `main`, but lifecycle cleanup is not symmetric with signal shutdown.

## Evidence-based lifecycle findings

### E1 — Startup ownership is fragmented

Severity: **HIGH**

Worker pool and gRPC are started before HTTP/HTTPS listener binding, while their cleanup is mostly owned by `APIServer.Run` after later points in startup.

If `APIServer.Run` returns during early listener/TLS setup, its worker-pool defer has not yet been installed, and stream/gRPC cleanup is not run through the normal shutdown sequence.

The top-level process exit masks this in normal CLI execution, but it makes lifecycle behavior hard to test, reuse and instrument.

### E2 — Shutdown paths are asymmetric

Severity: **HIGH**

Signal shutdown closes the stream hub and gRPC runtime. Unexpected HTTP/HTTPS server failure does not.

This should be unified before adding telemetry exporters, because exporters will introduce another lifecycle component requiring flush/shutdown on every exit path.

### E3 — Async work loses request/trace correlation at enqueue

Severity: **HIGH for observability**

`WorkItem` does not carry request-id or propagation metadata. Logs and future spans produced by workers cannot be causally linked to the original HTTP request without changing the message envelope or persisting correlation data with the Job.

### E4 — gRPC bridge propagation is request-id only

Severity: **MEDIUM**

The bridge forwards request-id and cancellation but not trace context or principal identity. Direct access to the gRPC listener does not share the HTTP JWT/RBAC boundary.

Security decisions for internal gRPC will be evaluated in Audit 12; trace propagation in Audit 5.

### E5 — Worker terminal write uses unbounded background context

Severity: **MEDIUM**

`WorkerPool.markJobFailed` writes terminal state using `context.Background()` with no explicit timeout. This avoids losing the terminal update after lifecycle cancellation, but a PostgreSQL call can outlive the intended shutdown budget until repository-level query timeout intervenes.

The effective bound currently depends on the repository implementation, not the worker method contract.

### E6 — Observability route identity is intentionally mux-derived

Status: **GOOD BASELINE**

Logger and HTTP metrics read `r.Pattern` after `ServeMux` execution, avoiding raw user IDs in metric keys. This is a strong foundation for later Prometheus/OTel route attributes.

### E7 — Streaming has a separate telemetry path

Status: **GOOD BASELINE / PARTIAL**

Long-lived SSE is excluded from ordinary latency metrics and has subscriber/event/drop counters. It still lacks connection duration, disconnect reason and trace strategy.

### E8 — Request context propagation is good on synchronous boundaries

Status: **GOOD BASELINE**

HTTP -> service -> repository, outbound HTTP and HTTP -> gRPC calls use request-derived contexts. The main context break is the intentional async queue boundary.

### E9 — Readiness has one shared sequential budget

Severity: **LOW / design constraint**

Checks execute sequentially under 200 ms. A slow earlier dependency can prevent later checks from being evaluated. This is deterministic and bounded, but dashboard/diagnostic semantics should account for it.

## Execution paths required for observability audits

Audit 3–8 must cover these distinct paths:

1. HTTP success and Problem error.
2. HTTP panic recovery.
3. HTTP -> PostgreSQL.
4. HTTP -> outbound Profile with retries.
5. HTTP -> local gRPC bridge.
6. Async enqueue -> worker -> PostgreSQL/user creation.
7. SSE subscription and dropped events.
8. Readiness checks.
9. Signal shutdown.
10. Unexpected HTTP/HTTPS failure.
11. gRPC runtime failure triggering root cancellation.

## Audit 02 conclusion

The project has clear request context propagation for synchronous I/O and a deliberate bounded streaming/queue design. The largest cross-cutting gap is lifecycle and correlation ownership: asynchronous jobs lose request causality, and startup/shutdown cleanup is spread across components with non-equivalent exit paths.

These findings directly shape the next pass: logging must be audited per execution path, not merely by counting `slog` calls.
