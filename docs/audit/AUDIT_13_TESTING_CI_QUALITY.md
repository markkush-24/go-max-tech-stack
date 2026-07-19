# Audit 13 — Testing and CI Quality

## Scope

This pass evaluates whether the uploaded `pet-study` working tree can prove its behavior through repeatable automated checks. It covers:

- unit, handler, router, lifecycle and protocol tests;
- `testkit` fidelity to the production composition root;
- concurrency/race coverage;
- PostgreSQL and migration verification;
- fuzzing, benchmarks, golden and contract tests;
- protobuf/code-generation reproducibility;
- CI workflow and local quality entrypoints;
- vulnerability/static-analysis checks;
- repository cleanliness and clean-checkout reproducibility.

The application code was not changed.

## Execution constraints and dynamic evidence

Target toolchain: Go `1.25.8`.

Sandbox toolchain: Go `1.23.2`.

The complete module still cannot execute in this sandbox because the required external module archives are not available locally and network access is disabled. A disposable Go 1.23-compatible copy was used only for dependency-light checks.

Successfully executed in the disposable copy:

```text
go test ./internal/requestid ./internal/runtimeinfo ./internal/stream ./internal/queue

go test -race ./internal/requestid ./internal/runtimeinfo ./internal/stream ./internal/queue
```

Results:

- `requestid` tests passed;
- `runtimeinfo` tests passed;
- `stream` and `queue` compiled but contain no committed tests;
- race-enabled execution passed for the exercised packages;
- this is compatibility evidence only and does not replace Go 1.25.8 CI.

Static inventory:

```text
119 Go files
27 test files
86 top-level Test functions
0 Benchmark functions
0 Fuzz functions
0 Example functions
0 t.Parallel calls
6 time.Sleep calls in tests
10 httptest.NewServer usages
41 httptest.NewRecorder usages
```

## Executive result

Testing and CI quality are **PARTIAL/BROKEN**.

The project has a meaningful functional test suite. It covers many HTTP status paths, JWT rejection, RBAC, CORS, request ID, TLS startup, Profile retries, ETag, Protobuf negotiation, SSE happy paths, Range and basic gRPC behavior. The intended CI workflow also includes a useful baseline of formatting, tests, vet, race, staticcheck and vulnerability scanning.

However, the suite does not currently prove the highest-risk parts identified by Audits 01–12. Critical queue/worker/job lifecycle code has no direct committed tests. PostgreSQL repositories and migrations have no integration suite. `testkit.NewServer` is described as full-stack but differs materially from `cmd/api/main.go`, uses two different SSE Hubs and automatically injects an admin token. Generated protobuf code, CI, scripts and the tools module are untracked, so a clean checkout of the current Git state does not contain the quality system described by README. There are no fuzz targets, benchmarks, load scripts, OpenAPI contract tests or coverage gates.

## Positive findings

### QP1 — Broad functional HTTP coverage exists

The existing suite includes direct checks for:

- routing and method errors;
- strict JSON and payload-size errors;
- validation and Problem-shaped responses;
- request-ID generation and propagation;
- JWT missing/invalid/expired cases;
- RBAC and resource ownership;
- CORS allow/deny/preflight;
- security headers and trusted proxy behavior;
- rate limiter and bulkhead rejection;
- ETag/304;
- JSON/Protobuf negotiation;
- Profile success, retry, timeout and error classification;
- SSE owner/admin happy paths;
- Range `206` behavior;
- gRPC success/invalid/not-found;
- HTTP/TLS startup and selected shutdown behavior.

This is substantially better than a project containing only isolated service unit tests.

### QP2 — Appropriate stdlib test tools are used

The suite uses:

- `httptest.NewRecorder` for focused handler/middleware tests;
- `httptest.NewServer` for socket-level HTTP behavior;
- `bufconn` for gRPC tests;
- generated self-signed certificates for TLS startup tests;
- `t.Cleanup`, bounded contexts and timeouts in many integration-style tests.

### QP3 — Intended CI baseline is sensible

The uploaded workflow runs:

```text
gofmt check
go test ./...
go vet ./...
go test -race ./...
staticcheck ./...
govulncheck -mode binary
```

Versions of `staticcheck` and `govulncheck` are pinned in the uploaded tooling files rather than installed from an unconstrained `latest` tag.

## Findings

### Q1 — The quality system is not part of the tracked repository

The uploaded working tree contains:

- `.github/workflows/ci.yml`;
- `scripts/*.ps1`;
- `tools/go.mod`, `tools/go.sum`, `tools/tools.go`.

`git ls-files` does not include any of them.

Consequences:

- the remote/clean checkout of current `HEAD` has no CI workflow;
- README promises commands that do not exist in tracked Git state;
- pinned tool versions are not reproducible from current `HEAD`;
- a reviewer cannot rely on CI status for the uploaded code.

Severity: **CRITICAL for reproducibility/governance**.

### Q2 — Generated protobuf code is required but untracked

Tracked files include:

```text
internal/transport/pb/job.proto
internal/transport/pb/user.proto
```

The working tree also contains required generated files:

```text
job.pb.go
job_grpc.pb.go
user.pb.go
```

They are not tracked and are imported by tracked application code.

The README generation commands install plugins using `@latest`, while generated headers record specific versions. There is no committed script or CI step that:

1. installs exact generator versions;
2. regenerates protobuf code;
3. fails on a non-empty diff.

A clean checkout is therefore not proven buildable or reproducibly generatable.

Severity: **CRITICAL**.

### Q3 — `testkit.NewServer` is not production full-stack

The test helper differs materially from `cmd/api/main.go`.

It does not include the real production behavior for:

- trusted-proxy request-ID sanitization;
- `TrustProxy` request information;
- real JWT verification;
- `APIServer.Run` and complete shutdown;
- worker pool startup;
- gRPC runtime/client;
- PostgreSQL repositories;
- TLS/HTTP2;
- production logger/config construction.

It also automatically injects `Authorization: Bearer test` and defaults to an admin principal. This can hide missing auth headers and authorization mistakes unless a test explicitly opts out.

The helper should be described as an in-memory HTTP integration fixture, not a faithful full-stack deployment.

Severity: **HIGH**.

### Q4 — `testkit` wires producers and SSE consumers to different Hubs

`newApp` creates:

```text
hub      -> User v1/v2 async producers
eventHub -> JobHandler SSE consumer and App.EventHub
```

A test can publish manually to `App.EventHub` and pass while the actual async producer publishes to another Hub.

Therefore the current suite does not prove the end-to-end path:

```text
HTTP async create
→ queue
→ worker
→ Hub.Publish
→ SSE client
```

Severity: **HIGH**.

### Q5 — The highest-risk concurrency packages have no direct tests

No committed test files exist for:

- `internal/queue`;
- `internal/stream`;
- `internal/workerpool`;
- `internal/metrics`;
- `internal/interceptors`;
- `internal/transport/grpcclient`;
- `internal/store/jobrepo`;
- `internal/db`;
- `internal/security`;
- `internal/service`.

Critical defects from Audits 04, 10 and 11 were therefore not caught by the normal suite:

- queue-depth binding to the first instance;
- canceled-context enqueue nondeterminism;
- repeated `WorkerPool.Stop` terminal-state overwrite;
- unsafe worker generation restart;
- terminal job-state overwrite;
- missing job transition repair;
- backoff overflow;
- permissive Profile response validation.

`go test -race ./...` cannot detect a race path that no test executes.

Severity: **CRITICAL/HIGH**.

### Q6 — PostgreSQL and migrations are untested

There are no committed tests for:

- `db.Open` and pool configuration;
- `WithinTransaction` commit/rollback/panic behavior;
- SQLX user repository;
- SQLX job repository;
- error conversion for unique violations;
- context timeout/cancellation;
- job transition updates;
- migration up/down behavior;
- schema readiness/version;
- multi-instance job interaction.

The CI workflow does not start PostgreSQL and `docker-compose.yml` defines only the database, without migration execution or application integration checks.

Because the default runtime storage backend is currently PostgreSQL, this is a major confidence gap.

Severity: **CRITICAL/HIGH**.

### Q7 — No OpenAPI or runtime contract test exists

The suite cannot mechanically compare actual responses with a machine-readable API contract because no OpenAPI document exists.

Missing checks include:

- undocumented route drift;
- status/content-type/schema conformance;
- security requirements;
- request/response headers;
- Problem Details variants;
- breaking API changes;
- generated client compatibility.

Severity: **HIGH**.

### Q8 — There are no fuzz targets

Zero `Fuzz*` functions are committed.

High-value fuzz surfaces include:

- strict JSON parser;
- Accept header parsing;
- Content-Type parsing;
- Authorization/Bearer parsing;
- request-ID validation;
- ETag/If-None-Match parsing;
- X-Forwarded-For parsing;
- path/query ID parsing;
- protobuf/JSON negotiation boundaries;
- retry/backoff configuration arithmetic.

Many Audit 12 protocol defects are exactly the kind of edge cases fuzz/property tests can find.

Severity: **HIGH for final quality step**.

### Q9 — There are no committed benchmarks or load scripts

Zero `Benchmark*` functions exist.

The repository also has no versioned k6/Vegeta/Gatling/hey workload definitions.

Therefore there is no repeatable measurement of:

- HTTP handler throughput;
- allocations;
- JWT overhead;
- JSON versus Protobuf;
- queue/worker throughput;
- SSE fan-out;
- gRPC versus HTTP bridge;
- PostgreSQL pool saturation;
- logging/metrics/tracing overhead;
- shutdown under load.

Severity: **HIGH for the planned load laboratory**.

### Q10 — No coverage report or coverage policy exists

CI does not generate coverage and no threshold or package-risk policy is defined.

A single global percentage would not be sufficient, but the complete absence of coverage reporting makes untested critical packages less visible.

A future policy should distinguish:

- critical state-machine/concurrency packages;
- protocol parsers;
- adapters/integration packages;
- generated code.

Severity: **MEDIUM/HIGH**.

### Q11 — Global `expvar` state prevents test isolation

Metrics tests read process-global variables and compare before/after values. No test invokes `t.Parallel`, and global state makes safe parallelization difficult.

Consequences:

- tests can affect later tests in the same package;
- exact-value assertions are awkward;
- multiple app fixtures share telemetry state;
- flaky behavior can appear when future tests are parallelized;
- isolated manual metric readers cannot be used.

This confirms the need for a DI-owned telemetry registry.

Severity: **HIGH for observability tests**.

### Q12 — Timing-based tests contain avoidable flakiness

The suite contains fixed sleeps and polling loops.

Examples:

- SSE tests sleep 50 ms before publishing;
- lifecycle/metrics tests poll every 10 ms;
- some tests reserve an ephemeral address by binding, closing and later rebinding.

The reserve-close-rebind pattern has a time-of-check/time-of-use window where another process can acquire the port.

Prefer:

- explicit synchronization channels/hooks;
- listener injection;
- direct readiness/done signals;
- condition loops only when no deterministic hook exists.

Severity: **MEDIUM/HIGH**.

### Q13 — Several tests assert brittle representation details

The Range test hardcodes:

```text
Content-Range: bytes 0-9/56
body: {"id":1,"n
```

A harmless JSON field/order change can fail the test even if Range semantics remain correct.

This can be appropriate for an explicit representation contract, but then the representation needs a golden/contract artifact. Otherwise the test should derive expected bytes from the same stable encoder contract without coupling to incidental formatting.

Similar drift risk exists in log-string substring assertions and exact free-text error details.

Severity: **MEDIUM**.

### Q14 — Configuration tests are environment-sensitive and incomplete

`TestLoad_Defaults` unsets only a small subset of environment variables. Existing host/CI values for DB, auth, CORS, proxy, limiter, outbound or security settings can alter the result.

Coverage is also missing for many current fields:

- PostgreSQL DSN/backend/pool/timeouts;
- queue and worker limits;
- retry upper bounds;
- JWT key parsing, issuer/audience and weak-secret guards;
- trusted proxy CIDRs;
- CORS header exposure;
- security-header configuration fidelity;
- invalid combinations across components.

Severity: **HIGH**.

### Q15 — CI does not verify generated or dependency state

Missing CI gates include:

- `go mod tidy` followed by `git diff --exit-code`;
- protobuf generation followed by diff check;
- OpenAPI generation/validation and breaking check;
- migration validation;
- clean-tree assertion after generation/tests;
- generated-code/tool version verification.

Severity: **HIGH**.

### Q16 — CI does not exercise deployment/integration topology

The current workflow does not verify:

- PostgreSQL-backed application startup;
- migrations;
- HTTPS and HTTP/2 in the real binary;
- direct gRPC listener security;
- OTel Collector outage/telemetry flush;
- Docker Compose stack;
- graceful shutdown under traffic.

These should not all run in every fast unit job, but at least one integration job is required.

Severity: **HIGH**.

### Q17 — Quality scripts are Windows-only

Local entrypoints are PowerShell scripts. CI duplicates the commands directly in YAML rather than invoking a shared cross-platform task.

Consequences:

- local Windows and CI Linux paths can drift;
- the PowerShell scripts themselves are not tested in CI;
- future generation/migration/observability commands may be duplicated.

A Makefile, Taskfile or small Go-based task command can provide one portable source of truth while preserving PowerShell wrappers if desired.

Severity: **MEDIUM**.

### Q18 — CI tool installation and workflow integrity can be hardened

The workflow pins tool versions, which is good, but:

- workflow and tools module are currently untracked;
- GitHub actions are referenced by mutable major tags rather than immutable commit SHAs;
- tools are installed on every run;
- no artifact is uploaded when a race/static/vulnerability check fails;
- no timeout is set per individual Go command;
- no repeat/shuffle mode is used for order-dependent tests.

These are hardening items rather than the first blockers.

Severity: **MEDIUM**.

### Q19 — Vulnerability scanning covers only the built application binary

`govulncheck -mode binary` is useful for the executable, but does not establish:

- Docker base-image vulnerability state;
- migration/tooling dependencies;
- test-only tooling;
- secret scanning;
- license/dependency policy;
- SBOM/provenance.

This is acceptable for the current learning stage but incomplete for a production-like supply-chain lab.

Severity: **MEDIUM/DEFERABLE**.

### Q20 — Previously discovered defects lack regression tests

The suite currently lacks committed regression tests for at least:

- first-committed HTTP status in recorder;
- queue depth across multiple instances;
- canceled-context enqueue semantics;
- repeated/timeout `WorkerPool.Stop`;
- terminal job CAS transitions;
- ordered job/SSE transitions;
- Profile trailing data/wrong user ID/body limit;
- retry-duration overflow;
- direct gRPC authentication/authorization;
- Accept `q` matrix;
- HEAD/Allow consistency;
- CORS exposed headers and denial `Vary`;
- host-wide HSTS and configured Referrer-Policy;
- readiness redaction;
- immediate SSE flush and post-commit errors;
- async crash/restart/reconciliation;
- telemetry exporter failure and final flush.

Severity: **CRITICAL/HIGH**.

## Test-tier assessment

| Tier | Current state | Main gap |
|---|---|---|
| Pure unit tests | PARTIAL | critical queue/worker/security/service/parser packages missing |
| Handler/middleware tests | PRESENT/PARTIAL | good breadth; edge/contract matrix incomplete |
| In-memory HTTP integration | PRESENT/PARTIAL | testkit differs from production and auto-injects admin auth |
| Lifecycle tests | PARTIAL | selected HTTP/TLS shutdown only; fatal gRPC/forced/global budget absent |
| PostgreSQL integration | MISSING | no DB/migration/repository tests |
| gRPC integration | PARTIAL | bufconn unary basics only; no security/TLS/lifecycle/streaming |
| SSE integration | PARTIAL/BROKEN | manual Hub publish bypasses real producer topology |
| Contract/OpenAPI | MISSING | no machine-readable source or runtime validation |
| Fuzz/property tests | MISSING | zero targets |
| Benchmarks/load | MISSING | zero committed benchmarks/workloads |
| Race/concurrency proof | PARTIAL/UNKNOWN | CI intent exists, critical paths unexercised |
| Fault/chaos tests | MISSING/PARTIAL | basic outbound timeout only |
| Supply-chain/deployment checks | PARTIAL | binary vuln scan only; no image/SBOM/secret policy |

## Required remediation groups

These are groups, not replacements for the individual Q findings.

1. **QA-REPO-REPRO** — commit CI/tooling/generated policy and make clean checkout reproducible.
2. **QA-CODEGEN** — pin protobuf/OpenAPI generators and enforce generate-diff checks.
3. **QA-TESTKIT** — align fixtures with production topology, use one Hub and remove implicit admin/auth masking.
4. **QA-CONCURRENCY** — add deterministic queue/worker/job/Hub lifecycle and race regression suites.
5. **QA-POSTGRES** — add disposable PostgreSQL migration/repository/transaction integration tests.
6. **QA-CONTRACT** — add OpenAPI runtime conformance, protocol matrices and breaking checks.
7. **QA-FUZZ** — add parser/header/query/state-transition fuzz/property tests.
8. **QA-BENCH-LOAD** — add benchmarks, load profiles and telemetry-overhead experiments.
9. **QA-TELEMETRY-TESTS** — isolate metrics/traces/logs and test propagation/exporter failure/shutdown.
10. **QA-CI-GATES** — add tidy/generate/migration/coverage/integration gates and portable entrypoints.
11. **QA-FLAKE** — replace fixed sleeps/port reservation with deterministic synchronization/listener injection.
12. **QA-REGRESSION** — convert every critical/high audit finding into a failing test before or alongside its fix.

## Minimum quality gate sequence

Fast pull-request lane:

```text
gofmt/goimports check
go mod tidy diff check
protobuf/OpenAPI generate diff check
go test -short ./...
go vet ./...
staticcheck ./...
selected race tests for concurrency packages
OpenAPI validation and contract tests
```

Integration lane:

```text
start disposable PostgreSQL
apply migrations
run SQLX/repository/full-app integration tests
run full go test -race ./...
run TLS/gRPC/SSE/shutdown scenarios
run govulncheck
```

Scheduled/manual laboratory lane:

```text
fuzz campaigns
benchmarks
load/fault/chaos scenarios
telemetry Collector outage
profile capture
container/SBOM/security scans
```

## Traceability rule carried into Audit 14

Every Q finding above and every finding from Audits 01–12 must appear in the final gap matrix with:

- a stable finding ID;
- one remediation group;
- one or more bounded agent task IDs;
- a verification test/command;
- an explicit final disposition.

The shorter remediation list is therefore not allowed to hide or discard individual findings.

## Final status

| Capability | Status |
|---|---|
| Functional HTTP test breadth | PRESENT/PARTIAL |
| Production-faithful test fixture | BROKEN/PARTIAL |
| Critical concurrency tests | MISSING/BROKEN |
| PostgreSQL/migration tests | MISSING |
| OpenAPI contract tests | MISSING |
| Fuzz tests | MISSING |
| Benchmarks/load scripts | MISSING |
| Coverage policy | MISSING |
| Race CI intent | PRESENT but untracked/unproven |
| Static analysis intent | PRESENT but untracked/unproven |
| Vulnerability scan intent | PRESENT/PARTIAL but untracked |
| Reproducible protobuf generation | BROKEN |
| Clean-checkout CI reproducibility | BROKEN |
