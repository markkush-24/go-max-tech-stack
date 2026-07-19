# Audit 07 — SLI/SLO, Dashboards and Alerting

## Scope

This pass defines the reliability measurement and alerting contract for the uploaded `pet-study` working tree. It evaluates whether current telemetry can support SLI/SLO calculations, selects provisional laboratory objectives, defines required metric semantics, proposes Prometheus recording/alerting rules, and specifies Grafana dashboard and runbook structure.

This pass does **not** implement OpenTelemetry, Prometheus, Grafana, Tempo, Loki, dashboards or alerts. It also does not claim that the proposed numerical targets are production commitments. They are initial laboratory objectives to be calibrated after a repeatable baseline load test.

## Reviewed evidence

Primary project evidence:

- `internal/metrics/http.go`
- `internal/metrics/outbound.go`
- `internal/middleware/metrics.go`
- `internal/middleware/status_recorder.go`
- `internal/middleware/rate_limiter.go`
- `internal/middleware/bulkhead.go`
- `internal/middleware/authorization.go`
- `internal/middleware/rbac.go`
- `internal/middleware/cors.go`
- `internal/queue/queue.go`
- `internal/workerpool/workerpool.go`
- `internal/stream/stream.go`
- `internal/db/stats.go`
- `internal/interceptors/interceptors.go`
- `internal/router/router.go`
- `internal/routes/users_handler.go`
- `internal/routes/users_handler_v2.go`
- `internal/routes/job_handler.go`
- `cmd/api/main.go`
- Audit 02, 04, 05 and 06 reports.

External design anchors:

- OpenTelemetry HTTP and RPC metric semantic conventions.
- Prometheus metric/label naming and histogram practices.
- Prometheus recording and alerting rules.
- Google SRE Workbook multi-window, multi-burn-rate alerting.
- Grafana dashboard and alert provisioning documentation.

## Executive result

### Current state

`pet-study` does not yet have an operational SLI/SLO system.

Current `expvar` variables provide process-local snapshots and cumulative counters, but the repository has no time-series backend, scraper, retention, recording rules, histograms, standardized labels, Grafana dashboards or alert manager configuration.

Several current metric defects prevent trustworthy SLO calculations even if `expvar` were scraped immediately:

1. `statusRecorder` can report a different status from the first status committed on the wire.
2. `queue_depth` is bound to the first queue instance.
3. latency has only sum/count and cannot calculate p95/p99.
4. async job counters describe incomplete transitions, not complete user outcomes.
5. outbound metrics describe physical attempts, not logical requests.
6. gRPC has no RED metrics.
7. SSE has no successful delivery, connection duration or disconnect reason metrics.
8. process-global metric state prevents clean instance and test ownership.

### Target state

The project should define user-facing SLOs over stable Prometheus time series generated from OpenTelemetry metrics, with:

- explicit eligible/good/bad event classification;
- bounded route/method/status/upstream labels;
- histogram-based latency SLIs;
- separate user-facing and component diagnostic signals;
- repository-provisioned dashboards and alert rules;
- multi-window burn-rate alerting for primary SLOs;
- symptom alerts for queue, database, upstream, runtime and telemetry pipeline saturation;
- documented runbooks and intentional-load-test muting policy.

## Principle: user-facing SLIs and component metrics are different

The primary SLO must model whether a user-visible operation succeeded within an acceptable time. It must not be a direct alert on every internal metric.

Examples:

- high DB pool utilization is a diagnostic signal, not itself a user SLO failure;
- a retry is not necessarily a user-visible failure if the logical Profile operation succeeds within latency budget;
- a `409 Conflict` can be a correct business response, not service unavailability;
- a `429` or overload `503` is user-visible inability to serve an otherwise eligible request and should be counted as bad availability for the relevant service class;
- a job reaching an expected terminal business failure is different from a job remaining non-terminal or failing because of infrastructure.

Dashboards should expose both layers, but paging-like alerts should prefer user symptoms and error-budget burn.

## Proposed service classes

The current API should not use one latency objective for every endpoint. Operations have different cost and dependency profiles.

| Service class | Current routes / operations | Rationale |
|---|---|---|
| `core_read` | `GET /api/v1/users`, `GET /api/v2/users`, `GET /api/v1/users/{id}`, `GET /api/v1/jobs/{id}` | local repository reads and response serialization |
| `core_write` | synchronous `POST /api/v1/users`, `POST /api/v2/users` | local validation and persistence |
| `async_accept` | `POST ...?async=1` | validates, persists Job and enqueues work; `202` is only acceptance, not completion |
| `dependency_read` | `GET /api/v1/users/{id}/profile` | includes outbound HTTP and retry budget |
| `grpc_bridge` | `GET /api/v1/jobs/{id}/grpc` and direct gRPC `JobsService/GetJob` | separate RPC boundary and loopback dependency |
| `export` | `GET /api/v1/users/{id}/export` | Range/ServeContent behavior and response size can differ |
| `stream` | `GET /api/v1/jobs/{id}/events` | long-lived SSE; ordinary request-duration SLO is not meaningful |
| `health` | `/livez`, `/readyz` | platform/control-plane signals, excluded from public API SLO |
| `debug` | `/debug/*` | internal/admin diagnostics, excluded from SLO and ordinary HTTP RED metrics |

A bounded `service_class` should be produced through recording rules or an explicit low-cardinality instrumentation mapping. It must never contain a user ID, job ID, raw path or query string.

## Event classification contract

### HTTP availability eligibility

For an HTTP service-class availability SLI:

- Include requests that reached the application and map to a known API operation.
- Exclude health, debug and SSE connection lifetime.
- Exclude responses that represent invalid caller input or authorization/business decisions when the service correctly evaluated the request.
- Count overload protection and server/dependency failures as bad.

Initial status classification:

| Status family/value | Availability treatment | Reason |
|---|---|---|
| `2xx`, `304` | good | requested operation or conditional response completed |
| `400`, `401`, `403`, `404`, `405`, `406`, `409`, `413`, `415`, `422` | excluded from availability numerator and denominator | caller, auth, representation or expected business outcome |
| `429` | bad | eligible work rejected by service capacity policy |
| `500`–`599` | bad | service, dependency, shutdown or capacity failure |
| cancellation caused by client disconnect before result | separately classified; normally excluded from server availability | caller abandoned request |
| server deadline/timeout | bad | service failed its budget |

This classification needs a normalized `error.type`/outcome value. Status-only classification is insufficient for distinguishing client cancellation from server deadline.

### Latency eligibility

Latency SLIs should count successfully completed eligible operations, normally `2xx` and `304`, and should be split by `service_class`.

Do not include:

- SSE total connection duration;
- debug/health;
- invalid requests rejected before business processing;
- canceled requests when the client abandoned the operation.

TTFB should be added separately for HTTP/SSE diagnostics. It is not a replacement for end-to-end request duration.

### Async job eligibility

An accepted job begins when the server commits `202 Accepted` after successful queue enqueue.

Async reliability must be separated into at least three SLIs:

1. **Acceptance availability:** an eligible async request receives `202` rather than `429/503/5xx`.
2. **Terminality:** an accepted job reaches any terminal state within the target time.
3. **System outcome:** an accepted job does not fail because of internal, timeout, shutdown or persistence failure.

Expected business failure such as a concurrent uniqueness conflict should not be merged with an internal worker/storage failure. It should have a bounded terminal category.

Required terminal categories:

```text
success
business_failure
system_failure
shutdown_failure
unknown
```

### Outbound eligibility

The Profile integration must expose two levels:

- physical HTTP attempts;
- logical Profile operation across retries.

The user-facing dependency SLI uses the logical operation. Physical attempts are diagnostic and calculate retry amplification.

### gRPC eligibility

For gRPC availability:

- `OK` is good;
- `INVALID_ARGUMENT`, `NOT_FOUND`, `PERMISSION_DENIED`, `UNAUTHENTICATED` are normally excluded expected caller/domain outcomes;
- `INTERNAL`, `UNAVAILABLE`, server-generated `DEADLINE_EXCEEDED`, `RESOURCE_EXHAUSTED`, `UNKNOWN`, transport failures are bad;
- client cancellation should be separated from server cancellation/deadline.

### SSE eligibility

SSE needs connection and delivery SLIs rather than total request duration:

- successful authorized stream establishment;
- write/flush success;
- delivered meaningful job transitions;
- dropped delivery ratio;
- unexpected disconnect/write timeout ratio;
- heartbeat continuity.

Heartbeats must not count as business event deliveries.

## Provisional laboratory objectives

These are initial hypotheses for a local production-like laboratory. They must be recalibrated after Audit 09 establishes repeatable baseline throughput and latency on a defined machine profile.

### SLO 1 — Core HTTP availability

```text
Objective: 99.9% over a rolling 30-day window
Scope: core_read + core_write
```

A 99.9% objective has a 0.1% error budget, equivalent to approximately 43 minutes 12 seconds of continuous total unavailability over 30 days. Request-based budget consumption remains the actual implementation.

Why provisional:

- there is no real traffic profile yet;
- current metric correctness defects must be fixed first;
- low-volume periods make request-ratio burn alerts noisy.

### SLO 2 — Core HTTP latency

```text
core_read:  99% of successful eligible requests <= 500 ms
core_write: 99% of successful eligible requests <= 1 s
Window: rolling 30 days
```

These targets are intentionally loose for the first PostgreSQL/local-container baseline. Audit 09 should lower or revise them based on measured distributions, not intuition.

### SLO 3 — Async acceptance

```text
Objective: 99.9% eligible async create requests receive 202
Window: rolling 30 days
```

Queue-full, queue-stopped, bulkhead and rate-limit rejections count as bad for this SLI, while malformed/auth/business-invalid requests are excluded.

### SLO 4 — Async terminality and system outcome

```text
Terminality: 99.9% of accepted jobs reach terminal state within 10 s
System outcome: 99.9% of accepted jobs avoid system_failure/shutdown_failure
Window: rolling 30 days
```

The 10-second threshold is an initial lab target. The system must expose queue wait and end-to-end duration before it can be evaluated.

### SLO 5 — Profile logical operation

```text
Availability: 99.5% logical operations complete without dependency/system failure
Latency: 95% of successful logical operations <= 1 s
Window: rolling 30 days
```

A 99.5% objective has a 0.5% budget, approximately 3 hours 36 minutes over 30 days. This class is intentionally separate because it includes an external dependency and retries.

### SLO 6 — gRPC GetJob

```text
Availability: 99.9% eligible calls avoid server/transport failures
Latency: 99% of successful calls <= 500 ms
Window: rolling 30 days
```

The loopback bridge should normally be much faster; the initial objective leaves room for PostgreSQL and local test noise. Audit 09 should establish direct and bridge baselines separately.

### SLO 7 — SSE delivery reliability

```text
Connection establishment: 99.5% successful eligible handshakes <= 1 s
Meaningful event delivery drop ratio: <= 0.1%
Unexpected write-timeout/server-close ratio: <= 0.5%
Window: rolling 30 days
```

This SLO cannot be implemented with current metrics because successful deliveries and connection outcomes are not counted.

### Operational objective — telemetry pipeline

Telemetry is not part of application readiness, but its loss must be observable.

Initial objective:

```text
No sustained exporter/Collector drops.
99.9% of accepted telemetry items exported during ordinary operation.
Final application shutdown completes telemetry flush within a bounded timeout.
```

This is an internal operational objective rather than a user-facing service SLO.

## Required target metric contract

Exact Prometheus exposition names must be integration-tested because OpenTelemetry exporter name conversion and semantic-convention versions can change. The following table defines required meaning, not an unchecked dependency on a specific library version.

### Standard HTTP/RPC metrics

Use OpenTelemetry semantic conventions where supported:

| Logical instrument | Type | Required bounded attributes |
|---|---|---|
| `http.server.request.duration` | histogram, seconds | method, matched route, response status, error type, protocol version |
| `http.server.active_requests` | up/down counter | method; route only if instrumentation can populate it correctly and boundedly |
| `http.server.request.body.size` | histogram, bytes | method, route |
| `http.server.response.body.size` | histogram, bytes | method, route, status |
| `http.client.request.duration` | histogram, seconds | upstream/server address, stable operation/route, status/error type |
| `http.client.active_requests` | up/down counter | upstream |
| `rpc.server.call.duration` | histogram, seconds | rpc system, full logical method, response status/error type |
| `rpc.client.call.duration` | histogram, seconds | rpc system, full logical method, response status/error type |

Do not place request ID, trace ID, user ID, job ID, raw URL, email or error message in metric labels.

### Project-owned business and saturation metrics

Recommended contract:

| Metric | Type | Attributes | Purpose |
|---|---|---|---|
| `pet_study_http_request_outcomes` | counter | service class, outcome | explicit SLO numerator/denominator classification if standard HTTP metrics cannot express it safely |
| `pet_study_jobs_accepted` | counter | API version | accepted jobs after successful enqueue |
| `pet_study_jobs_terminal` | counter | terminal category | complete job outcomes |
| `pet_study_job_queue_wait` | histogram seconds | terminal category optional | enqueue to worker start |
| `pet_study_job_processing_duration` | histogram seconds | terminal category | worker start to terminal attempt |
| `pet_study_job_end_to_end_duration` | histogram seconds | terminal category | accepted to terminal state |
| `pet_study_jobs_current` | up/down counter | state | current queued/running state, reconciled with storage semantics |
| `pet_study_queue_depth` | gauge | queue name | current queue occupancy |
| `pet_study_queue_capacity` | gauge | queue name | configured capacity |
| `pet_study_queue_oldest_item_age` | gauge seconds | queue name | age of oldest item |
| `pet_study_queue_operations` | counter | queue name, operation, outcome/reason | accepted/rejected/dequeued/stopped/canceled |
| `pet_study_workers_active` | up/down counter | pool name | active jobs |
| `pet_study_workers_configured` | gauge | pool name | configured concurrency |
| `pet_study_outbound_operations` | counter | upstream, operation, final outcome | logical request result |
| `pet_study_outbound_operation_duration` | histogram seconds | upstream, operation, final outcome | logical latency across retries |
| `pet_study_outbound_attempts` | counter | upstream, operation, outcome | physical retry attempts |
| `pet_study_outbound_backoff_duration` | histogram seconds | upstream, operation | time spent delaying retries |
| `pet_study_rate_limit_decisions` | counter | policy, outcome | allow/reject without caller identity |
| `pet_study_bulkhead_operations` | counter | policy, outcome | accepted/rejected |
| `pet_study_bulkhead_in_flight` | up/down counter | policy | current protected concurrency |
| `pet_study_sse_connections` | up/down counter | route | active streams |
| `pet_study_sse_connection_outcomes` | counter | outcome/reason | opened, client_disconnect, server_close, write_timeout, write_error |
| `pet_study_sse_connection_duration` | histogram seconds | outcome | stream lifetime diagnostic |
| `pet_study_sse_events` | counter | event type, outcome | delivered/dropped meaningful events |
| `pet_study_authn_failures` | counter | bounded reason | missing/invalid/expired/other |
| `pet_study_authz_denials` | counter | bounded reason/policy | RBAC/resource denial categories |
| `pet_study_cors_decisions` | counter | outcome/reason | allowed preflight/denied origin/method/header |
| DB pool standard metrics | gauges/counters | pool name | open/in-use/idle/max/wait/closed |
| `pet_study_db_operation_duration` | histogram seconds | operation, outcome | normalized repository operations, not raw SQL |
| `pet_study_db_operations` | counter | operation, outcome | success/error/timeout/canceled |

Prometheus exposition should use seconds and bytes as base units. Do not embed labels in metric names.

## Histogram bucket guidance

Initial explicit duration boundaries should cover both normal and degraded operation.

### HTTP and gRPC

```text
0.005, 0.01, 0.025, 0.05, 0.075, 0.1,
0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10 seconds
```

This is aligned with current RPC semantic-convention guidance and is a reasonable initial shared shape. It should be reviewed after baseline measurements.

### Job end-to-end

```text
0.01, 0.025, 0.05, 0.1, 0.25, 0.5,
1, 2, 5, 10, 30, 60 seconds
```

### Queue wait

```text
0.001, 0.005, 0.01, 0.025, 0.05, 0.1,
0.25, 0.5, 1, 2, 5, 10 seconds
```

Buckets should be defined through OTel views/configuration rather than scattered in handlers.

## Proposed recording rules

The following PromQL is conceptual and assumes Prometheus-normalized OTel names/labels. Exact names must be verified against the selected Collector/exporter version in integration tests.

### Core HTTP request rate

```promql
sum by (service_class) (
  rate(pet_study_http_request_outcomes_total{service_class=~"core_read|core_write"}[5m])
)
```

### Core HTTP bad-event ratio

```promql
sum(rate(pet_study_http_request_outcomes_total{
  service_class=~"core_read|core_write",
  outcome="bad"
}[5m]))
/
sum(rate(pet_study_http_request_outcomes_total{
  service_class=~"core_read|core_write",
  outcome=~"good|bad"
}[5m]))
```

Recommended recording-rule names:

```text
pet_study:slo_core_http_bad_ratio:rate5m
pet_study:slo_core_http_bad_ratio:rate30m
pet_study:slo_core_http_bad_ratio:rate1h
pet_study:slo_core_http_bad_ratio:rate2h
pet_study:slo_core_http_bad_ratio:rate6h
pet_study:slo_core_http_bad_ratio:rate24h
pet_study:slo_core_http_bad_ratio:rate3d
```

### HTTP p95/p99

```promql
histogram_quantile(
  0.99,
  sum by (le, service_class) (
    rate(http_server_request_duration_seconds_bucket{
      service_class=~"core_read|core_write"
    }[5m])
  )
)
```

### Threshold-based latency good ratio

For `core_read <= 0.5s`:

```promql
sum(rate(http_server_request_duration_seconds_bucket{
  service_class="core_read", le="0.5"
}[30d]))
/
sum(rate(http_server_request_duration_seconds_count{
  service_class="core_read"
}[30d]))
```

The actual implementation should ensure this denominator includes only latency-eligible successful operations.

### Async terminality ratio

```promql
sum(increase(pet_study_job_end_to_end_duration_seconds_bucket{le="10"}[30d]))
/
sum(increase(pet_study_jobs_accepted_total[30d]))
```

This needs consistent correlation between accepted and terminal jobs and must handle jobs still open at the window boundary. A state/event model or periodic reconciliation may be required for strict correctness.

### Async system-failure ratio

```promql
sum(rate(pet_study_jobs_terminal_total{
  terminal_category=~"system_failure|shutdown_failure"
}[1h]))
/
sum(rate(pet_study_jobs_terminal_total[1h]))
```

### Queue utilization

```promql
pet_study_queue_depth{queue="users_create"}
/
pet_study_queue_capacity{queue="users_create"}
```

### Retry amplification

```promql
sum(rate(pet_study_outbound_attempts_total{upstream="profile"}[5m]))
/
sum(rate(pet_study_outbound_operations_total{upstream="profile"}[5m]))
```

A value of `1` means no retries. Values above `1` indicate amplification.

### SSE drop ratio

```promql
sum(rate(pet_study_sse_events_total{outcome="dropped"}[5m]))
/
sum(rate(pet_study_sse_events_total{outcome=~"delivered|dropped"}[5m]))
```

## Burn-rate alert design

The primary Core HTTP availability objective uses 99.9%, therefore the allowed bad ratio is:

```text
1 - 0.999 = 0.001
```

Use multi-window, multi-burn-rate alerts rather than a single instantaneous error-rate threshold.

### Page-like critical alert

```promql
(
  pet_study:slo_core_http_bad_ratio:rate1h > 14.4 * 0.001
  and
  pet_study:slo_core_http_bad_ratio:rate5m > 14.4 * 0.001
)
or
(
  pet_study:slo_core_http_bad_ratio:rate6h > 6 * 0.001
  and
  pet_study:slo_core_http_bad_ratio:rate30m > 6 * 0.001
)
```

In this educational project, `page` means a high-severity Grafana/Alertmanager notification. It does not require a real personal pager.

### Ticket-like warning alert

```promql
(
  pet_study:slo_core_http_bad_ratio:rate24h > 3 * 0.001
  and
  pet_study:slo_core_http_bad_ratio:rate2h > 3 * 0.001
)
or
(
  pet_study:slo_core_http_bad_ratio:rate3d > 1 * 0.001
  and
  pet_study:slo_core_http_bad_ratio:rate6h > 1 * 0.001
)
```

### Low-traffic guard

The project may have very low traffic outside load tests. A single failure can produce a misleading enormous burn rate.

Burn alerts should therefore include one of:

- a minimum eligible request count in the long window;
- synthetic probe traffic;
- separate low-traffic availability probes;
- a lab run label/session with known load.

Example conceptual guard:

```promql
sum(increase(pet_study_http_request_outcomes_total{
  service_class=~"core_read|core_write",
  outcome=~"good|bad"
}[1h])) >= 100
```

A black-box probe should not replace request-based SLOs, but it can provide a clear signal when ordinary traffic is absent.

## Symptom and saturation alerts

These alerts do not replace burn-rate alerts. They accelerate diagnosis and protect upcoming high-load experiments.

### Application readiness

- `readyz` unavailable for more than a short startup/shutdown grace period.
- process absent/restarting unexpectedly.
- readiness failure reason grouped by bounded check name.

Do not page during intentional shutdown/deployment windows.

### Queue

Candidate warnings:

```text
queue utilization > 0.80 for 10m
oldest item age > 5s for 5m
queue rejection rate > 0 outside an intentional overload experiment
```

Candidate critical conditions:

```text
queue utilization > 0.95 for 5m AND depth is growing
oldest item age > job terminality threshold
accepted jobs not reaching terminal state
```

Depth alone is insufficient; combine depth, capacity, oldest age, dequeue rate and active workers.

### Workers/jobs

- configured workers > 0 but active/running capability is absent;
- all workers saturated while queue age grows;
- terminal system-failure ratio exceeds objective;
- active jobs remain non-terminal after shutdown repair;
- storage transition errors are non-zero.

### Rate limiter and bulkhead

- rejection ratio increases unexpectedly under ordinary traffic;
- in-flight reaches configured capacity with rising latency;
- rate/bulkhead rejection is correlated with SLO burn.

Do not page only because protection mechanisms activate during a deliberate stress test. Load-test runs need labels, annotations or scheduled silences.

### Outbound Profile

Candidate warnings:

```text
logical failure ratio > 5% for 10m
retry amplification > 1.2 for 10m
p95 logical latency > 1s for 10m
```

Candidate critical:

```text
logical failure ratio > 20% for 5m
retry amplification > 2 with latency/error growth
all operations failing with unavailable/timeout
```

Physical attempt errors alone should normally not page when the logical operation still succeeds within objective.

### gRPC

- server/transport bad ratio above objective;
- p99 duration above threshold;
- `UNAVAILABLE`, server `DEADLINE_EXCEEDED` or `INTERNAL` increase;
- loopback client connection/runtime not serving while enabled.

Expected `NOT_FOUND` and auth/domain statuses should not trigger availability alerts.

### SSE

- dropped meaningful event ratio > 0.1% for 5m;
- write timeout/error rate > 0 under ordinary operation;
- active subscribers grow without corresponding closes;
- connection duration distribution and goroutine count suggest leaks;
- heartbeat/write failures correlate with proxy/TLS behavior.

### PostgreSQL

Candidate warnings:

```text
in_use / max_open_connections > 0.80 for 10m
rate(wait_count[5m]) > 0 with growing wait duration
repository p95 duration above route budget
```

Critical:

```text
pool saturated near 1.0 with growing waits and HTTP/job SLO burn
DB readiness failing outside deployment/startup
systematic timeout/error outcome growth
```

Pool utilization alone should not page if user SLIs remain healthy.

### Go runtime/process

Monitor:

- process CPU;
- resident/heap memory;
- heap allocation rate;
- GC cycles and pause time;
- goroutine count and growth;
- mutex/block profiles during experiments;
- file descriptors/sockets where available;
- process restart count.

Static thresholds should be derived from the container/machine budget and baseline. Prefer alerts on sustained growth, exhaustion proximity and user impact.

### Telemetry pipeline

Alert when:

- application exporter send failures/drops are sustained;
- Collector refuses or drops spans/metrics;
- Collector sending queue utilization is high;
- Prometheus cannot scrape Collector/application target;
- Tempo/Loki export fails persistently;
- Collector process memory limiter frequently refuses data;
- expected service telemetry disappears while the service remains ready.

Collector outage must not make application readiness fail, but it must produce a separate observability-degraded alert.

## Dashboard plan

Dashboards must be provisioned from repository files and reviewed in source control. UI-only dashboard changes are not the source of truth.

### Dashboard 1 — Service Overview

Purpose: answer “Are users affected?” within one screen.

Panels:

1. Core availability SLO and 30-day error budget remaining.
2. Current burn rate: 5m, 1h, 6h, 3d.
3. Request rate by service class.
4. Bad-event ratio by service class.
5. p50/p95/p99 duration by service class.
6. active HTTP requests and active SSE connections.
7. current readiness and process uptime/restarts.
8. queue utilization and oldest job age.
9. Profile logical success and retry amplification.
10. active alerts and recent deploy/build version annotations.

### Dashboard 2 — HTTP API

Panels:

- request rate by method and matched route;
- response status distribution;
- error type distribution;
- duration heatmap/histogram;
- p95/p99 by route;
- response/request body size;
- protocol version (`1.1`, `2`);
- TTFB where implemented;
- cancellations/timeouts;
- 429, bulkhead 503 and queue 503 separately;
- exemplars linking slow/error observations to Tempo traces.

### Dashboard 3 — Queue, Workers and Jobs

Panels:

- depth, capacity and utilization;
- oldest item age;
- enqueue/dequeue/rejection rate by reason;
- configured/active workers;
- accepted and terminal jobs;
- current states;
- queue-wait histogram;
- processing and end-to-end p95/p99;
- terminal category ratio;
- non-terminal age distribution;
- event delivery to SSE.

### Dashboard 4 — Outbound and Resilience

Panels:

- logical operation rate/outcome;
- physical attempts;
- retry amplification;
- retries exhausted;
- backoff time;
- logical and attempt latency histograms;
- timeout/network/4xx/5xx/parse classifications;
- HTTP client active/open connections;
- rate limiter and bulkhead decisions;
- future circuit-breaker state if added.

### Dashboard 5 — gRPC

Panels:

- client and server call rate;
- response status codes;
- client/server duration p95/p99;
- in-flight calls if exposed;
- deadline/cancellation;
- HTTP bridge latency compared with direct gRPC server latency;
- trace exemplars.

### Dashboard 6 — SSE

Panels:

- active connections;
- opens/closes by reason;
- connection duration;
- events published, delivered and dropped;
- drop ratio;
- write timeout/error;
- heartbeat success/failure;
- goroutines and memory correlated with active subscribers.

### Dashboard 7 — PostgreSQL

Panels:

- open/in-use/idle/max connections;
- pool utilization;
- wait count rate and wait duration;
- operation rate/outcome by repository operation;
- query/operation duration p95/p99;
- timeouts/cancellations;
- readiness failures;
- application SLO correlation.

Do not use raw SQL or user IDs as metric labels.

### Dashboard 8 — Go Runtime and Process

Panels:

- CPU and memory;
- heap live/allocated;
- allocation rate;
- GC cycles and pauses;
- goroutine count;
- mutex/block indicators;
- process uptime/restarts;
- network/file descriptor usage when available.

### Dashboard 9 — Security and Edge Policy

Panels:

- authn failures by bounded reason;
- authz denials by policy/reason;
- CORS allowed/denied/preflight;
- trusted-proxy/request-ID sanitation failures if instrumented;
- 401/403 by matched route without token/user identifiers;
- rate-limit/bulkhead outcomes.

This dashboard is diagnostic and should not expose secrets, JWT claims or PII.

### Dashboard 10 — Telemetry Pipeline

Panels:

- application exporter success/failure/drop;
- Collector accepted/refused/dropped spans and metric points;
- Collector queue utilization and memory limiter activity;
- export latency/failures to Tempo/Prometheus/Loki;
- Prometheus scrape health;
- service telemetry freshness;
- trace/log correlation smoke checks.

## Grafana and alert provisioning layout

Recommended repository layout:

```text
observability/
  collector/
    config.yaml
  prometheus/
    prometheus.yml
    rules/
      recording.yml
      slo-alerts.yml
      component-alerts.yml
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
      alerting/
    dashboards/
      service-overview.json
      http-api.json
      jobs-queue.json
      outbound.json
      grpc.json
      sse.json
      postgres.json
      go-runtime.json
      security.json
      telemetry-pipeline.json

docs/observability/
  SLI_SLO.md
  METRIC_CONTRACT.md
  ALERT_POLICY.md
  RUNBOOKS.md
```

Dashboard JSON and file-provisioned alert resources should be version controlled. Provisioned resources should be treated as code rather than edited permanently in the UI.

## Alert labels and routing policy

Every alert should have bounded labels such as:

```text
service=pet-study
environment=local|lab|stage-like
severity=critical|warning|info
signal=slo|queue|db|outbound|grpc|sse|telemetry
owner=pet-study
runbook=<stable identifier>
```

Do not label alerts with request ID, trace ID, user ID, job ID or raw error message.

Suggested semantics:

- `critical`: active fast SLO burn, major loss of service, persistent telemetry blindness during an experiment.
- `warning`: slow burn, sustained saturation, growing retry amplification, queue age risk.
- `info`: expected laboratory event, deployment annotation, intentional chaos/load condition.

For local use, notifications may remain inside Grafana initially. Alert rules should still be provisioned and testable as code.

## Intentional load/failure experiment policy

The future load laboratory will deliberately trigger queue full, limiter rejection, DB saturation and upstream failure. Alerts must distinguish experiments from accidental incidents.

Required mechanism:

- every load run has a unique bounded `test_run` identifier in logs/traces, but **not** as a long-lived Prometheus label if unbounded;
- Grafana annotations record experiment start/end and scenario;
- alert silences or maintenance windows are created for expected component alerts;
- primary SLO burn remains visible even when notifications are muted;
- after each experiment, a report records load profile, error budget consumed, bottleneck, profiles and remediation.

Do not permanently weaken alert thresholds just because a chaos/load experiment intentionally fired them.

## Runbook minimum contract

Every actionable alert should link to a short runbook containing:

1. symptom and affected SLO;
2. exact query/panel;
3. first checks;
4. related logs and trace filters;
5. likely causes;
6. safe mitigation;
7. validation that service recovered;
8. post-incident evidence to preserve.

Example queue saturation investigation:

```text
1. Check queue depth/capacity and oldest age.
2. Check workers configured/active and job processing p99.
3. Check PostgreSQL waits and operation latency.
4. Check terminal transition/storage errors.
5. Check upstream dependencies if worker processing uses them.
6. Compare request rate and async acceptance rejections.
7. Inspect representative slow traces using exemplars.
```

## Findings

### SLO-1 — Current data cannot support trustworthy SLOs

Severity: **HIGH**

There is no Prometheus/OTel time-series pipeline, retention, histogram distribution or recording-rule layer. `expvar` snapshots alone cannot calculate rolling availability, p99 or error-budget burn.

### SLO-2 — Current status attribution can corrupt availability

Severity: **HIGH**

The repeated-`WriteHeader` recorder defect can attribute requests to a status different from the wire response. SLO work must not start before this is fixed and tested.

### SLO-3 — Service-class mapping is not formally encoded

Severity: **HIGH**

Current HTTP metrics have method, pattern and status, but no explicit controlled grouping for core, dependency, async, export and stream semantics. SLO recording rules need a reviewed mapping that cannot silently omit new routes.

### SLO-4 — Average-only latency cannot define latency objectives

Severity: **HIGH**

Current sum/count metrics provide only averages. Histogram instruments and reviewed boundaries are mandatory for threshold ratios and p95/p99.

### SLO-5 — Async acceptance and completion are conflated/incomplete

Severity: **HIGH**

`jobs_total` records partial transitions. The project cannot currently calculate accepted-to-terminal ratio, queue wait, end-to-end duration or system-vs-business failure.

### SLO-6 — Outbound user outcome is not represented

Severity: **HIGH**

Current outbound metrics count physical attempts. They cannot calculate logical Profile success or latency across retries and can make retries look like multiple user operations.

### SLO-7 — gRPC is absent from SLO telemetry

Severity: **HIGH**

No gRPC RED metrics exist, so direct and bridge operations are invisible to dashboards and alerting.

### SLO-8 — SSE delivery reliability is not measurable

Severity: **HIGH**

Current metrics count publish calls and drops, but not successful deliveries, connection outcomes, write errors or timeouts. A drop ratio denominator is unavailable.

### SLO-9 — No alert/dashboard-as-code artifacts exist

Severity: **MEDIUM/HIGH**

The repository has no Grafana provisioning, Prometheus rules, dashboard JSON or runbooks. Manual UI creation would be non-reproducible.

### SLO-10 — Low traffic requires explicit guardrails

Severity: **MEDIUM**

Outside load tests, request counts may be too low for reliable burn-rate paging. Synthetic probes, minimum-event guards and separate lab sessions are required.

### SLO-11 — Operational and user-facing alerts must be separated

Severity: **MEDIUM**

Queue, DB and exporter metrics are necessary for diagnosis but should not independently replace user-symptom alerts. The alert policy must prevent noisy component-only paging.

### SLO-12 — Numerical objectives are provisional until baseline audit

Severity: **MEDIUM**

The targets in this report are an explicit starting hypothesis. Audit 09 must measure stable latency/throughput/resource baselines and recommend confirmed values.

## Required prerequisites before implementation

1. Fix first-status-only `statusRecorder` semantics and add regression tests.
2. Replace process-global metric ownership with composition-root/DI ownership.
3. Fix queue gauge instance ownership.
4. Establish standard resource attributes and low-cardinality policy.
5. Add OTel HTTP/RPC histograms and Prometheus export.
6. Implement explicit SLO outcome classification.
7. Add async accepted/terminal/current/duration metrics.
8. Add outbound logical operation metrics.
9. Add gRPC RED metrics.
10. Add SSE connection/delivery outcome metrics.
11. Provision recording rules before alert rules and dashboards.
12. Establish repeatable baseline traffic before treating thresholds as final.

## Implementation acceptance criteria for the future roadmap

The SLI/SLO capability can be considered complete only when:

- a clean repository startup provisions Collector, Prometheus, Grafana, Tempo and Loki/Alloy;
- Prometheus contains stable HTTP/RPC/business metrics with bounded labels;
- histogram queries return meaningful p95/p99 and threshold ratios;
- service-class availability good/bad/excluded classification is tested;
- async accepted-to-terminal and end-to-end SLI is queryable;
- logical Profile operation is separate from attempts;
- gRPC and SSE dashboards have real data;
- recording and alert rules pass validation/unit tests;
- Grafana dashboards and alerts are provisioned from repository files;
- a synthetic failure fires the expected warning/critical rule;
- a recovery clears the alert;
- low-traffic guard behavior is tested;
- intentional load-test silencing/annotation procedure is documented;
- every actionable alert has a runbook;
- shutdown/exporter failures are visible without affecting `/readyz`.

## Decision carried forward

SLI/SLO and alerting status is **DESIGNED / NOT IMPLEMENTED**.

The observability implementation roadmap must not begin with dashboard JSON. It should first build trustworthy metric semantics and a time-series pipeline, then recording rules, then SLO alerts, and finally dashboards/runbooks. Dashboards created before metric correctness would visualize misleading data.
