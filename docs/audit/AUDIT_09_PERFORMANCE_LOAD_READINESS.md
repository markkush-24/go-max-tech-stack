# Audit 09 — Performance and Load Readiness

## Scope

This pass evaluates whether the uploaded `pet-study` working tree is ready for repeatable performance baselines, saturation tests, profiling and future high-load laboratory work.

The pass covers:

- existing benchmarks and load tooling;
- HTTP server and middleware limits;
- worker/queue capacity;
- PostgreSQL pool and repository query shape;
- outbound HTTP connection behavior and retry amplification;
- SSE fan-out and long-lived connection costs;
- gRPC capacity controls;
- runtime/GC and pprof readiness;
- reproducibility requirements for future performance experiments.

The pass does **not** optimize code and does not establish production capacity numbers. The target Go toolchain remains Go 1.25.8. Small stdlib-only microbenchmarks were executed in the disposable Go 1.23.2 compatibility copy and are treated only as directional evidence.

## Reviewed evidence

Primary project evidence:

- `cmd/api/main.go`
- `internal/config/config.go`
- `internal/api/server.go`
- `internal/router/router.go`
- `internal/middleware/metrics.go`
- `internal/middleware/status_recorder.go`
- `internal/middleware/rate_limiter.go`
- `internal/middleware/bulkhead.go`
- `internal/queue/queue.go`
- `internal/workerpool/workerpool.go`
- `internal/stream/stream.go`
- `internal/routes/users_handler.go`
- `internal/routes/job_handler.go`
- `internal/routes/user_profile_handler.go`
- `internal/outbound/httpclient/client.go`
- `internal/outbound/profile.go`
- `internal/outbound/profile_retry.go`
- `internal/store/userrepo/memory_user_repo.go`
- `internal/store/userrepo/sqlx.go`
- `internal/store/jobrepo/sqlx.go`
- `internal/db/open.go`
- `internal/db/stats.go`
- `internal/runtimeinfo/runtimeinfo.go`
- `internal/transport/grpcserver/runtime.go`
- `internal/transport/grpcclient/grpcclient.go`
- `internal/interceptors/interceptors.go`
- `internal/testkit/testkit.go`
- `.github/workflows/ci.yml`
- `scripts/*.ps1`
- `docker-compose.yml`
- `README.md`

Repository-wide searches were also performed for `Benchmark`, `Fuzz`, pprof, runtime trace and common load-generator references.

## Executive result

### Current state

The service has a useful diagnostic foundation:

- guarded `net/http/pprof` endpoints;
- CPU, heap, goroutine and execution-trace access through the debug subtree;
- a custom runtime/GC snapshot;
- configurable HTTP, DB, worker, queue and outbound pool values;
- bounded queue and subscriber buffers;
- PostgreSQL `DBStats` exposure;
- HTTP/HTTPS, unary gRPC, SSE, async jobs and outbound paths that can form a meaningful load laboratory.

However, performance readiness is currently **PARTIAL/BROKEN** because:

1. there are no committed benchmarks, fuzz targets or load scenarios;
2. the default configuration intentionally throttles the service to `5 RPS` and one concurrent API operation;
3. the global bulkhead wraps the SSE endpoint, so one long-lived stream can occupy the only default slot and reject all other API traffic;
4. all API routes share one limiter and one bulkhead, creating cross-route noisy-neighbor behavior;
5. list endpoints and repository queries are unbounded and have no pagination;
6. worker count, queue size and PostgreSQL connection count are fixed independently with no capacity model;
7. outbound connections are effectively unbounded per host when higher API concurrency is enabled;
8. gRPC and SSE lack capacity controls and RED telemetry;
9. current metrics cannot measure tail latency or queue age;
10. no reproducible target-environment baseline exists.

The project is ready to **build** a performance laboratory, but not yet ready to make trustworthy throughput or SLO claims.

## Positive findings

### 1. Profiling endpoints already exist

`internal/router/debug.go` mounts `net/http/pprof`, including CPU profiles and execution traces, under `/debug/pprof/*`. The debug subtree is conditional and protected by the existing admin policy.

The README documents:

- heap profile download;
- 30-second CPU profiling;
- goroutine profile inspection;
- runtime/GC environment experiments.

This means the project does not need a new profiling server before load work begins.

### 2. Runtime and GC snapshot is useful for controlled experiments

`internal/runtimeinfo/runtimeinfo.go` records:

- heap and total runtime memory;
- allocation/free counters;
- GC cycles and pause totals;
- goroutine count;
- `GOMAXPROCS` and CPU count;
- selected `runtime/metrics` values.

This is useful for pre-test/post-test snapshots and validating `GOGC`/`GOMEMLIMIT` experiments.

### 3. Resource limits are configurable

The project exposes configuration for:

- HTTP timeouts and max header bytes;
- DB connection pool size/lifetime;
- worker count and queue capacity;
- API rate and burst;
- bulkhead concurrency;
- outbound idle and active connection behavior;
- profile timeout and retry count;
- SSE heartbeat, subscriber buffer and write timeout.

This gives future load scenarios explicit experimental controls instead of requiring code edits.

### 4. Queue and SSE fan-out are bounded at the immediate buffer level

The queue is a bounded channel with fast-fail overflow. SSE subscribers use bounded channels and `Publish` performs non-blocking delivery, incrementing a drop counter rather than blocking the worker.

These are important backpressure primitives, even though the surrounding capacity model is incomplete.

### 5. SQL-backed operations use contexts and query timeouts

The SQLX repositories derive a bounded context for each query. This prevents a single database operation from waiting without limit during pool or database degradation.

### 6. Outbound HTTP transport is reused

A single cloned `http.Transport` and `http.Client` are created in the composition root and reused. Idle connections are explicitly closed at application shutdown. The project does not create a transport per request.

## Dynamic microbenchmark evidence

Temporary audit-only benchmark files were created and removed in the disposable Go 1.23.2 compatibility copy. They did not modify the audited working tree.

Environment:

```text
linux/amd64
Go 1.23.2 compatibility copy
Intel Xeon Platinum 8370C
```

Representative results across three runs:

| Primitive | Result | Allocations |
|---|---:|---:|
| Queue enqueue + receive round trip | ~101–102 ns/op | 0 B/op, 0 allocs/op |
| Hub publish, no subscribers | ~56 ns/op | 0 B/op, 0 allocs/op |
| Hub publish, one subscriber + receive | ~157 ns/op | 0 B/op, 0 allocs/op |
| Hub publish, 100 subscribers + receive | ~7.2–7.3 µs/op | 0 B/op, 0 allocs/op |
| Runtime snapshot | ~82–87 µs/op | ~6 KB/op, 27 allocs/op |

Interpretation:

- the basic queue/channel operation is inexpensive in isolation;
- hub publish cost grows approximately with subscriber count, as expected from its loop;
- the runtime snapshot is suitable for occasional diagnostics, not high-frequency scraping;
- these numbers exclude HTTP, JWT, JSON/Protobuf, database, logging, expvar, scheduling, TLS and network costs;
- they are **not** a Go 1.25.8 capacity baseline and must not be used as SLO thresholds.

## Findings

## F1 — No persistent benchmark or load-test suite exists

Severity: **HIGH / process gap**

Repository search found no committed `BenchmarkXxx` functions and no load-generator configuration for sustained HTTP, SSE or gRPC traffic.

The CI workflow runs:

- unit/integration tests;
- vet;
- race;
- staticcheck;
- vulnerability scanning.

It does not run:

- benchmarks;
- allocation regression checks;
- load smoke tests;
- performance artifact collection;
- pprof comparisons.

Without a versioned harness, results cannot be compared across code changes.

## F2 — Current defaults are not a high-load profile

Severity: **HIGH / interpretation risk**

`defaultConfig` currently sets:

```text
RATE_LIMIT_RPS=5
RATE_LIMIT_BURST=10
BULKHEAD_MAX_PARALLEL=1
WORKERS_COUNT=10
QUEUE_SIZE=10
DB_MAX_OPEN_CONNS=10
DB_MAX_IDLE_CONNS=10
```

These values are reasonable for demonstrating overload behavior, but they cannot serve as a throughput baseline.

A test that forgets to override them will mostly measure intentional rejection rather than service capacity.

The project needs named configuration profiles, for example:

```text
functional-dev
memory-baseline
postgres-baseline
full-observability
saturation
failure-injection
```

Every recorded result must state the exact profile.

## F3 — SSE occupies the shared bulkhead for the full stream lifetime

Severity: **CRITICAL for current defaults**

Every API route is wrapped by the same `wrap` function in `internal/router/router.go:24-31`. The SSE route `/api/v1/jobs/{id}/events` also uses that wrapper (`internal/router/router.go:82-87`).

The bulkhead acquires a semaphore slot before calling the handler and releases it only when the handler returns (`internal/middleware/bulkhead.go:34-45`). An SSE handler may remain active for minutes or hours.

With the default `BULKHEAD_MAX_PARALLEL=1`:

```text
one active SSE connection
→ one occupied bulkhead slot
→ every other wrapped API request fast-fails with 503
```

Even with a larger limit, every long-lived SSE client permanently consumes one slot intended for ordinary request concurrency.

SSE needs a separate admission/concurrency policy from finite HTTP operations.

## F4 — One global bulkhead creates cross-route head-of-line and noisy-neighbor behavior

Severity: **HIGH**

The same bulkhead protects:

- cheap GET user;
- list users;
- synchronous create;
- async enqueue;
- outbound Profile calls;
- HTTP-to-gRPC bridge;
- Range export;
- SSE streams.

A slow upstream Profile request or long SSE connection can reject unrelated cheap reads. This does not provide useful isolation between dependencies or workload classes.

Future bulkheads should be scoped by constrained resource, such as:

```text
profile_upstream
postgres_write
async_admission
sse_connections
grpc_bridge
```

A global emergency cap can still exist separately if needed.

## F5 — One global rate limiter mixes different traffic classes

Severity: **MEDIUM/HIGH**

All wrapped routes share one token bucket. Health and debug are outside it, but normal reads, writes, SSE handshakes and gRPC bridge calls compete for the same tokens.

This makes it difficult to:

- reserve capacity for reads;
- separately protect expensive writes;
- limit streams by connection count;
- apply per-principal or per-tenant policy;
- distinguish expected throttling from system saturation.

For laboratory work, rate limiting must be either disabled in a capacity profile or configured per workload class.

## F6 — The user list path is unbounded

Severity: **HIGH for data growth**

The SQL query in `internal/store/userrepo/sqlx.go:16-20` selects all users with no `LIMIT` or pagination. The handler then allocates a second DTO slice and serializes the complete result (`internal/routes/users_handler.go:188-203`).

The memory repository also copies every stored user into a new slice while holding an `RLock` (`internal/store/userrepo/memory_user_repo.go:35-44`).

As data grows, this path increases:

- database read duration;
- response size;
- heap allocations;
- GC pressure;
- lock hold time in the memory backend;
- tail latency and bandwidth.

Cursor or keyset pagination and response-size limits are prerequisites for meaningful high-load tests with large datasets.

## F7 — Memory repository email uniqueness is O(n)

Severity: **MEDIUM / test-backend scaling**

`MemoryUserRepository.Save` and `ExistsByEmail` scan every user and call `strings.EqualFold` (`internal/store/userrepo/memory_user_repo.go:59-66`, `98-107`).

This backend is not the production persistence path, but a large in-memory load baseline would increasingly measure a linear duplicate-email scan rather than HTTP/service overhead.

An auxiliary normalized-email index would make the memory baseline more representative and stable.

## F8 — Worker, queue and DB pool sizing have no explicit capacity relationship

Severity: **HIGH**

Defaults are:

```text
workers = 10
queue capacity = 10
DB max open connections = 10
```

Workers and synchronous HTTP requests share the same DB pool. Each async job can execute several persistence operations:

```text
SetRunning
CreateUser
SetSucceeded or SetFailed
```

Under mixed synchronous and async traffic, workers can occupy most DB connections while HTTP handlers wait, or HTTP traffic can delay job transitions.

The project currently has no documented formula or experiment relating:

- worker concurrency;
- DB pool size;
- average queries per job;
- query service time;
- queue capacity;
- accepted request rate;
- target job completion time.

This relationship must be measured, not guessed.

## F9 — Queue observability cannot support capacity tuning

Severity: **HIGH**

Current queue telemetry exposes depth and full rejections, but not:

- configured capacity;
- utilization ratio;
- accepted rate;
- enqueue wait/decision reason;
- oldest item age;
- queue wait duration;
- dequeue throughput.

Depth alone cannot distinguish a healthy short burst from a stale queue with old work. Audit 04 and 07 already defined the missing metric contract.

## F10 — Outbound active connection count is unlimited per host

Severity: **HIGH after concurrency is raised**

The default outbound transport sets:

```text
MaxIdleConns=1000
MaxIdleConnsPerHost=1000
MaxConnsPerHost=0
```

`MaxConnsPerHost=0` leaves active connection count unlimited. The current global bulkhead of one masks this risk, but once API concurrency is increased, a slow Profile service can cause a large number of concurrent connections/dials.

The Profile dependency needs an explicit per-upstream concurrency budget, chosen together with `MaxConnsPerHost`, timeout and retry policy.

## F11 — Retry amplification is not bounded by a dependency-level concurrency policy

Severity: **HIGH when retries are enabled**

Retries are currently disabled by default (`MaxAttempts=1`), which avoids amplification. If attempts are increased, each logical Profile operation can produce multiple physical requests.

There is no:

- circuit breaker;
- retry budget;
- per-upstream bulkhead;
- global retry concurrency limit;
- logical-operation metric.

During a dependency slowdown, retries can increase load exactly when the dependency has reduced capacity.

The shared RNG mutex in `RetryingProfileClient` is likely a minor cost under normal traffic, but should be included in a high-retry benchmark rather than optimized speculatively.

## F12 — Outbound response draining is not byte-bounded

Severity: **MEDIUM/HIGH**

On non-200 and after JSON decode, the client drains the remainder using `io.Copy(io.Discard, resp.Body)` without a byte limit (`internal/outbound/profile.go:51-75`).

The request context and response-header timeout bound time, but a large or continuous upstream body can still consume bandwidth and keep a handler/connection busy until the operation budget expires.

The client needs a maximum response-body size and a documented keep-alive reuse policy for oversized/error bodies.

## F13 — SSE fan-out uses one global mutex across all job IDs

Severity: **MEDIUM/HIGH at large fan-out**

`Hub.Publish` holds one hub-wide mutex while iterating and attempting delivery to all subscribers for a job (`internal/stream/stream.go:125-141`).

Consequences:

- publish cost is linear in subscriber count;
- a publish for one job blocks subscribe/unsubscribe and publish operations for other job IDs;
- slow scheduler progress while holding the lock increases contention even though channel sends are non-blocking.

The isolated audit benchmark showed roughly 7.2–7.3 µs for publish plus receive across 100 subscribers on the audit machine. That is not itself alarming, but the linear/global-lock design must be tested at thousands of clients and multiple job IDs.

## F14 — Every SSE connection owns a goroutine, ticker and socket

Severity: **MEDIUM/HIGH**

Each SSE handler:

- occupies one HTTP handler goroutine;
- creates a `time.Ticker`;
- maintains a subscription channel;
- holds a socket until disconnect;
- performs JSON marshal and flush for each delivered event.

The same event payload is marshaled independently by every SSE handler rather than encoded once at publish time.

This may be acceptable for hundreds or low thousands of connections, but it requires explicit fan-out, slow-client, memory and file-descriptor experiments before any capacity claim.

## F15 — SSE is excluded from ordinary HTTP metrics but not from resource accounting

Severity: **MEDIUM**

`middleware.Metrics` excludes `/events` from HTTP request duration because a long-lived stream would distort ordinary request latency. That decision is reasonable.

However, the endpoint still consumes:

- HTTP in-flight count until the middleware returns;
- bulkhead capacity;
- goroutine/socket/ticker resources.

Dedicated SSE connection duration, active connection, write outcome and drop metrics are required so the cost is not invisible.

## F16 — HTTP access logging can become a load bottleneck

Severity: **MEDIUM/HIGH**

Every completed HTTP request synchronously formats and writes a text `slog` event to stderr. Outbound attempts and gRPC calls are also logged synchronously.

At high request rates, log formatting, serialization and destination backpressure can materially affect throughput and latency. The current system has no:

- access-log sampling;
- asynchronous log pipeline boundary;
- separate audit/error and high-volume access policies;
- benchmark with logging enabled/disabled.

The future baseline must report both modes and never hide errors merely to improve benchmark numbers.

## F17 — Current expvar metrics add per-request string work and shared-map contention

Severity: **MEDIUM**

Every ordinary HTTP request constructs keys such as:

```text
METHOD|/pattern|status
METHOD|/pattern
```

and updates process-global `expvar.Map` values. Under high concurrency this introduces allocation/string work and shared synchronization in the request path.

The future OTel implementation must also be benchmarked; replacing expvar does not automatically make instrumentation free.

A telemetry-on versus telemetry-off comparison is required.

## F18 — gRPC has no explicit load controls or measurements

Severity: **MEDIUM/HIGH**

The current gRPC service is unary-only and uses default server/client options apart from the logging/request-ID interceptor.

There is no explicit:

- concurrent-call budget;
- keepalive policy;
- message-size policy;
- TLS cost profile;
- gRPC RED telemetry;
- load harness;
- distinction between direct gRPC and HTTP-to-gRPC bridge overhead.

The service is simple, so tuning defaults in advance is unnecessary, but direct and bridge paths must be measured independently.

## F19 — The Range endpoint is not a meaningful large-object streaming workload

Severity: **MEDIUM / scope clarification**

`UsersHandler.Export` marshals one user into a byte slice, wraps it in `bytes.Reader`, then calls `http.ServeContent` (`internal/routes/users_handler.go:152-185`).

This correctly exercises Range semantics, but the object is tiny and fully materialized in memory. It does not model:

- large file I/O;
- object storage;
- zero-copy/file-backed serving;
- large concurrent Range requests;
- bandwidth and disk saturation.

A separate synthetic large-object fixture is needed if Range/high-bandwidth behavior becomes part of the laboratory.

## F20 — Block and mutex profiles are not explicitly enabled

Severity: **MEDIUM**

The pprof handlers are present, but the application does not configure runtime block or mutex profile rates.

CPU, heap, goroutine and execution trace are immediately useful. Meaningful contention investigation will need an explicit lab-only mechanism for enabling block and mutex profiling with documented overhead.

This should not be enabled continuously without measurement.

## F21 — No process/container/host metrics exist yet

Severity: **HIGH for capacity analysis**

The runtime snapshot observes Go memory and scheduler state, and DB stats expose connection-pool state. Missing from the current telemetry surface are stable time series for:

- process CPU;
- resident memory;
- file descriptors;
- network throughput/errors;
- container CPU throttling;
- container memory limits/OOM;
- host saturation;
- PostgreSQL server CPU/I/O/locks.

Grafana dashboards cannot explain a throughput plateau without these resource signals.

## F22 — PostgreSQL baseline environment is not reproducibly constrained

Severity: **HIGH / experiment reproducibility**

`docker-compose.yml` starts PostgreSQL but does not pin:

- CPU or memory limits;
- storage behavior;
- connection count at the server;
- database configuration;
- dataset seed/size;
- migration runner;
- metrics exporter.

A laptop run and CI runner run will therefore produce incomparable results unless environment metadata and resource constraints are recorded.

## F23 — No performance regression policy exists

Severity: **MEDIUM/HIGH**

There is no rule for deciding whether a change is a regression.

A future policy should distinguish:

- microbenchmark noise;
- statistically meaningful ns/op or alloc changes;
- full-stack throughput changes;
- p95/p99 changes;
- increased retry/queue/DB pressure;
- telemetry overhead.

Performance checks should initially report artifacts rather than hard-fail CI until stable baselines and variance are known.

## Required performance laboratory structure

## 1. Environment profiles

At minimum, keep four separate profiles:

### Profile A — in-memory application baseline

Purpose: isolate HTTP/router/middleware/JWT/serialization/queue costs.

- memory repositories;
- Profile upstream stub local and deterministic;
- no TLS for first baseline;
- rate limiter and shared bulkhead disabled or set far above offered load;
- observability measured both off and on.

### Profile B — PostgreSQL baseline

Purpose: measure DB pool, queries and data-size effects.

- fixed PostgreSQL image/config;
- fixed CPU/memory resources;
- deterministic migrations and seed sizes;
- DB server metrics enabled;
- explicit pool sizes.

### Profile C — full protocol and observability stack

Purpose: measure production-like overhead.

- TLS/HTTP2;
- gRPC;
- SSE;
- OTel Collector, Prometheus, Tempo, Loki and Grafana;
- tracing sampling recorded;
- JSON logging enabled.

### Profile D — failure and saturation

Purpose: expose operational failure modes.

- delayed/failing Profile upstream;
- constrained PostgreSQL;
- queue saturation;
- slow SSE clients;
- shutdown under load;
- Collector unavailable or slow.

## 2. Mandatory experiment metadata

Every result artifact must record:

```text
Git commit and dirty-tree hash/status
Go version and toolchain
OS/architecture
CPU count/model
GOMAXPROCS
memory/container limits
GOGC and GOMEMLIMIT
configuration profile and all overrides
storage backend
PostgreSQL version/config/dataset size
TLS and HTTP protocol
telemetry/logging/sampling state
route and payload
concurrency/arrival model
duration and warm-up
load-generator host
```

Without this metadata, results are anecdotal.

## 3. Core measurements

Each scenario should collect:

- offered and achieved RPS;
- successful RPS;
- p50/p90/p95/p99/max latency;
- status/error classification;
- CPU and memory;
- allocations and GC;
- goroutines;
- connection/file-descriptor counts;
- DB pool utilization/wait;
- queue depth, capacity, age and wait;
- retry amplification;
- SSE active connections/drops/write outcomes;
- gRPC code and latency;
- telemetry drops/export failures.

Average latency alone is insufficient.

## 4. Initial load scenario matrix

### L01 — single-request correctness baseline

- one virtual user;
- GET user, list, create, async create, Profile, gRPC bridge;
- verify status and response before increasing load.

### L02 — HTTP read throughput

- `GET /api/v1/users/{id}`;
- memory then PostgreSQL;
- JSON and Protobuf separately;
- ETag `200` and `304` separately;
- telemetry/logging A/B.

### L03 — synchronous writes

- unique payloads;
- observe DB connection usage, conflicts and p99;
- increase until throughput plateaus or error budget burns.

### L04 — async admission and worker saturation

- burst and constant arrival rates;
- vary workers, queue and DB pool independently;
- record queue wait, rejection and accepted-to-terminal latency.

### L05 — mixed read/write/async workload

- representative traffic mix;
- detect starvation and noisy-neighbor behavior.

### L06 — Profile upstream normal/slow/failing

- normal latency;
- latency near timeout;
- 5xx burst;
- connection reset;
- retries enabled;
- measure logical success, attempts and amplification.

### L07 — SSE connection scale and fan-out

- increasing concurrent clients;
- one job with many subscribers;
- many jobs with few subscribers;
- slow/non-reading clients;
- event bursts;
- disconnect storms.

### L08 — direct gRPC versus HTTP bridge

- direct unary GetJob;
- HTTP endpoint that calls loopback gRPC;
- compare overhead, code distribution and cancellation.

### L09 — HTTP/1.1 versus HTTPS/HTTP2

- same route and payload;
- controlled client connection counts;
- multiplexing and TLS handshake/warm-connection phases separated.

### L10 — PostgreSQL pool saturation

- vary `MaxOpenConns` and concurrency;
- observe wait count/duration and query latency;
- include mixed workers and HTTP requests.

### L11 — shutdown under load

- ordinary requests, active workers, gRPC calls and SSE clients;
- record graceful/forced outcomes, dropped work and total duration.

### L12 — runtime/GC experiments

- fixed workload;
- vary `GOGC`, `GOMEMLIMIT` and container memory;
- collect heap, CPU, GC pauses and throughput.

### L13 — telemetry overhead and outage

- telemetry disabled;
- traces/metrics/logs enabled;
- different sampling ratios;
- Collector unavailable;
- compare throughput, p99, memory and drops.

## 5. Profiling protocol

For a stable saturation plateau:

1. warm up connections, caches and JIT-independent runtime state;
2. run a sufficiently long steady window;
3. capture a CPU profile during the steady window;
4. capture heap before and after;
5. capture goroutine profile;
6. enable block/mutex profiles only for dedicated contention runs;
7. capture an execution trace for a short focused window;
8. save configuration and load-generator output with profiles;
9. compare profiles only under equivalent environment and workload.

## Implementation prerequisites before serious baselines

The minimum fixes before claiming a baseline are:

1. correct `statusRecorder` first-status behavior;
2. separate SSE from the finite-request bulkhead;
3. introduce per-workload limiter/bulkhead policies or an explicit disabled capacity profile;
4. add pagination to list endpoints;
5. implement trustworthy histogram/queue/job/outbound/gRPC/SSE metrics;
6. define deterministic dataset seed and environment profiles;
7. add versioned benchmark/load scripts;
8. add process/container/PostgreSQL observability;
9. run the final suite on Go 1.25.8 with cached dependencies and reproducible resources.

## Status summary

| Area | Status |
|---|---|
| pprof/runtime diagnostics | PRESENT/PARTIAL |
| committed microbenchmarks | MISSING |
| full-stack load harness | MISSING |
| reproducible capacity environment | MISSING |
| finite HTTP capacity isolation | BROKEN/PARTIAL |
| SSE capacity isolation | BROKEN |
| queue/worker boundedness | PRESENT, insufficiently measured |
| PostgreSQL pool controls | PRESENT, uncalibrated |
| outbound transport reuse | PRESENT |
| outbound concurrency protection | MISSING/PARTIAL |
| gRPC load controls/RED metrics | MISSING |
| tail-latency measurement | MISSING |
| process/container/DB-server metrics | MISSING |
| performance regression policy | MISSING |

## Decision carried forward

- Audit 10 must examine the concurrency correctness behind the shared bulkhead, hub lock, worker shutdown and global registries.
- Audit 11 must turn Profile slowdown, retry amplification, DB degradation and queue saturation into fault-handling scenarios.
- Audit 13 must verify benchmark/load entrypoints, CI artifacts and reproducibility.
- Final implementation planning must prioritize observability correctness and workload isolation before dashboards or aggressive tuning.
