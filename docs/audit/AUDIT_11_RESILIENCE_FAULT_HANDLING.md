# Audit 11 — Resilience and Fault Handling

## Scope

This pass evaluates how the uploaded `pet-study` working tree behaves under dependency failure, overload, partial persistence failure, process interruption and protocol degradation.

The pass covers:

- outbound Profile timeout/retry/backoff behavior;
- circuit-breaker and retry-amplification controls;
- PostgreSQL startup/runtime failure handling;
- async queue overflow, persistence/queue atomicity and crash recovery;
- job terminal-state repair and partial success;
- readiness/liveness degradation policy;
- HTTP/gRPC error classification under cancellation and dependency failure;
- SSE delivery loss/reconnect behavior;
- admission controls and noisy-neighbor behavior;
- fault-injection and recovery-test coverage.

The pass does not change application code. Two audit-only tests were created and removed in the disposable Go 1.23.2 compatibility copy:

1. exponential-backoff overflow at a high attempt number;
2. acceptance of a successful Profile response containing a mismatched `user_id` and trailing garbage.

Full Go 1.25.8 integration and PostgreSQL fault tests remain required.

## Reviewed evidence

Primary files:

- `cmd/api/main.go`
- `internal/api/server.go`
- `internal/config/config.go`
- `internal/db/open.go`
- `internal/db/tx.go`
- `internal/health/health.go`
- `internal/health/router.go`
- `internal/httputils/apphandler.go`
- `internal/httputils/errmap.go`
- `internal/middleware/bulkhead.go`
- `internal/middleware/rate_limiter.go`
- `internal/outbound/profile.go`
- `internal/outbound/profile_retry.go`
- `internal/outbound/profile_instrumentation.go`
- `internal/outbound/httpclient/client.go`
- `internal/queue/queue.go`
- `internal/routes/users_handler.go`
- `internal/routes/users_handler_v2.go`
- `internal/routes/job_handler.go`
- `internal/service/service.go`
- `internal/service/jobService.go`
- `internal/service/user_profile_service.go`
- `internal/store/jobrepo/memory_job_repo.go`
- `internal/store/jobrepo/sqlx.go`
- `internal/store/userrepo/memory_user_repo.go`
- `internal/store/userrepo/sqlx.go`
- `internal/stream/stream.go`
- `internal/transport/grpcclient/grpcclient.go`
- `internal/transport/grpcserver/runtime.go`
- `internal/transport/grpcserver/grpc_job_service.go`
- `internal/testkit/testkit.go`
- migrations, Docker Compose, README and related tests.

## Executive result

Resilience is **PARTIAL/BROKEN**.

The project already has useful defensive primitives:

- bounded queue and subscriber buffers;
- fast-fail queue/bulkhead behavior;
- request-scoped and repository-scoped timeouts;
- idempotent-method retries with exponential backoff and jitter;
- bounded HTTP headers/body input;
- startup DB ping;
- readiness checks;
- graceful HTTP/gRPC shutdown attempts;
- a PostgreSQL uniqueness constraint that protects email creation;
- context-aware outbound requests and retry sleep.

However, the async subsystem is not crash-recoverable and cannot guarantee accepted-job terminality. The durable job row does not contain the payload required to replay work, while the payload exists only in the process-local channel. User creation and job success persistence are not atomic. A crash, DB transition error or process restart can therefore leave a user committed while the job remains `running`, or leave a `queued` job that can never be processed.

The outbound path is bounded by context but lacks a circuit breaker, retry budget, upstream-specific concurrency bound and strict response validation. HTTP/gRPC error mappings also collapse several cancellation, deadline and dependency-failure conditions into generic `500/Internal` responses.

These issues must be corrected before introducing a durable broker, horizontal workers or meaningful reliability/SLO experiments.

## Dynamic audit evidence

### Exponential-backoff overflow

An audit-only test called the existing `backoffWithJitter` with:

```text
baseDelay = 50ms
maxDelay  = 500ms
attempt   = 100
```

The method returned `0` because `time.Duration` multiplication overflowed before the cap was applied. The implementation multiplies repeatedly and only applies `maxDelay` after the loop.

A sufficiently high configured `OUTBOUND_RETRY_MAX_ATTEMPTS` can therefore turn later retries into immediate retries rather than capped delays.

### Profile response semantic/trailing-data acceptance

An audit-only upstream returned HTTP 200 with:

```text
{"user_id":999,"bio":"ok","city":"x"} trailing-garbage
```

for requested user ID `42`.

The current client returned success with `Profile.UserID=999` because it:

- decodes one JSON value;
- does not require EOF after the first value;
- drains and ignores trailing bytes;
- does not verify that the returned `user_id` matches the requested user.

This confirms that a malformed or cross-user upstream response can be accepted as valid success.

## Positive findings

### P1 — Outbound operations use one caller-owned time budget

`UserProfileService` derives a timeout from the incoming request context and passes it through the retry wrapper and HTTP request. Retry sleep is context-aware, and the wrapper checks the remaining deadline before sleeping.

This prevents unconstrained total retry time and preserves cancellation.

### P2 — Retries are limited to a GET operation

The Profile operation is a `GET`, so the current retry policy does not blindly retry a non-idempotent HTTP write.

### P3 — Several failure classes are typed

The Profile client distinguishes:

- canceled;
- timeout;
- network;
- upstream 4xx;
- upstream 5xx;
- parse;
- bad response.

This is a sound base for policy and telemetry, although the downstream mappings remain incomplete.

### P4 — Storage operations are timeout-bounded

SQLX repository methods derive a query timeout from the incoming context. DB startup ping also has a timeout. This avoids unbounded DB calls in the reviewed repositories.

### P5 — Queue and SSE subscriber buffers are bounded

The queue fast-fails when full, and Hub publish does not block a worker on a slow subscriber. This protects the process from unbounded application-managed buffering.

### P6 — Email uniqueness has a final persistence guard

The service performs an existence check, but both storage implementations retain a final guard:

- memory repo rechecks under its write lock;
- PostgreSQL uses a unique index and maps duplicate-key conflict.

This prevents duplicate email rows despite the check-then-insert race.

## Findings

## F1 — Accepted async work is not durable or crash-recoverable

Severity: **CRITICAL**

The durable job row stores status, owner and result/error fields, but not the `CreateUserInput` payload. The payload exists only in `queue.WorkItem` inside a process-local channel.

Failure windows include:

```text
job row committed
→ process crashes before channel enqueue
→ durable queued job has no payload and can never be replayed
```

and:

```text
item enqueued in memory
→ process crashes
→ channel content disappears
→ durable queued/running job remains
```

Startup does not scan/recover queued or running jobs. `FailActiveOnShutdown` only applies during the graceful shutdown path, not a crash, kill -9, OOM or host loss.

Required direction:

- transactional outbox or durable broker publish;
- payload/reference persisted durably;
- startup reconciliation;
- attempt/lease/ownership metadata;
- idempotent consumer behavior.

## F2 — User creation and job terminal success are not atomic

Severity: **CRITICAL**

Worker processing performs separate operations:

```text
SetRunning
CreateUser
SetSucceeded
```

If `CreateUser` commits and `SetSucceeded` fails:

- the user exists;
- the job remains `running` or is later repaired to `failed`;
- replaying the job can produce conflict rather than a reliable success result.

`internal/db/tx.go` exists but is not used by the user/job operation, and the two repository abstractions do not share a transaction boundary.

Required direction:

- one transactional application operation for PostgreSQL-backed execution; or
- idempotency key plus reconciliation that can discover the already-created user; or
- outbox/saga semantics with explicit partial-success repair.

## F3 — Shared-database shutdown repair is unsafe for horizontal workers

Severity: **CRITICAL for future multi-instance deployment**

`FailActiveOnShutdown` updates every job with status `queued` or `running` in the database.

There is no:

- worker instance ID;
- lease owner;
- lease expiry;
- heartbeat;
- partition/consumer ownership.

If two application instances share PostgreSQL, shutdown of one instance can mark work owned by another instance as failed.

Before horizontal scaling, repair must target only jobs leased/owned by the stopping instance, or move work ownership to a broker/lease model.

## F4 — Queue-full compensation can create orphan jobs and amplify DB load

Severity: **HIGH**

Async handlers currently:

```text
INSERT job
→ attempt non-blocking enqueue
→ on queue full/stopped, DELETE job
```

Problems:

1. Under saturation, every rejected request performs an insert followed by delete, increasing DB pressure precisely when the system is overloaded.
2. Delete uses the original request context. If the context is canceled, cleanup can fail and leave a `queued` job outside the queue.
3. `errors.Join(enqueueErr, deleteErr)` is mapped by the queue error first, so the cleanup failure is returned as an ordinary queue `503` and is not centrally logged as an unexpected persistence failure.
4. Crash between insert and enqueue creates the same orphan without executing compensation.

Required direction:

- capacity/admission reservation before durable acceptance; or
- transactional outbox where durable acceptance is the queue; or
- persist a terminal `rejected` outcome with bounded detached cleanup rather than deleting history.

## F5 — No poison-job, retry, DLQ or reconciliation model exists

Severity: **HIGH**

When job transition persistence fails, the item is consumed and not retried. When user creation fails, the job is marked failed immediately. There is no:

- attempt count;
- retryable/non-retryable classification for job execution;
- next-attempt timestamp;
- dead-letter state;
- repair queue;
- reconciliation worker;
- operator retry command.

A transient DB error in `SetRunning`, `CreateUser` or `SetSucceeded` can permanently violate accepted-job terminality.

## F6 — Worker panics are not contained or supervised

Severity: **HIGH**

Worker goroutines have no panic boundary. A panic in job processing, repository code, observer or event publication terminates the entire Go process.

HTTP recover middleware does not protect worker goroutines.

A production policy must be explicit:

- recover per job, record panic telemetry and mark/requeue the job; or
- deliberately crash the process under a supervisor after preserving enough durable state.

The current behavior is accidental rather than policy-driven.

## F7 — SSE can permanently miss the terminal event

Severity: **HIGH**

The SSE handler:

1. reads the current job;
2. authorizes;
3. subscribes to the in-memory Hub;
4. waits for future events.

A terminal transition can occur between the initial read and subscription. The event is then published with no subscriber, and the connected client receives only heartbeats forever.

Additionally, Hub overflow drops events without replay. There is no:

- initial state event after subscription;
- transition sequence/version;
- event ID;
- `Last-Event-ID` replay;
- durable event log;
- close-on-terminal fallback.

At minimum, subscribe/re-read/version logic or an initial current-state event is required. For durable semantics, events need a persisted transition log/outbox.

## F8 — Streaming errors are routed through Problem+JSON after the stream may be committed

Severity: **HIGH/PARTIAL**

After SSE headers or events have been written, a heartbeat/write/flush failure is returned to `AppHandler`. `AppHandler` then calls `WriteProblem`, attempting to change content type/status and write JSON to an already-committed event stream.

The same risk exists for recovered panics after a response is committed.

Streaming handlers need a committed-stream error path that terminates/logs the connection without attempting a second HTTP response.

## F9 — Dependency/cancellation errors collapse into misleading HTTP 500 responses

Severity: **HIGH**

`MapError` has specific Profile mappings, but generic repository and context failures are not classified.

Examples:

- PostgreSQL query deadline exceeded -> generic 500;
- PostgreSQL unavailable -> generic 500;
- request context canceled inside service/repository -> generic 500 and attempted Problem write;
- joined/ wrapped DB errors -> generic 500.

This obscures operational cause and makes retry/SLO interpretation unreliable.

A stable application error taxonomy is needed for:

- canceled request;
- application deadline;
- DB unavailable;
- DB timeout;
- conflict;
- internal invariant failure.

## F10 — gRPC error mapping loses cancellation, deadline and availability semantics

Severity: **HIGH**

The gRPC server maps every non-not-found repository error to `codes.Internal`. It does not map:

- `context.Canceled` -> `codes.Canceled`;
- `context.DeadlineExceeded` -> `codes.DeadlineExceeded`;
- dependency unavailability -> `codes.Unavailable`.

The HTTP bridge maps only a subset of codes. `Unavailable`, `DeadlineExceeded`, `Canceled` and `Internal` fall through to a generic HTTP 500.

This prevents callers from applying safe retry policy and corrupts error telemetry.

## F11 — HTTP-to-gRPC bridge has no explicit call budget

Severity: **HIGH/PARTIAL**

The bridge uses `r.Context()` directly. The server's `WriteTimeout` is not an application context deadline, and the gRPC client is created lazily without a startup blocking dial.

The call should have a route-specific deadline smaller than the HTTP response budget and should classify connection/unavailability separately.

## F12 — No circuit breaker or retry budget protects the Profile dependency

Severity: **HIGH**

The client has attempts/backoff/jitter but no:

- circuit breaker;
- concurrent retry budget;
- Profile-specific bulkhead;
- bounded active connections by default (`MaxConnsPerHost=0`);
- logical-operation success/failure measurement;
- exhausted-retry metric.

When retries are enabled and the upstream degrades, every caller can independently amplify load.

The existing global API bulkhead is not a substitute because it mixes unrelated routes and includes SSE.

## F13 — Backoff can overflow before applying its configured cap

Severity: **HIGH when attempts are misconfigured**

Dynamically reproduced.

`backoffWithJitter` multiplies a `time.Duration` in a loop and applies `maxDelay` only after all multiplications. `OUTBOUND_RETRY_MAX_ATTEMPTS` has no safe upper bound or cross-field validation.

Required direction:

- cap before multiplication;
- overflow-safe arithmetic;
- practical upper bound for attempts;
- validate `baseDelay <= maxDelay` where policy requires it.

## F14 — Retry policy is too coarse for HTTP status and server guidance

Severity: **MEDIUM**

All upstream 5xx responses are retryable, including statuses that may represent stable non-transient behavior. Upstream `Retry-After` is ignored. HTTP 429 is classified as non-retryable 4xx without a documented policy.

The retry policy should explicitly define status classes/codes, honor remaining operation budget and optionally parse bounded `Retry-After` for supported statuses.

## F15 — Profile response validation is not strict enough

Severity: **HIGH**

Dynamically reproduced.

The success path lacks:

- response body size limit;
- Content-Type validation;
- EOF/trailing-data validation;
- semantic required-field validation;
- requested ID versus returned ID validation.

A compromised or buggy upstream can return an oversized, concatenated or cross-user response that is accepted or consumes the full operation budget while being drained.

## F16 — Profile 4xx semantics are flattened to HTTP 502

Severity: **MEDIUM/POLICY GAP**

All upstream 4xx responses become `502 Bad Gateway`. This may be correct for some contract violations, but the policy does not distinguish:

- profile not found;
- upstream auth/configuration failure;
- rate limiting;
- malformed request generated by this service.

The client should retain normalized upstream status/reason while exposing a deliberate public contract.

## F17 — POST operations have no idempotency contract

Severity: **HIGH**

A client can lose the response after the server commits a synchronous user or accepts an async job. Retrying the same POST can create:

- duplicate jobs;
- a conflict caused by a user created by the first attempt;
- ambiguous client state.

Unique email prevents duplicate user rows but does not provide request-level idempotency or return the original result.

Future high-load and broker work should add an idempotency key and durable result record for create operations.

## F18 — Startup readiness validates connectivity, not schema or migrations

Severity: **HIGH/PARTIAL**

PostgreSQL startup performs `PingContext`, and readiness repeats ping. Neither verifies that required tables, columns or migrations exist.

The application can become ready with a reachable empty/incompatible database, then fail every business request.

There is also no startup retry/backoff for temporary DB unavailability. Fail-fast can be a valid policy under an orchestrator, but it must be paired with deployment restart policy and migration gating. The provided Compose file only starts PostgreSQL and has no app health/restart orchestration.

## F19 — Readiness checks are sequential under one shared 200 ms deadline

Severity: **MEDIUM**

One slow check can consume the shared deadline and prevent later checks from executing. Returned details then reflect ordering rather than the complete component state.

DB ping on every readiness request also needs controlled probe frequency in deployment configuration.

Profile upstream and telemetry correctly do not participate in readiness, preventing cascading eviction, but there is no separate degraded/dependency status surface.

## F20 — Global admission controls enable cross-route noisy-neighbor failure

Severity: **HIGH**

One rate limiter and one bulkhead protect every API route, before authentication. Therefore:

- anonymous/invalid traffic can consume the global token budget;
- one slow Profile operation can reject unrelated reads;
- one SSE connection can occupy the default only bulkhead slot;
- traffic classes cannot reserve capacity.

Required direction:

- cheap edge/global abuse limiter;
- principal/route-aware limits after authentication where appropriate;
- resource-specific bulkheads;
- independent SSE connection limits.

## F21 — Queue overload response lacks durable admission semantics

Severity: **MEDIUM/HIGH**

`503 queue is full` indicates fast-fail overload, but there is no `Retry-After`, idempotency token or durable acceptance identifier for safe client retry.

A retry can create a new job row and repeat the insert/delete compensation path.

## F22 — Job model lacks recovery metadata

Severity: **HIGH**

The job model/table lacks:

- payload or durable payload reference;
- idempotency key;
- attempt count;
- next-attempt time;
- lease owner/expiry;
- transition version;
- failure kind;
- trace/request correlation;
- last heartbeat.

Without these fields, reliable requeue, recovery, horizontal workers and broker migration cannot be implemented safely.

## F23 — `WithinTransaction` exists but is not integrated into application operations

Severity: **MEDIUM**

The transaction helper is unused, and repository interfaces accept only `context.Context`, not a unit-of-work/transaction boundary.

This creates the partial-commit problem described in F2 and prevents atomic job/outbox/user changes.

## F24 — Fault-injection coverage is insufficient

Severity: **PROCESS GAP**

Existing tests cover successful retries, 4xx no-retry, timeout retries, parse error, network error, queue overflow and some shutdown behavior.

Missing scenarios include:

- PostgreSQL unavailable at startup and runtime;
- schema/migration missing;
- DB pool exhaustion;
- transition write failure after user commit;
- crash between job save/enqueue and enqueue/response;
- process restart with queued/running rows;
- worker panic;
- repeated retry storm;
- circuit-open/half-open behavior;
- oversized/never-ending upstream body;
- malformed/trailing/mismatched Profile response;
- gRPC unavailable/deadline/cancellation mapping;
- SSE transition between snapshot and subscribe;
- SSE subscriber overflow and terminal-event loss;
- queue/full compensation failure;
- multi-instance shutdown repair;
- shutdown under dependency stall.

## Failure-mode matrix

| Failure | Current outcome | Reliability issue |
|---|---|---|
| Profile timeout | retry if budget remains, then 503 | no circuit/retry budget; attempts only |
| Profile 5xx | retry all 5xx | coarse policy; ignores Retry-After |
| Profile malformed trailing response | may be accepted | incomplete decode validation |
| Profile wrong user ID | accepted | semantic integrity failure |
| PostgreSQL unavailable at startup | process exits | no startup retry/orchestrator policy in repo |
| PostgreSQL unavailable at request time | generic 500 | no dependency taxonomy or retry guidance |
| Queue full | job insert then delete, 503 | DB amplification and orphan window |
| Crash after job insert | queued row remains | no payload/recovery |
| Crash after enqueue | in-memory item lost | no durable broker/outbox |
| User commit then job success write fails | user exists, job nonterminal | no transaction/idempotent reconciliation |
| Worker panic | process crash | no explicit panic policy/supervision |
| SSE event before subscribe | event lost | no snapshot/replay/version |
| SSE subscriber slow | events dropped | terminal event may be lost indefinitely |
| gRPC DB deadline | Internal/HTTP 500 | semantic code loss |
| One instance graceful shutdown in shared DB | all active rows failed | no lease/instance ownership |

## Required remediation order

Before brokers, horizontal scaling or reliability SLO experiments:

1. Fix job compare-and-set state transitions from Audit 10.
2. Redesign worker lifecycle as one supervised generation.
3. Define durable async acceptance: outbox/broker plus persisted payload/reference.
4. Add idempotency keys and a durable operation-result contract.
5. Add job lease/attempt/version/failure metadata and reconciliation.
6. Make user creation + job/outbox result atomic or safely idempotent.
7. Replace insert-then-delete queue compensation with a durable admission design.
8. Add worker panic policy and fatal-error supervision.
9. Correct context/DB/gRPC error classification.
10. Add explicit gRPC bridge timeout.
11. Add Profile-specific bulkhead, connection bound, retry budget and circuit breaker.
12. Harden Profile body/content/semantic validation.
13. Fix SSE snapshot/subscription race and define replay/terminal behavior.
14. Separate streaming error handling from Problem+JSON response writing.
15. Add migration/schema startup gate and deployment restart/health policy.
16. Introduce fault-injection harnesses and tests.

## Target resilience shape

```text
HTTP create request
  → idempotency lookup/reservation
  → durable transaction:
       job + payload/reference + outbox event
  → 202 only after durable acceptance

Broker/dispatcher
  → lease/claim job
  → per-attempt context and retry policy
  → idempotent user operation
  → CAS terminal transition
  → transition event/outbox
  → retry/DLQ/reconciliation on failure

Profile dependency
  → route-specific timeout budget
  → Profile bulkhead + MaxConnsPerHost
  → circuit breaker
  → bounded retry budget
  → strict response limit/content/schema validation

SSE
  → subscribe + versioned snapshot
  → durable/monotonic transition events
  → bounded delivery with replay/resync policy
```

## Audit status

- Outbound time budgeting: **PRESENT/PARTIAL**
- Retry implementation: **PARTIAL/BROKEN at extreme configuration**
- Circuit breaker/retry budget: **MISSING**
- Strict upstream response validation: **BROKEN/PARTIAL**
- Durable async acceptance: **MISSING/BROKEN**
- Crash recovery/reconciliation: **MISSING**
- Job execution atomicity/idempotency: **BROKEN**
- Multi-instance job ownership: **MISSING**
- SSE delivery resilience: **BROKEN/PARTIAL**
- DB/gRPC failure taxonomy: **PARTIAL/BROKEN**
- Readiness dependency policy: **PARTIAL**
- Fault-injection coverage: **INSUFFICIENT**
