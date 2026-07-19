# Audit 08 — Telemetry Reliability and Shutdown

## Scope

This pass evaluates whether the uploaded `pet-study` working tree can reliably preserve operational state and future telemetry during normal shutdown, startup failure, listener failure, gRPC failure, worker cancellation and observability-backend outage.

The pass does **not** add OpenTelemetry or change application code. It defines lifecycle prerequisites that must be satisfied before OTLP exporters, Collector, Prometheus, Tempo, Loki and Grafana are introduced.

## Reviewed evidence

Primary project evidence:

- `cmd/api/main.go`
- `internal/api/server.go`
- `internal/api/server_test.go`
- `internal/workerpool/workerpool.go`
- `internal/transport/grpcserver/runtime.go`
- `internal/transport/grpcclient/grpcclient.go`
- `internal/routes/job_handler.go`
- `internal/stream/stream.go`
- `internal/queue/queue.go`
- `internal/health/health.go`
- `internal/health/router.go`
- `internal/db/open.go`
- `internal/config/config.go`
- Audit 02, 05, 06 and 07 reports.

External design anchors:

- Go `net/http.Server.Shutdown` and `Close` lifecycle semantics.
- grpc-go `GracefulStop` and forced `Stop` behavior.
- OpenTelemetry SDK `ForceFlush` and `Shutdown` contracts.
- OpenTelemetry Collector resiliency, sending queues, retry and memory-limiter guidance.

## Executive result

### Current state

The normal signal-driven path has several good properties:

- readiness is cleared before shutdown;
- queue admission is stopped;
- SSE subscriptions are actively closed so long-lived handlers can exit;
- HTTP and HTTPS use graceful shutdown with a forced-close fallback;
- gRPC uses graceful stop with forced stop fallback;
- worker pool cancellation is attempted on every return from `APIServer.Run`;
- database and outbound idle connections are closed by `run` defers.

However, lifecycle ownership is split across `main`, `run`, `APIServer`, `grpcserver.Runtime`, `WorkerPool` and package-level defers. There is no single supervisor, no single application shutdown deadline, no component outcome model and no telemetry runtime.

This creates failure paths where:

1. startup components begin before construction is complete;
2. a later construction failure does not synchronously stop and wait for every started component;
3. an unexpected HTTP listener failure does not explicitly close the event hub or stop gRPC;
4. a gRPC serve failure is converted into root-context cancellation and may result in process exit code 0;
5. forced HTTP/gRPC shutdown is logged but treated as a clean return;
6. per-component timeouts can accumulate beyond the external termination grace period;
7. worker repair and worker waiting share one timeout budget;
8. future telemetry flush has no guaranteed final position in the shutdown order.

### Status

**Telemetry reliability and unified shutdown: PARTIAL/BROKEN.**

The service has a useful graceful-shutdown baseline, but it is not yet safe to attach buffered telemetry exporters and claim reliable final export.

## Positive findings

### 1. Shutdown begins with service withdrawal

`APIServer.Run` sets readiness false and calls `queue.StopAccepting()` before shutting down protocol servers (`internal/api/server.go:135-140`). This is the correct high-level intent: stop advertising readiness and stop accepting new asynchronous work before destroying dependencies.

### 2. SSE has an explicit close mechanism

`Hub.Close` is idempotent, marks the hub closed, closes subscriber channels under the hub mutex and resets the subscriber gauge (`internal/stream/stream.go:194-219`). Closing subscriber channels allows the SSE handler select loop to exit instead of waiting indefinitely for a request context (`internal/routes/job_handler.go:148-151`).

This is important because long-lived streaming handlers can otherwise delay `http.Server.Shutdown`.

### 3. HTTP has a bounded forced-close fallback

`shutdownServer` uses a timeout context and invokes `Server.Close` after a graceful-shutdown deadline (`internal/api/server.go:194-221`). The existence of a bounded fallback is necessary for stuck handlers.

### 4. gRPC has graceful and forced stop paths

`grpcserver.Runtime.Shutdown` starts `GracefulStop` and calls `Stop` after the supplied context expires (`internal/transport/grpcserver/runtime.go:68-89`). This prevents relying on unbounded graceful completion alone.

### 5. Worker stop is idempotent at the running-state boundary

`WorkerPool.Stop` handles repeated calls and attempts terminal repair even when the pool is already marked stopped (`internal/workerpool/workerpool.go:82-106`). Existing tests cover queued and running jobs becoming failed during normal shutdown.

### 6. Observability is not currently part of readiness

No Collector, exporter or Grafana dependency participates in `/readyz`. This should remain true: telemetry failure must be observable but must not automatically become application unavailability.

## Findings

## F1 — There is no unified application supervisor

Severity: **HIGH**

Lifecycle ownership is distributed:

- signal context: `cmd/api/main.go:51-52`;
- DB open/close: `cmd/api/main.go:85-100`;
- worker start: `cmd/api/main.go:111-117`;
- outbound transport defer: `cmd/api/main.go:119-121`;
- gRPC start/client connection: `cmd/api/main.go:140-162`;
- HTTP/HTTPS/hub/gRPC/pool shutdown: `internal/api/server.go`;
- process exit: `cmd/api/main.go:43-45`.

There is no object that owns all started components and can execute the same shutdown sequence for:

- normal signal;
- startup error after partial construction;
- HTTP listener failure;
- HTTPS listener failure;
- gRPC serve failure;
- future telemetry bootstrap/exporter failure.

Future implementation should introduce a single runtime/supervisor with explicit `Start`, `Run` and `Shutdown` ownership or construct all fallible resources before starting any goroutine.

## F2 — Worker and gRPC components start before construction is complete

Severity: **HIGH**

The worker pool starts at `cmd/api/main.go:111-117`. gRPC starts at `cmd/api/main.go:140-145`. JWT verifier, auth middleware, RBAC, routers and HTTP listener binding are constructed later (`cmd/api/main.go:186-264`).

Examples of later failures:

- invalid JWT verifier configuration (`cmd/api/main.go:191-200`);
- auth/RBAC constructor failure (`cmd/api/main.go:202-210`);
- HTTP/TLS bind or certificate failure in `APIServer.Run` (`internal/api/server.go:56-81`).

On such failures, deferred signal cancellation eventually cancels the worker context, but the code does not synchronously wait for the worker pool or perform `FailActiveOnShutdown`. gRPC is explicitly stopped only in the narrow gRPC-client-construction failure branch (`cmd/api/main.go:147-152`), not on later construction failures.

Because `main` then calls `os.Exit(1)`, process termination becomes the final cleanup mechanism.

## F3 — `os.Exit(1)` makes deferred cleanup in `main` unusable

Severity: **HIGH for future telemetry**

`main` calls `os.Exit(1)` immediately after logging `run` failure (`cmd/api/main.go:43-45`). Go does not run deferred functions when `os.Exit` is called.

Currently most important defers are inside `run`, so they execute before the error reaches `main`. But a future pattern such as:

```text
telemetry := setup()
defer telemetry.Shutdown(...)
if err := run(); err != nil { os.Exit(1) }
```

would lose telemetry cleanup.

Telemetry shutdown must complete inside the supervised `run` lifecycle before returning to `main`, or `main` must compute an exit code and return naturally from a helper that owns all cleanup.

## F4 — Unexpected HTTP/HTTPS server error cleanup is asymmetric

Severity: **HIGH**

The listener-error branch (`internal/api/server.go:177-190`) does:

- readiness false;
- queue stop accepting;
- force-close HTTP and HTTPS;
- wait for HTTP goroutines;
- return the listener error;
- stop the worker pool later via defer.

It does **not** explicitly:

- close the event hub;
- stop gRPC;
- close the internal gRPC client before protocol shutdown;
- expose a component-shutdown outcome;
- flush future telemetry.

DB and outbound transport are eventually closed by `run` defers, but ordering is incidental rather than supervised.

Normal signal shutdown and unexpected listener failure should enter the same shutdown function with different causes, not separate cleanup implementations.

## F5 — A gRPC serve failure can produce a successful process exit

Severity: **HIGH / correctness bug**

When gRPC `Serve` returns unexpectedly, `Runtime.Start` logs the error and invokes the signal cancel function (`internal/transport/grpcserver/runtime.go:53-64`). It does not return the error to a supervisor.

If HTTP is already running, `APIServer.Run` receives `ctx.Done()`, performs normal shutdown and returns `nil` when cleanup succeeds (`internal/api/server.go:135-175`). `run` therefore returns `nil`, and the process can exit with status 0 even though a required gRPC component failed.

The root cancellation cause is also lost because `signal.NotifyContext` does not carry the gRPC error.

A fatal component must publish its error to a shared supervisor/error group. Shutdown cause and final process result must preserve that error.

## F6 — Forced HTTP shutdown is reported as success

Severity: **HIGH for operational truth**

If `http.Server.Shutdown` reaches its deadline, `shutdownServer` logs a warning, calls `Close`, then returns `nil` when `Close` succeeds (`internal/api/server.go:207-212`).

This hides that active requests were forcefully terminated. The application can report a clean shutdown even though requests were dropped.

The shutdown result needs a typed outcome such as:

```text
graceful
forced
timeout
failed
```

Forced termination may still be an accepted bounded fallback, but it must be returned or recorded as degraded shutdown and emitted into logs/metrics/traces.

## F7 — gRPC timeout is deliberately suppressed by `APIServer.Run`

Severity: **HIGH for operational truth**

`Runtime.Shutdown` returns `ctx.Err()` after forced `Stop` (`internal/transport/grpcserver/runtime.go:81-89`). `APIServer.Run` explicitly excludes `context.DeadlineExceeded` from accumulated shutdown errors (`internal/api/server.go:160-166`).

Therefore forced gRPC termination is also treated as clean shutdown.

The caller should distinguish an expected forced fallback from a graceful result without discarding the event.

## F8 — Shutdown has no global deadline

Severity: **HIGH**

Each component creates or receives a fresh `HTTP_SHUTDOWN_TIMEOUT` budget:

- HTTPS shutdown;
- HTTP shutdown;
- gRPC shutdown;
- deferred worker-pool stop.

These operations are mostly sequential (`internal/api/server.go:144-169`, then deferred pool stop at `125-132`). With TLS enabled, total duration can approach multiple times the configured timeout, before DB close, gRPC client close, outbound transport close and future telemetry flush.

A platform termination grace period is global. The application needs one global shutdown deadline and bounded component sub-budgets derived from it. HTTP and HTTPS can normally be drained concurrently.

## F9 — Worker wait and terminal repair consume the same context budget

Severity: **HIGH**

`WorkerPool.Stop` waits for `wg` and then calls `FailActiveOnShutdown` with the same context (`internal/workerpool/workerpool.go:82-105`). If workers consume most of the deadline, no time remains to repair queued/running job state.

If the context expires first, `Stop` returns without attempting `FailActiveOnShutdown` in that call.

Worker completion and durable terminal repair need separate bounded phases or reserved sub-budgets.

## F10 — Worker terminal fallback uses an unbounded background context

Severity: **MEDIUM/HIGH**

`markJobFailed` calls `SetFailed(context.Background(), ...)` (`internal/workerpool/workerpool.go:194-207`). Current PostgreSQL repositories add internal query timeouts, but the service interface itself does not guarantee this for every implementation.

This can keep a worker goroutine alive beyond the shutdown budget and prevents carrying shutdown cause/correlation metadata.

Use a detached but bounded repair context, preserving safe values/trace links without inheriting request cancellation.

## F11 — A timed-out `WorkerPool.Stop` leaves a waiter goroutine behind

Severity: **MEDIUM**

Every stop call creates a goroutine that waits on `wp.wg` (`internal/workerpool/workerpool.go:94-98`). If `Stop` returns on context expiration while a worker remains stuck, that waiter remains until all workers exit. If a worker never exits, it is permanent for the remaining process lifetime.

This is not the primary failure, but it complicates leak diagnostics and future telemetry shutdown. Pool lifecycle should expose a stable done channel created once, rather than creating one waiter per stop call.

## F12 — Closing the hub before shutting down HTTP is useful but creates an admission race

Severity: **MEDIUM**

The current order closes the hub before calling HTTP shutdown (`internal/api/server.go:144-158`). This is beneficial because it unblocks existing SSE handlers.

However, readiness false does not immediately prevent direct clients from reaching the API. During the interval before listener shutdown, a new SSE request can authenticate, reach `Subscribe`, receive `ErrHubClosed` and return `503`. Other normal HTTP requests can also begin.

A process-level admission gate should reject new business operations consistently as soon as shutdown begins, while allowing health behavior to reflect not-ready state. Active SSE subscriptions should then be closed to allow drain.

## F13 — SSE timing has a boundary race with server `WriteTimeout`

Severity: **MEDIUM; confirm in Audit 09/10**

Default HTTP `WriteTimeout` is 15 seconds and default SSE heartbeat is also 15 seconds. The SSE handler resets a per-write deadline immediately before heartbeat/event writes (`internal/routes/job_handler.go:109-117`), but the initial server write deadline can expire at roughly the same time as the first heartbeat.

This exact-boundary configuration is fragile and may differ between HTTP/1.1 and HTTP/2 behavior. It needs an integration test under the target Go toolchain. The heartbeat should occur comfortably before any applicable idle/write deadline.

## F14 — Component stop order is not explicitly tied to telemetry completion

Severity: **HIGH for future OTel**

There is no telemetry runtime today. When it is introduced, merely adding `defer provider.Shutdown(...)` is insufficient because defer registration order in `run` currently controls DB, outbound and gRPC-client cleanup.

Final spans may be created during:

- forced HTTP close;
- gRPC graceful/forced stop;
- worker repair;
- DB close failure;
- outbound client close;
- shutdown summary logging.

Telemetry must remain active through all of those events and then execute bounded `ForceFlush`/`Shutdown` as the final supervised phase.

## F15 — Collector/exporter outage policy is not implemented

Severity: **MISSING / expected**

The project has no OTel SDK or Collector yet, so it has no explicit behavior for:

- Collector unavailable at startup;
- Collector becomes unavailable under load;
- application-side batch queue full;
- exporter timeout/retry exhaustion;
- Collector sending queue full;
- Collector restart with in-memory data;
- telemetry memory pressure;
- final flush timeout.

The required policy is fail-open for business traffic:

- telemetry startup failure is non-fatal by default;
- telemetry is excluded from readiness;
- application-side queues are bounded;
- request handlers must not block indefinitely on export;
- dropped telemetry is counted and logged with rate limiting;
- Collector uses `memory_limiter`, `batch`, bounded sending queues and retry;
- persistent Collector queue is optional for later high-fidelity experiments;
- strict startup mode may exist only as an explicit laboratory/configuration option.

## F16 — Current tests do not cover shutdown outcome truth

Severity: **MEDIUM/HIGH**

Existing server tests cover:

- bind failure does not set ready;
- queued jobs fail on shutdown;
- running jobs fail on shutdown;
- HTTPS serves;
- missing TLS key pair does not set ready.

Missing tests include:

- unexpected HTTP/HTTPS serve error closes hub and gRPC;
- gRPC serve error reaches process result;
- forced HTTP close is reported as forced;
- forced gRPC stop is reported as forced;
- global shutdown budget is respected;
- worker repair receives reserved time;
- stuck worker does not block telemetry flush forever;
- startup failure after pool/gRPC start cleans all components;
- Collector unavailable does not affect readiness;
- telemetry queue full does not block HTTP;
- final spans/metrics are flushed on normal and error exits;
- flush timeout is visible and bounded.

## Target lifecycle contract

## Construction phase

Preferred rule:

1. load and validate configuration;
2. create passive dependencies/resources;
3. bind all required listeners;
4. create telemetry runtime;
5. construct middleware/routers/services;
6. start goroutines only after the application is fully constructible;
7. mark ready only after required components report running.

If any component must start earlier, it must immediately register a synchronous idempotent cleanup action with the supervisor.

## Run phase

All fatal component errors must enter one shared supervisor channel/error group:

```text
HTTP Serve error
HTTPS Serve error
gRPC Serve error
worker fatal error
root signal/cancellation
```

The first terminal cause begins shutdown, but additional component errors should be joined into the final result where useful.

Use cancellation with cause or an explicit terminal-event type so the shutdown log and final process code retain the original reason.

## Shutdown phase

Recommended target sequence under one global deadline:

```text
1. record shutdown cause and start time
2. readiness=false
3. close business admission gate / queue.StopAccepting
4. stop accepting new protocol work
5. close SSE hub to release long-lived streams
6. drain HTTP and HTTPS concurrently
7. gracefully stop gRPC, force stop on sub-deadline
8. cancel and wait workers
9. perform bounded terminal job repair
10. close gRPC client, outbound idle connections and DB
11. emit shutdown summary and component outcomes
12. ForceFlush telemetry
13. Shutdown telemetry providers/exporters
14. return joined final result / correct exit status
```

Exact HTTP/SSE ordering may use a coordinated admission gate because SSE streams need explicit release before HTTP drain can finish.

## Shutdown outcome model

Every component should return an outcome, not only an error:

```text
not_started
graceful
forced
timeout
failed
```

The final log/metric should include bounded component names and durations:

```text
shutdown.cause
shutdown.duration
shutdown.outcome
component=http|https|grpc|workers|jobs_repair|db|telemetry
component.outcome
```

No request ID, job ID or error string should become a metric label.

## Telemetry reliability contract

### Application SDK

- use asynchronous/batch processors/readers;
- keep queues bounded;
- do not enable blocking-on-full on request paths by default;
- set explicit export timeout and batch delay;
- use parent-based sampling;
- expose SDK/exporter health as diagnostics, not readiness;
- perform final `ForceFlush` and `Shutdown` under bounded contexts;
- shutdown is idempotent;
- after shutdown, instrumentation must safely become no-op or reject internally without panicking.

### Collector

Minimum processors:

```text
memory_limiter
batch
```

Remote exporters need bounded sending queues and retry. Queue capacity must be sized from measured telemetry volume and acceptable backend outage, not guessed indefinitely upward.

Monitor at least:

```text
otelcol_exporter_queue_size
otelcol_exporter_queue_capacity
otelcol_exporter_send_failed_*
otelcol_exporter_enqueue_failed_*
otelcol_processor_refused_*
otelcol_receiver_refused_*
otelcol_process_memory_*
```

### Failure policy

| Failure | Application behavior | Required signal |
|---|---|---|
| Collector absent at startup | start in fail-open mode | one bounded warning + telemetry disabled/degraded gauge |
| Collector unavailable at runtime | continue business traffic | exporter failures, queue utilization, drops |
| application telemetry queue full | drop telemetry rather than block indefinitely | SDK queue-full/drop metric/log |
| Collector queue full | bounded drop/backpressure according to lab policy | Collector queue/refused metrics |
| final flush timeout | process exits after global deadline | shutdown outcome `telemetry_timeout` |
| strict laboratory mode enabled | startup may fail intentionally | explicit config and clear log |

## Required tests before observability rollout

### Lifecycle tests

- all started components are stopped after every constructor failure point;
- HTTP, HTTPS and gRPC serve errors are returned as fatal causes;
- signal shutdown returns clean only when every required component was graceful;
- forced shutdown is distinguishable from graceful shutdown;
- total shutdown duration stays within one global deadline plus a small scheduling tolerance;
- repeated shutdown is idempotent;
- readiness never returns true before required components are running.

### Worker/job tests

- cancellation near the stop deadline still leaves time for terminal repair;
- repair uses a bounded detached context;
- a stuck repository does not hold process shutdown forever;
- queue items and running jobs receive deterministic shutdown outcomes.

### Telemetry tests

Use in-memory/manual exporters/readers where possible:

- normal shutdown exports final HTTP/gRPC/worker spans;
- error shutdown exports fatal component and shutdown-summary spans;
- exporter `ForceFlush` timeout is bounded and visible;
- Collector/exporter unavailable does not change readiness;
- queue saturation does not block request handlers;
- SDK/Collector drop counters increase under configured overload;
- telemetry provider shutdown is called exactly once;
- logs produced during shutdown still include trace correlation before provider shutdown.

## Implementation prerequisites produced by this audit

Before adding Grafana dashboards or load experiments, create implementation tasks for:

1. introduce a unified application supervisor/runtime;
2. separate construction/binding from goroutine start;
3. propagate fatal HTTP/HTTPS/gRPC component errors;
4. preserve shutdown cause and final exit code;
5. use one global shutdown budget with component sub-budgets;
6. return graceful/forced/timeout/failed component outcomes;
7. reserve a separate bounded job-repair budget;
8. replace unbounded worker repair background context;
9. add a stable worker-pool done channel;
10. add a shutdown admission gate;
11. normalize SSE heartbeat/write-timeout relationship;
12. add telemetry runtime as the final lifecycle-owned component;
13. configure bounded non-blocking SDK and Collector queues;
14. add shutdown/exporter fault tests.

## Final assessment

The existing service demonstrates awareness of graceful shutdown and has useful building blocks, but cleanup is currently path-dependent and partially hidden behind defers and process termination.

The most important prerequisite for observability is **not** selecting Grafana panels. It is making lifecycle outcomes truthful and deterministic. Without that, final spans will be dropped exactly during the failures where they are most valuable, and the process may report success after a required protocol component has failed.
