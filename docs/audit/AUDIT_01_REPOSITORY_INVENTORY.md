# Audit 01 — Repository Inventory

## Scope

This pass inventories the uploaded repository snapshot only. It does not yet perform a deep correctness review of request flows, observability, concurrency, resilience, security, or contracts.

Snapshot:

- Repository: `pet-study`
- Branch: `main`
- HEAD: `9ea4389` (`feature: add tests and patch README.md`)
- Declared module: `pet-study`
- Declared Go version: `go 1.25.0`
- Declared toolchain: `go1.25.8`

## Verification performed

- ZIP path traversal check: passed (`678` archive entries, no unsafe paths).
- Go source syntax parse: passed (`119` Go files, `0` syntax errors).
- `gofmt -l cmd internal`: no output.
- `git diff --check`: passed.
- Full `go test ./...`: **not executed successfully in this environment**.
  - Local Go is `1.23.2`.
  - The module requests `go1.25.8`.
  - The sandbox cannot download the toolchain because outbound network access is unavailable.

Therefore, test/build/CI claims remain **unverified by this audit pass**.

## Repository size and composition

Excluding `.git`:

- Files: `160`
- Git-tracked files: `130`
- Untracked, non-ignored files: `20`
- Ignored files present in the archive: `10`
- Go files: `119`
- Test files: `27`
- Non-test Go LOC: approximately `8,612`
- Test Go LOC: approximately `3,922`

The archive also includes the complete `.git` directory.

## Main components

### Entry point and lifecycle

- `cmd/api/main.go`
  - composition root;
  - configuration loading;
  - repository selection (`memory` / `postgres`);
  - HTTP/HTTPS, gRPC, queue, worker pool, outbound client, security and middleware wiring.
- `internal/api/server.go`
  - HTTP and optional HTTPS lifecycle;
  - readiness changes;
  - graceful shutdown;
  - queue/pool, stream hub and gRPC shutdown integration.

### Configuration

- `internal/config/config.go`
  - HTTP/TLS;
  - gRPC;
  - streaming;
  - PostgreSQL/storage backend;
  - worker pool and queue;
  - rate limiting and bulkhead;
  - outbound client/retries;
  - JWT, CORS, trusted proxies and security headers.

### Domain and application

- `internal/entity`
  - users, jobs and profile models.
- `internal/service`
  - user, job and profile application services;
  - repository interfaces.
- `internal/apperr`
  - shared application/service error type.

### Persistence

- In-memory repositories:
  - `internal/store/userrepo/memory_user_repo.go`
  - `internal/store/jobrepo/memory_job_repo.go`
- PostgreSQL/SQLX repositories:
  - `internal/store/userrepo/sqlx.go`
  - `internal/store/jobrepo/sqlx.go`
- DB helpers:
  - `internal/db/open.go`
  - `internal/db/stats.go`
  - `internal/db/tx.go`
- Migrations:
  - `migrations/000001_create_users.*.sql`
  - `migrations/000002_create_jobs.*.sql`
- Local PostgreSQL compose file:
  - `docker-compose.yml`

### HTTP/API

- `internal/router`
  - root/API/health/debug muxes;
  - `http.ServeMux` Go 1.22+ patterns;
  - Problem+JSON 404 handling.
- `internal/routes`
  - v1/v2 user handlers;
  - jobs, SSE, gRPC bridge;
  - profile and Range export.
- `internal/httputils`
  - AppHandler adapter;
  - strict JSON parsing;
  - error mapping;
  - Problem+JSON;
  - response/content negotiation helpers.

### Concurrency and async processing

- `internal/queue`
  - bounded channel queue;
  - stop-accepting policy.
- `internal/workerpool`
  - worker lifecycle and job execution.
- `internal/stream`
  - SSE event hub and subscriber buffers.

### Security

- `internal/security`
  - JWT HS256 verification;
  - principal/request information;
  - authorization policy and errors.
- `internal/middleware`
  - authentication/RBAC;
  - CORS;
  - trusted proxy handling;
  - security headers;
  - rate limiting and bulkhead;
  - logging/recovery/metrics.

### Outbound integration

- `internal/outbound`
  - Profile service client;
  - instrumentation;
  - retries/backoff/jitter.
- `internal/outbound/httpclient`
  - custom `http.Client` / `Transport` construction.

### Protocols

- `internal/transport/grpcserver`
  - gRPC runtime and Jobs service.
- `internal/transport/grpcclient`
  - internal Jobs client.
- `internal/transport/pb`
  - `job.proto`, `user.proto` and locally generated Go files.

### Runtime diagnostics and telemetry baseline

- Structured logging via `log/slog` with text handler.
- Metrics via `expvar`:
  - HTTP;
  - jobs/queue;
  - outbound;
  - security;
  - SSE;
  - DB pool.
- Debug endpoints:
  - `/debug/vars`;
  - `/debug/runtime`;
  - `/debug/pprof/*`.
- Request correlation through request ID.
- No OpenTelemetry SDK/exporter detected.
- No Prometheus client/exporter detected.
- No Grafana/Tempo/Loki/OTel Collector configuration detected.

## Actual route surface found in code

API:

- `GET /api/v1/users`
- `POST /api/v1/users`
- `GET /api/v1/users/{id}`
- `GET /api/v1/users/{id}/profile`
- `GET /api/v1/users/{id}/export`
- `GET /api/v1/jobs/{id}`
- `GET /api/v1/jobs/{id}/grpc`
- `GET /api/v1/jobs/{id}/events`
- `GET /api/v2/users`
- `POST /api/v2/users`

Operational:

- `GET /livez`
- `GET /readyz`
- optional `/debug/*`

Important inventory note: there is no `GET /api/v2/users/{id}` registration in the uploaded `internal/router/router.go`.

## gRPC surface

`internal/transport/pb/job.proto` defines:

- `JobsService.GetJob` — unary RPC.

No gRPC streaming method is present in the current proto contract.

## Tooling and automation found in the uploaded snapshot

- `.github/workflows/ci.yml`
  - format;
  - test;
  - vet;
  - race;
  - staticcheck;
  - binary `govulncheck`.
- PowerShell scripts under `scripts/`.
- Tool pinning module under `tools/`.
- `docker-compose.yml` contains PostgreSQL only.

No Makefile, Taskfile or equivalent cross-platform command entrypoint was found.

## Critical repository-state findings

### R1 — Clean checkout is not reproducible

Severity: **HIGH**

Evidence:

- Current `HEAD` imports and uses `pet-study/internal/transport/pb` from many tracked Go files.
- At `HEAD`, only these files are tracked under that directory:
  - `internal/transport/pb/job.proto`
  - `internal/transport/pb/user.proto`
- The required generated files are untracked:
  - `job.pb.go`
  - `job_grpc.pb.go`
  - `user.pb.go`
- There is no committed generation step in the tracked repository state.

Consequence: a clean checkout of `HEAD` cannot reproduce the uploaded source tree and is expected to fail compilation unless code generation is run manually with undocumented external state.

### R2 — CI and local quality tooling are not committed

Severity: **HIGH**

The following are present in the archive but untracked:

- `.github/workflows/ci.yml`
- `scripts/`
- `tools/`

The README states that CI and these tools exist, but they are absent from the current committed `HEAD`.

### R3 — Uploaded project contains a large uncommitted persistence change

Severity: **HIGH for audit traceability**

The working tree contains roughly `959` added lines and multiple staged/unstaged changes introducing:

- PostgreSQL configuration;
- SQLX repositories;
- migrations;
- DB pool metrics;
- compose configuration;
- storage backend switching.

The uploaded state is therefore not represented by a single Git commit. Future audits must explicitly treat the uploaded working tree—not `HEAD`—as the source of truth until this is resolved.

### R4 — Archive contains sensitive/local artifacts

Severity: **MEDIUM**

Ignored files included in the archive:

- `certs/localhost-key.pem` — private key material;
- `certs/localhost.pem`;
- `server.log`;
- `.idea/*`.

Also present:

- empty accidental file `-H`;
- local request payload files `req.json`, `req-async.json`.

These should be excluded from future audit/share archives. The private key should be treated as disposable and rotated if used anywhere beyond local development.

### R5 — README/config drift around storage defaults

Severity: **MEDIUM**

Code defaults (`internal/config/config.go`):

- `StorageBackend: "postgres"`
- non-empty local PostgreSQL DSN.

README configuration table says:

- `DB_DSN` default is empty;
- Step 3 uses in-memory storage.

This affects out-of-the-box startup expectations and must be reconciled.

### R6 — Current context/documentation and actual route surface differ

Severity: **MEDIUM**

The actual router exposes only v2 collection routes. Some prior project summaries described a v2 item route. This confirms the need for code-derived contract inventory rather than relying on chat summaries.

### R7 — Full automated verification unavailable in this audit environment

Severity: **BLOCKER FOR TEST CONFIDENCE, not a source defect**

Only syntax/format checks were performed. Build, tests, vet, race, staticcheck and vulnerability scanning remain unverified in this pass.

## Testing inventory

Packages with meaningful existing tests include:

- API server lifecycle;
- config;
- middleware/security/metrics;
- request ID;
- router/health/authz;
- routes including jobs, negotiation, ETag, SSE and Range;
- outbound client/retries;
- in-memory user repository;
- gRPC server;
- runtime diagnostics.

Notable packages without direct tests in the uploaded tree include:

- PostgreSQL DB helpers;
- SQLX user repository;
- SQLX job repository;
- queue implementation;
- worker pool;
- stream hub;
- service layer;
- metrics packages themselves.

This is an inventory observation; test adequacy will be evaluated in later focused passes.

## Initial gap classification relevant to the final roadmap

Already present at baseline:

- `slog` logging;
- request ID;
- `expvar` metrics;
- runtime/GC snapshot;
- pprof;
- HTTP, gRPC, SSE, async queue, retries, auth and DB-pool instrumentation foundations.

Not detected:

- OpenTelemetry tracing/metrics/log pipeline;
- trace context propagation;
- Prometheus endpoint/exporter;
- OTel Collector;
- Grafana dashboards;
- Tempo/Loki configuration;
- SLI/SLO definitions;
- alert rules;
- OpenAPI specification/codegen;
- contract validation tooling;
- fuzz and benchmark suite as a systematic final quality block.

## Audit 01 conclusion

The project is structurally substantial and already contains most features described through Step 7 plus an additional PostgreSQL persistence WIP. However, before deep subsystem analysis, repository reproducibility must be treated as a first-class risk: the uploaded working tree, committed `HEAD`, README and automation files are not aligned.

No code changes should be made yet. The next pass should build the execution/lifecycle map from the uploaded working tree.
