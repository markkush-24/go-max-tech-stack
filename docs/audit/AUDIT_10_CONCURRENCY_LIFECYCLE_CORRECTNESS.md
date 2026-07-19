# Audit 10 — Concurrency and Lifecycle Correctness

## Scope

This pass evaluates concurrency safety and lifecycle state correctness in the uploaded `pet-study` working tree.

The pass covers:

- queue admission and cancellation semantics;
- worker-pool start/stop/restart ownership;
- `WaitGroup`, mutex, atomic and channel usage;
- async job state transitions;
- stream Hub subscribe/publish/close behavior;
- HTTP/gRPC lifecycle concurrency;
- global process registries and counters;
- repository locking and copy discipline;
- concurrency-focused test coverage.

The pass does not change application code. Audit-only tests were created and removed in the disposable Go 1.23.2 compatibility copy. Full Go 1.25.8 `-race` coverage remains required.

## Reviewed evidence

Primary files:

- `cmd/api/main.go`
- `internal/api/server.go`
- `internal/queue/queue.go`
- `internal/workerpool/workerpool.go`
- `internal/stream/stream.go`
- `internal/routes/users_handler.go`
- `internal/routes/users_handler_v2.go`
- `internal/routes/job_handler.go`
- `internal/service/jobService.go`
- `internal/service/service.go`
- `internal/store/jobrepo/memory_job_repo.go`
- `internal/store/jobrepo/sqlx.go`
- `internal/store/userrepo/memory_user_repo.go`
- `internal/store/userrepo/sqlx.go`
- `internal/transport/grpcserver/runtime.go`
- `internal/middleware/bulkhead.go`
- `internal/middleware/rate_limiter.go`
- `internal/middleware/authorization.go`
- `internal/middleware/rbac.go`
- `internal/middleware/cors.go`
- `internal/health/health.go`
- `internal/health/router.go`
- `internal/testkit/testkit.go`

Repository-wide searches were performed for goroutine creation, channels, `sync.Mutex`, `sync.RWMutex`, `sync.Once`, `sync.WaitGroup`, atomics, `select`, `close`, `Start` and `Stop` methods.

## Executive result

Concurrency correctness is **PARTIAL/BROKEN**.

The project has several sound low-level primitives:

- memory repositories protect maps with `RWMutex` and return copies;
- request IDs use an atomic sequence;
- readiness and gRPC lifecycle flags use atomics;
- outbound jitter protects `rand.Rand` with a mutex;
- the SSE Hub serializes send/close/subscription mutation under one mutex;
- queue and subscriber channels are bounded;
- Hub publish is non-blocking per subscriber.

However, worker-pool lifecycle and async job state transitions have critical semantic races:

1. `WorkerPool.Stop` is not truly idempotent;
2. a second `Stop` can mark a job failed while the original worker is still processing it;
3. the still-running worker can later overwrite that terminal failure with success;
4. a timed-out `Stop` sets `running=false` before workers have exited, allowing unsafe restart;
5. restart can overwrite the shared worker context and reuse the same `WaitGroup` while an earlier `Wait` is still active;
6. job repository transitions do not enforce expected source states;
7. a canceled enqueue context can still enqueue work;
8. queued metrics/events are emitted after the item is visible to workers, so transition events can be observed out of order.

These defects must be corrected before durable brokers, multiple worker generations, retries/requeue or high-load experiments are added.

## Dynamic audit evidence

### Queue cancellation selection

An audit-only test repeatedly called `Enqueue` with:

- a context canceled before the call;
- a queue with one free buffered slot.

Result across 20,000 iterations:

```text
accepted = 9,842
canceled = 10,158
```

Both the send and `ctx.Done()` cases are ready, so the `select` may choose either. The current method therefore does not guarantee that an already-canceled operation is rejected.

### Stream Hub race stress

An audit-only `-race` test concurrently executed:

- subscribe;
- publish;
- receive;
- unsubscribe;
- Hub close.

The test passed without a race report or send-on-closed panic in the disposable Go 1.23.2 copy.

This does not prove all Hub semantics, but it confirms that the current mutex/atomic ordering protects the tested send/close mutation path.

### Repeated worker Stop and terminal overwrite

An audit-only test used the original `WorkerPool` logic with local stub repositories:

1. a worker entered a deliberately blocked user save;
2. the first `Stop` canceled the pool but returned immediately through an already-canceled stop context;
3. the second `Stop` observed `running=false` and executed `FailActiveOnShutdown` while the worker was still active;
4. the job became `failed`;
5. the worker was released and then wrote `succeeded` over the shutdown failure.

Observed log/result:

```text
marked active jobs failed on shutdown count=1
job was failed by second Stop, then overwritten to succeeded by the still-running worker
```

The test passed under `-race` because this is primarily a state-machine/lifecycle race, not necessarily an unsynchronized memory access in the stub repository.

## Findings

## F1 — `WorkerPool.Stop` is not idempotent

Severity: **CRITICAL**

`Stop` sets `running=false` before workers have necessarily exited. A subsequent call takes the `!running` branch and immediately calls `failActiveOnShutdown`.

Therefore repeated or concurrent calls can perform new persistence side effects while the first stop is still in progress.

Idempotent lifecycle shutdown should return the same completion result or wait on one shared stop operation. It should not run terminal repair independently on every repeated call.

Required direction:

- one stop `sync.Once`/state transition;
- one stable `done` channel;
- one stored stop result;
- all callers wait on the same stop generation.

## F2 — Active worker can overwrite shutdown terminal state

Severity: **CRITICAL**

`FailActive` marks queued/running jobs failed, but `SetSucceeded` and `SetFailed` update by ID without checking the previous status.

A worker that completes after shutdown repair can overwrite:

```text
running -> failed(shutdown) -> succeeded
```

The opposite ordering can also overwrite a legitimate terminal result.

This applies to both memory and SQLX repositories. SQL updates use `WHERE id = ...`, not an expected-state condition.

Required direction:

- explicit job state machine;
- compare-and-set transitions, for example `queued -> running`, `running -> succeeded|failed`;
- terminal states immutable;
- repository returns a typed transition-conflict error;
- shutdown repair only changes active states;
- worker handles lost transition races explicitly.

## F3 — Timed-out Stop allows unsafe restart of the same pool

Severity: **CRITICAL**

`Stop` changes `running` to false before waiting. If its context expires, workers may still be running, but `Start` is allowed again.

`Start` then overwrites shared fields:

```text
wp.ctx
wp.cancel
```

Old workers read `wp.ctx` from the struct on every loop/operation rather than capturing a generation-local context. An old worker can therefore begin using the new generation context and continue processing after its original generation was canceled.

Required direction:

- either declare WorkerPool single-start/no-restart and reject later starts;
- or create an explicit generation object containing immutable `ctx`, `cancel`, `done`, and worker count;
- worker loops receive generation context as an argument instead of reading mutable pool fields.

## F4 — `WaitGroup` lifecycle can be reused before previous Wait returns

Severity: **CRITICAL**

Every `Stop` creates a goroutine that calls `wp.wg.Wait()`. If the stop context expires, that waiter remains alive until all workers exit.

Because `running` is already false, another `Start` may call `wg.Add` for a new generation while an old `Wait` is still active. This violates the intended independent-generation reuse discipline and can mix old and new workers in one counter.

Required direction:

- no restart until the prior generation `done` is closed;
- preferably do not reuse the same `WaitGroup` across generations;
- use a generation-local `WaitGroup` and stable `done` channel.

## F5 — Stop timeout can leak waiter goroutines

Severity: **HIGH**

Each `Stop` creates:

```text
go func() {
    wp.wg.Wait()
    close(done)
}()
```

If a worker blocks forever and the stop context expires, this waiter remains blocked forever. Repeated `Stop` calls can create more waiter goroutines.

A stable `done` channel closed once by the worker generation removes the need to create a new waiter per call.

## F6 — `running` does not represent actual worker health

Severity: **HIGH**

`running=true` means only that `Start` was called.

It can remain true when:

- the parent context was already canceled and all workers immediately exited;
- `workers` is zero when `Start` is called directly;
- all loops have exited for lifecycle reasons before `Stop` updates the flag.

Conversely, `running=false` can coexist with workers that are still executing after a timed-out stop.

Readiness therefore observes an administrative flag, not actual active worker generation health.

Required direction:

- lifecycle state enum such as `new/starting/running/stopping/stopped/failed`;
- generation done signal;
- active-worker gauge/count;
- readiness based on state and expected worker count.

## F7 — Canceled context can still enqueue work

Severity: **HIGH**

`Queue.Enqueue` checks `closed`, then executes one `select` containing:

- send to queue;
- receive from `ctx.Done()`;
- default/full case.

If the context is already canceled and the channel has capacity, both the send and cancellation cases are ready and either may be chosen.

This was dynamically reproduced.

Required direction:

- check `ctx.Err()` before attempting the send;
- define whether cancellation or acceptance wins if cancellation happens concurrently;
- document a clear linearization point;
- test canceled-before-call, cancel-during-call, full queue and stopped queue separately.

## F8 — Queue stop is not a strict barrier for an already-entered Enqueue

Severity: **MEDIUM/HIGH**

`StopAccepting` is an atomic store. An `Enqueue` that loaded `closed=false` before the store may complete its channel send after `StopAccepting` returns.

This may be acceptable if the contract permits already-admitted calls to finish, but it is not a strict “no successful send after return” barrier.

The shutdown policy must explicitly define the admission linearization point. A strict barrier would require coordinated locking or a separate admission component around request handling and queue publication.

## F9 — Queued events and metrics can appear after running/succeeded

Severity: **HIGH**

Async handlers perform:

```text
save queued job
enqueue visible item
increment queued metric
publish queued event
return 202
```

A worker can receive the item immediately after enqueue and publish `running` or even `succeeded` before the handler publishes `queued`.

Possible observed event order:

```text
running
succeeded
queued
```

The response body also uses the local queued object even if persistence already reached a terminal state. Returning an initially queued representation is defensible, but transition event order must not go backwards.

Required direction:

- make initial job publication part of the producer operation before worker visibility;
- or have the event stream derive ordered transitions from a durable sequence/version;
- define per-job monotonic transition sequence numbers.

## F10 — Job transitions consume work without guaranteed terminality

Severity: **HIGH**

If `SetRunning` fails, the item is consumed and the worker continues. The persisted job may remain queued forever.

If user creation succeeds but `SetSucceeded` fails, the user exists while the job can remain running forever.

There is no retry, transaction, outbox, idempotency key or repair queue for these transition failures.

This crosses into Audit 11 resilience, but it is also a lifecycle correctness issue: accepted work is not guaranteed to reach one terminal state.

## F11 — Async enqueue rollback uses the request context

Severity: **HIGH**

When enqueue fails, the handler deletes the previously persisted job using `r.Context()`.

If the request context is already canceled, SQL cleanup can fail and leave an orphan `queued` job that was never enqueued.

Required direction:

- bounded detached cleanup context;
- or persist a rejected/aborted terminal state instead of deletion;
- ideally one transactional/outbox-style admission boundary for durable implementations.

This is analyzed further in Audit 11.

## F12 — gRPC Runtime has no single-start/single-stop ownership

Severity: **HIGH**

`Runtime.Start` has no guard and returns no completion/error channel. Multiple calls can launch multiple goroutines calling `Serve` on the same server/listener.

`Shutdown` can also be called repeatedly and creates a new `GracefulStop` goroutine each time.

The atomics report state but do not serialize the lifecycle operation itself.

Required direction:

- explicit runtime state machine;
- `Start` returns/owns one `done` and fatal-error result;
- shutdown is one idempotent operation;
- supervisor consumes the fatal error instead of converting it only into context cancellation.

## F13 — APIServer and component lifecycle remain path-dependent

Severity: **HIGH**

Audit 08 already established asymmetric cleanup. Concurrency review reinforces that ownership is split across:

- `main` defers;
- `APIServer.Run`;
- gRPC internal goroutine;
- worker-pool defer;
- event Hub close.

A unified supervisor is necessary not only for telemetry flush but also to prevent concurrent or repeated component shutdown sequences.

## F14 — Testkit wires two different event hubs

Severity: **HIGH for test validity**

`testkit.newApp` creates:

- one Hub for v1/v2 producers;
- another Hub for `JobHandler` SSE subscriptions and `App.EventHub`.

Therefore full-stack tests created through testkit do not exercise the real producer-to-SSE-consumer path. Producer events can be published to one Hub while the endpoint listens to another.

This can hide event-ordering, drops, close and concurrency defects.

## F15 — Global authentication first-increment update is not atomic

Severity: **MEDIUM**

`incAuthN` performs:

```text
Get
if nil: create Int and Set
Add
```

Two goroutines observing a new key can each create an `expvar.Int`; one map value can replace the other after it was incremented, losing an increment.

Use `expvar.Map.Add(key, 1)` or replace globals with an injected metrics registry.

## F16 — Bulkhead gauge can transiently disagree with semaphore occupancy

Severity: **LOW/MEDIUM**

The handler registers two defers:

```text
defer inFlight.Add(-1)
defer release semaphore
```

Defers run in reverse order, so the semaphore slot is released before the global gauge is decremented. Another request can acquire and increment during that window, producing a transient gauge above the actual configured per-instance limit.

The global gauge also aggregates every Bulkhead instance, so it cannot represent one semaphore accurately.

## F17 — Stream Hub synchronization is currently sound for tested send/close paths

Status: **POSITIVE FINDING**

`Publish`, `Subscribe`, unsubscribe mutation and `Close` coordinate through the same mutex. `Close` sets the atomic closed flag before taking the mutex, then closes channels while holding the mutex. `Publish` does not send without the mutex.

The audit stress test found no data race or send-on-closed panic.

Remaining design limitations are not memory races:

- one global mutex limits fan-out scalability;
- unsubscribe removes but does not close the subscription channel;
- `Event.Data` is `any`, so future producers must treat payloads as immutable or copy them before publishing;
- metrics remain process-global across Hub instances.

## F18 — Memory repositories have good locking and copy discipline

Status: **POSITIVE FINDING**

Memory user/job maps are protected by mutexes. Read methods return copies. Job cloning also copies nested result/problem data and invalid-parameter slices.

The email existence check and save are separate service calls, but `MemoryUserRepository.Save` rechecks duplicates under its write lock. PostgreSQL relies on its unique constraint and error mapping. This avoids duplicate email creation from the check-then-save race.

## F19 — Full concurrency proof is still unavailable

Severity: **PROCESS GAP**

Full `go test -race ./...` could not be executed in the sandbox because the exact Go 1.25.8 toolchain and external modules are unavailable.

The final verification must run on Linux/WSL or CI with:

```text
go test -race ./...
```

and include new tests for lifecycle generations, repeated stop, queue cancellation, state-transition CAS, SSE producer/consumer wiring, fatal component errors and shutdown under load.

## Required remediation order

Before adding a durable broker or high-load worker features:

1. Define the job transition state machine and enforce CAS transitions in both repositories.
2. Redesign WorkerPool lifecycle as one owned generation with a stable `done` result.
3. Make Stop genuinely idempotent and prohibit restart until the prior generation is fully stopped.
4. Pass generation context into worker loops; do not read mutable `wp.ctx` fields from old workers.
5. Remove one-waiter-goroutine-per-Stop.
6. Define queue cancellation/admission linearization and fix canceled-before-call behavior.
7. Guarantee monotonic producer/worker event ordering.
8. Add bounded repair/compensation for failed transition writes.
9. Unify application supervision across HTTP, HTTPS, gRPC, workers, Hub and future telemetry.
10. Fix testkit to use one event Hub.
11. Replace global metrics with DI-owned registries.
12. Run full Go 1.25.8 race and shutdown/load suites.

## Target lifecycle shape

Conceptually:

```text
ApplicationSupervisor
  owns app context and fatal-error channel

WorkerGeneration
  immutable ctx/cancel
  local WaitGroup
  stable done channel
  explicit state
  stopOnce
  stored stop result

Job transition
  queued --CAS--> running --CAS--> succeeded|failed
  terminal states cannot transition

Queue admission
  explicit accepting state
  documented linearization point
  canceled-before-call always rejected
```

## Audit status

- Worker-pool lifecycle: **BROKEN**
- Queue cancellation semantics: **BROKEN/PARTIAL**
- Job state machine: **BROKEN**
- Stream Hub memory synchronization: **PRESENT/PASS for tested paths**
- Repository memory locking: **PRESENT**
- gRPC lifecycle ownership: **PARTIAL/BROKEN**
- Application supervision: **MISSING**
- Race-test coverage: **PARTIAL/INSUFFICIENT**

