# Audit 04 — Metrics Audit

## Scope

This pass audits the uploaded working tree's application and runtime metrics. It covers metric inventory, semantic correctness, update sites, concurrency, cardinality, process-global state, tests and readiness for Prometheus/OpenTelemetry. It does not implement metric changes or choose the final backend.

## Method and environment

Reviewed:

- `internal/metrics/http.go`
- `internal/metrics/outbound.go`
- `internal/middleware/metrics.go`
- `internal/middleware/status_recorder.go`
- `internal/middleware/middleware.go`
- `internal/middleware/authorization.go`
- `internal/middleware/rbac.go`
- `internal/middleware/cors.go`
- `internal/middleware/rate_limiter.go`
- `internal/middleware/bulkhead.go`
- `internal/queue/queue.go`
- `internal/workerpool/workerpool.go`
- `internal/stream/stream.go`
- `internal/db/stats.go`
- `internal/outbound/profile_instrumentation.go`
- `internal/outbound/profile_retry.go`
- `internal/interceptors/interceptors.go`
- metric-related tests, `cmd/api/main.go`, `internal/testkit/testkit.go` and README metric documentation.

The target remains Go `1.25.8`. In the disposable Go `1.23.2` compatibility copy, the stdlib-only packages `internal/metrics`, `internal/stream` and `internal/queue` compile successfully. A temporary audit-only test also reproduced the first-queue `queue_depth` binding defect described below. The temporary test was removed after execution. Full application tests remain unavailable because external modules are not cached and the sandbox has no network.

## Current metric inventory

| Metric | Current type | Dimensions encoded | Meaning |
|---|---|---|---|
| `http_in_flight` | `expvar.Int` | none | currently executing HTTP requests, including long-lived streams |
| `http_requests_total` | `expvar.Map` | method, mux pattern, exact status | completed non-debug/non-SSE requests |
| `http_errors_total` | `expvar.Map` | method, mux pattern | completed HTTP responses with recorded status `>=500` |
| `http_latency_ns_sum/count` | `expvar.Map` | method, mux pattern | total duration and count for average latency |
| `jobs_total` | `expvar.Map` | transition status | counts observed queued/running/succeeded/failed transitions |
| `job_processing_latency_ns_sum/count` | `expvar.Int` | none | worker processing duration after running transition |
| `queue_depth` | `expvar.Func` | none | `len(ch)` of the first queue constructed in the process |
| `queue_rejections_total` | `expvar.Int` | none | queue-full fast-fail rejections only |
| `rate_limited_total` | `expvar.Int` | none | rejected limiter reservations |
| `bulkhead_in_flight` | `expvar.Func` over global atomic | none | aggregate in-flight across all bulkhead instances |
| `bulkhead_rejections_total` | `expvar.Int` | none | aggregate bulkhead rejections |
| `outbound_requests_total` | `expvar.Map` | host, stable route, status class | physical outbound attempts |
| `outbound_latency_ns_sum/count` | `expvar.Map` | host, stable route | physical attempt duration |
| `outbound_errors_total` | `expvar.Map` | normalized error kind | physical failed attempts |
| `authn_failures_total` | `expvar.Map` | normalized authn kind | authentication failures |
| `authz_forbidden_total` | `expvar.Int` | none | all authorization denials |
| `cors_preflight_total` | `expvar.Int` | none | accepted preflight requests |
| `cors_denied_total` | `expvar.Int` | none | all CORS denial reasons |
| `sse_subscribers` | `expvar.Map` | constant key `subscribers` | aggregate current subscribers across hubs |
| `sse_events_total` | `expvar.Map` | constant key `eventsTotal` | publish calls, whether or not a subscriber receives them |
| `sse_drops_total` | `expvar.Map` | constant key `dropsTotal` | per-subscriber non-blocking delivery drops |
| `db_pool` | `expvar.Map` of funcs | fixed DB stat names | live `database/sql.DBStats` values |
| `/debug/runtime` | JSON snapshot | selected runtime metrics | ad hoc runtime/GC diagnostic snapshot, not a metrics exporter |

Built-in `expvar` also exposes `cmdline` and `memstats` on `/debug/vars`.

## Strengths already present

### M-S1 — Stable HTTP route identity is used

`internal/middleware/metrics.go:29-45` reads `r.Pattern` after `ServeMux` execution and strips the method prefix before recording. This avoids user IDs and concrete path parameters in HTTP metric keys.

### M-S2 — Outbound route identity is normalized

`internal/outbound/profile_instrumentation.go` records the stable route `GET /profiles/{user_id}` instead of the concrete path. Current host cardinality is bounded by process configuration.

### M-S3 — Counter updates generally use concurrency-safe primitives

Most counters use `expvar.Int.Add`, `expvar.Map.Add`, or atomic values. The stream hub additionally protects per-hub state with a mutex.

### M-S4 — Units are explicit

Latency sums are named with `_ns_`, DB wait duration uses `_ns`, and the runtime endpoint preserves metric units in runtime metric names.

### M-S5 — Debug exposure is gated by the existing debug/admin policy

Metrics are not exposed as an unauthenticated public endpoint in the current routing model. `/debug/*` is conditional and admin protected.

### M-S6 — Runtime and DB diagnostic data already exists

The runtime snapshot and `database/sql.DBStats` provide a useful starting point for capacity experiments, even though they are not yet connected to a standard exporter.

## Findings

### M1 — `queue_depth` is permanently bound to the first `Queue` instance

Severity: **HIGH**

Evidence:

- `internal/queue/queue.go:11-14` uses package-global `sync.Once`.
- `internal/queue/queue.go:48-57` publishes a closure capturing the local `q` from the first `New` call.

Every later queue shares the rejection counter but cannot replace the depth function. This is already relevant because tests create many queues in one process. It would also be wrong for any future multi-tenant, multi-priority or multiple-server process.

Dynamic reproduction in the disposable Go 1.23.2 copy:

```text
first queue depth  = 1
second queue depth = 2
published queue_depth = 1
```

The temporary reproduction test passed, proving the metric observes the first queue rather than the active/test queue.

### M2 — Process-global metric singletons break isolation and blur instance ownership

Severity: **HIGH for tests and future composition**

Examples:

- `metrics.DefaultHTTP()` and `DefaultOutbound()` are package singletons.
- queue, rate limiter, bulkhead, authn, authz and CORS metrics are package globals.
- SSE metrics aggregate every `Hub` instance in the process.

Consequences:

- tests must read before/after deltas and cannot reset state;
- parallel tests can interfere;
- multiple application instances inside one process cannot be distinguished;
- route-specific limiters/bulkheads cannot be observed independently;
- an injected component can accidentally report into another test/application's registry.

A registry/observer owned by the composition root is required before production-grade exporters are added.

### M3 — Recorded HTTP status can disagree with the wire status

Severity: **HIGH**

This is the same defect identified in Audit 03, now confirmed as a metric correctness issue.

`internal/middleware/middleware.go:19-22` overwrites the recorder status on every `WriteHeader`, while `net/http` commits only the first status. `http_requests_total`, `http_errors_total` and latency series may therefore be attributed to the wrong status.

This must be fixed before metric baselines, alerts or SLO calculations are trusted.

### M4 — Latency metrics cannot describe tail latency

Severity: **HIGH for SLO/load analysis**

HTTP, outbound and job processing expose only sum and count. They allow an average, but no distribution, buckets, p50, p95 or p99.

An average can remain healthy while a small but important fraction of requests is extremely slow. The planned load laboratory and SLO work require histograms with deliberately bounded labels and reviewed buckets.

### M5 — HTTP metric inclusion policy is heuristic and inconsistent for streaming

Severity: **MEDIUM**

`internal/middleware/metrics.go:35-37` skips:

- any raw path under `/debug/`;
- exactly `/api/v1/jobs/{id}/events`.

Issues:

- a v2 or renamed stream route would silently enter ordinary latency metrics;
- matching is split between raw path and exact pattern strings;
- `http_in_flight` is incremented before the skip and therefore includes active SSE streams, while duration/count do not;
- there is no explicit metric family distinguishing ordinary requests from streams.

Including streams in total in-flight may be useful, but it must be an explicit semantic choice and normally needs a separate `stream_connections` gauge.

### M6 — HTTP metrics omit important service signals

Severity: **MEDIUM**

Current HTTP metrics do not expose:

- time to first byte;
- request/response body size distributions;
- active/accepted/closed connections;
- panic/recovery count;
- cancellation/disconnect count;
- timeout class;
- protocol (`HTTP/1.1` vs `HTTP/2`);
- server identity/resource attributes.

Not all of these need labels on every metric, but the final observability model must select enough to explain saturation and user-visible latency.

### M7 — `jobs_total` is a transition-event counter with incomplete terminal coverage

Severity: **HIGH for business/job observability**

`internal/metrics/http.go:23-30` increments status-named counters, but the semantics are not current job counts and not strictly terminal outcomes. They are transition events.

Coverage gaps in `internal/workerpool/workerpool.go`:

- shutdown `FailActiveOnShutdown` repairs jobs but does not increment failed transitions;
- a `SetRunning` storage error produces no terminal/failed metric;
- a `SetSucceeded` storage error produces no failed metric and no processing observation;
- processing duration begins after the running transition and does not include queue wait;
- there is no enqueue-to-start duration or total job end-to-end duration.

The metric should be renamed/redefined and accompanied by explicit gauges/counters such as current jobs by state, transitions by outcome, queue wait histogram and total processing histogram.

### M8 — Queue observability is too coarse even aside from the first-instance defect

Severity: **HIGH for planned high-load experiments**

Only current depth and queue-full rejection count are available. Missing signals include:

- capacity and utilization ratio;
- accepted/enqueued total;
- rejection reason (`full`, `stopped`, `context canceled`);
- oldest queued item age;
- enqueue-to-dequeue wait distribution;
- dequeue/processing rate;
- queue stopped/accepting state.

Without queue age and wait duration, a queue can look non-full while still violating latency objectives.

### M9 — Outbound metrics describe physical attempts, not logical operations

Severity: **HIGH for retry and dependency analysis**

Wiring is:

```text
RetryingProfileClient
  -> InstrumentedProfileClient
    -> ClientImpl
```

Therefore every attempt increments outbound request/error/latency metrics. Missing:

- logical operation total and final outcome;
- attempt number/count;
- retries total;
- backoff duration;
- exhausted retry budget;
- remaining deadline;
- in-flight outbound operations;
- transport pool state.

A retry storm can inflate apparent request volume and errors without showing how many user operations ultimately succeeded or failed.

### M10 — Authentication's first increment for a new kind can be lost under concurrency

Severity: **MEDIUM/HIGH for correctness**

`internal/middleware/authorization.go` performs a check-then-create sequence:

```text
Get(key) -> if nil: create Int -> Set(key) -> Add(1)
```

`expvar.Map` operations are individually synchronized, but this multi-step sequence is not atomic. Two first-time failures of the same kind can create separate `expvar.Int` values; one can be replaced in the map after the other goroutine increments its detached value.

The direct `expvar.Map.Add(key, 1)` pattern used elsewhere avoids this race.

### M11 — Security and admission-control metrics lack actionable dimensions

Severity: **MEDIUM**

Current coarse totals:

- `authz_forbidden_total` has no normalized reason or route;
- `cors_denied_total` has no origin-policy reason (`origin`, `method`, `headers`, `credentials`);
- `rate_limited_total` has no route/policy and no accepted count;
- bulkhead metrics aggregate all instances and have no route/policy.

Dimensions must remain bounded, but stable reason/policy/route labels are needed for diagnosis and alerting.

### M12 — SSE metric names and semantics are weak

Severity: **MEDIUM**

`internal/stream/stream.go:53-58` uses `expvar.Map` with constant keys such as `subscribers`, `eventsTotal`, `dropsTotal`. This adds no useful dimension and complicates later conversion.

Semantic gaps:

- `events_total` counts publish calls even with zero subscribers;
- it does not count successful deliveries separately;
- drops are per subscriber, not per published event;
- no connection duration;
- no disconnect/write-timeout/flush-error counters;
- no heartbeat count;
- no bounded event-type dimension.

### M13 — gRPC has logging but no metrics

Severity: **HIGH**

`internal/interceptors/interceptors.go` records logs only. There are no metrics for:

- calls by full method and status code;
- duration histogram;
- in-flight RPCs;
- cancellations/deadlines;
- request/response message sizes;
- server starts/stops.

This leaves a major protocol surface invisible to dashboards and SLOs.

### M14 — Database metrics stop at pool snapshots

Severity: **MEDIUM**

`internal/db/stats.go` exposes useful pool gauges/cumulative values, but there are no repository/query metrics for:

- query duration;
- query/transaction errors;
- timeout/cancellation;
- affected rows;
- transaction commit/rollback;
- operation identity.

Additionally, each `db_pool` field calls `db.Stats()` independently, so one `/debug/vars` serialization can contain values sampled at slightly different instants. This is a minor diagnostic consistency issue, not a data race.

### M15 — The runtime endpoint is diagnostic JSON, not an operational metrics pipeline

Severity: **MEDIUM**

`/debug/runtime` is useful for manual snapshots but cannot substitute for:

- Prometheus/OpenTelemetry metric export;
- scrape timestamps and resource labels;
- histograms suitable for dashboards;
- retention and alerting;
- standard process/runtime collectors.

It should remain a debug tool while runtime metrics are exported through the chosen production-like pipeline.

### M16 — Metric test coverage is narrow and process-global state makes it fragile

Severity: **HIGH for regression confidence**

Direct assertions exist mainly for:

- HTTP requests/errors/latency;
- async queued/running/succeeded and processing count.

No direct metric assertions were found for:

- outbound metrics;
- queue depth/rejection semantics;
- rate limiter/bulkhead counters and gauges;
- authn/authz/CORS counters;
- SSE subscribers/events/drops;
- DB pool metrics;
- future gRPC metrics.

Tests use global expvar state and delta comparisons. They are intentionally non-parallel but remain order-sensitive and cannot verify clean per-app initialization. The first-queue bug is a direct example of this weakness.

### M17 — `testkit` constructs disconnected stream hubs while metrics aggregate them globally

Severity: **MEDIUM, broader than metrics**

`internal/testkit/testkit.go:160-164` creates one hub for user handlers and a second hub for the job events handler/application handle. This means a queued event published by the user handler may not reach the hub used by the SSE endpoint. Since SSE expvar values are global, aggregate counters can still increase and hide the wiring mismatch.

This should be addressed during lifecycle/testing audits, but it is recorded here because global metrics can make the faulty topology look healthy.

### M18 — No standard exporter, resource identity or cardinality policy exists

Severity: **HIGH**

There is no Prometheus or OpenTelemetry metric exporter, and no defined resource attributes such as service name, version, environment or instance ID.

Current cardinality is mostly bounded because HTTP uses `r.Pattern` and outbound uses a stable route. However, there is no written policy prohibiting future labels such as raw URL, user ID, job ID, request ID, trace ID or error text.

A cardinality policy must be established before dashboards and load tests generate meaningful volume.

## Current metric maturity by subsystem

| Subsystem | Status | Summary |
|---|---|---|
| HTTP RED baseline | PARTIAL | rate and average duration available; status recorder and tail latency gaps |
| Queue/jobs | BROKEN/PARTIAL | first-instance depth defect; incomplete lifecycle and wait metrics |
| Outbound dependency | PARTIAL | good normalized attempt metrics; no logical retry operation view |
| Security/CORS | PARTIAL | counters exist but lack bounded reasons/policies; authn init race |
| Bulkhead/rate limiter | PARTIAL | coarse global totals; no per-policy saturation view |
| SSE | PARTIAL | current subscribers/publishes/drops only; weak naming and semantics |
| gRPC | MISSING | no metric instrumentation |
| Database | PARTIAL | pool snapshot only |
| Runtime | PRESENT AS DEBUG | manual snapshot/pprof, not an exporter |
| Export pipeline | MISSING | no Prometheus/OTel Collector/Grafana pipeline |

## Required remediation direction

Do not directly convert every current expvar key into a Prometheus metric. First define and inject a process-owned telemetry registry/observer model.

Recommended order after the audit phase:

1. Fix `statusRecorder` first-status semantics.
2. Replace component package-global metric ownership with injected observers/registry.
3. Fix queue metric binding and define queue/job semantics.
4. Define naming, units, resource attributes and bounded-label policy.
5. Add histograms for HTTP, outbound, queue wait, job duration and gRPC.
6. Add logical retry-operation and admission-control metrics.
7. Add gRPC and DB operation instrumentation.
8. Add Prometheus or OTel metric export while retaining expvar only as optional debug compatibility.
9. Add deterministic metric tests using an isolated in-memory/manual reader rather than the global expvar registry.

## Audit result

Metrics capability status: **PARTIAL with correctness defects**.

The project has a useful educational `expvar` baseline and several good low-cardinality decisions. It is not yet safe to use these metrics as the source for production-like SLOs or capacity conclusions because:

- at least one gauge observes the wrong component instance;
- recorded HTTP status can be wrong;
- metric state is process-global and test-contaminated;
- tail latency is unavailable;
- queue, retry, gRPC and job lifecycle semantics are incomplete;
- no standard export pipeline exists.

These findings directly inform Audit 05 (tracing/propagation) and Audit 06 (telemetry pipeline architecture).
