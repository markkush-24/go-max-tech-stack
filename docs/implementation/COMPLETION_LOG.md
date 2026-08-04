# Completion Log

Append one entry only after review acceptance.

```text
Task: TASK-XXX
Accepted: YYYY-MM-DD
Commit: <hash>
Reviewer: <name/chat>
Verification:
- ...
Notes:
- ...
Unlocked tasks:
- ...
```

## P0 Provenance Notes

- Dependency security remediation: pgx and affected transitive modules were updated after reproduced `govulncheck` findings so binary vulnerability scans could pass on the Go 1.25.12 baseline. The current source state records `github.com/jackc/pgx/v5 v5.9.2` and `golang.org/x/net v0.53.0`.
- Manual cleanup: removal of empty or unused `internal/db` files was accepted as repository cleanup outside the original task-card scopes. The remaining `internal/db` package owns database open/stats helpers only.
- TASK-006 telemetry validation: telemetry-specific config validation was not applicable because `internal/config` has no telemetry configuration yet. Hostile telemetry/runtime environment values were covered only as unrelated environment input.

```text
Task: TASK-001
Accepted: 2026-07-19
Commit: 4d89e258
Reviewer: Mark
Verification:
- go test ./internal/transport/...: PASS
- go test ./...: PASS
- .\scripts\check-format.ps1: PASS
- go vet ./...: PASS
- .\scripts\run-staticcheck.ps1: PASS
- .\scripts\run-race.ps1: PASS; local script skipped race because CGO_ENABLED=0
- .\scripts\run-govulncheck.ps1: FAIL; reported existing vulnerabilities in Go 1.25.8 stdlib and github.com/jackc/pgx/v5@v5.9.1
- git ls-files .github scripts tools internal/transport/pb: PASS
- git status --short: PASS; command ran and showed pre-existing unrelated repository changes
- git diff --check: PASS
- git diff --cached --check -- .github scripts tools internal/transport/pb: PASS
- git diff --cached --check: FAIL; pre-existing staged audit file outside TASK-001 had a blank line at EOF
Notes:
- Tracked CI workflow, PowerShell quality scripts, tools module, and generated protobuf/gRPC Go files.
- Documented generated-artifact policy: internal/transport/pb/*.pb.go files are committed; internal/transport/pb/*.proto files are canonical sources; normal builds do not run generation.
- Remaining limitations: vulnerability remediation and unrelated dirty-tree cleanup were outside TASK-001.
Unlocked tasks:
- TASK-002
```

```text
Task: TASK-002
Accepted: 2026-07-19
Commit: ab19e5c9
Reviewer: Mark
Verification:
- go mod tidy: PASS; run in tools module
- .\scripts\install-tools.ps1: PASS
- go test ./scripts/...: PASS
- go generate ./...: PASS
- go generate ./...: PASS; second consecutive run produced no generated drift
- git diff --name-status -- internal/transport/pb: PASS; no generated protobuf diff
- go test ./...: PASS
- go vet ./...: PASS
- git diff --exit-code: PASS after staging task changes
- git diff --cached --check: PASS
- staged diff scan for @latest, local paths and obvious secrets: PASS
Notes:
- Pinned protobuf codegen tools and quality tools through tools/go.mod.
- Added canonical cross-platform protobuf generation entrypoint via go generate ./....
- Documented required external protoc version: libprotoc 34.0.
- Remaining limitation: developers still need external protoc installed separately.
Unlocked tasks:
- TASK-003
```

```text
Task: TASK-003
Accepted: 2026-07-19
Commit: 5f37f34d
Reviewer: Mark
Verification:
- go test ./scripts/...: PASS
- .\scripts\check-format.ps1: PASS
- go mod tidy: PASS
- go generate ./...: PASS
- go build ./cmd/api: PASS
- go test ./...: PASS
- go vet ./...: PASS
- git clean -xfd && git reset --hard HEAD: PASS; run only in disposable clone
- pwsh -File .\scripts\check-clean-checkout.ps1: PASS; run in disposable clone
- missing generated file simulation in disposable clone: PASS; gate failed on untracked generated file
- missing root tidy dependency simulation in disposable clone: PASS; gate failed on git diff --exit-code
- git diff --exit-code: PASS; run in disposable clone
- git diff --exit-code: FAIL; current workspace had unrelated tools/tools.go deletion outside TASK-003
- git diff --cached --check: PASS
Notes:
- Added CI clean-checkout drift gate for tidy, generation, formatting, build, tests and clean-tree verification.
- Added shared scripts/check-clean-checkout.ps1 and expanded format checks to scripts.
- Remaining limitations: existing govulncheck behavior is unchanged; unrelated local tools/tools.go deletion was outside TASK-003.
Unlocked tasks:
- TASK-007
```

```text
Task: TASK-004
Accepted: 2026-07-19
Commit: 86e577b8
Reviewer: Mark
Verification:
- go test ./scripts/...: PASS
- .\scripts\export-archive.ps1 -Force: PASS
- git check-ignore certs/localhost-key.pem server.log .idea || true: PASS
- find . -maxdepth 2 -type f | sort: FAIL; PowerShell resolved Windows find.exe, which does not support POSIX flags
- C:\Program Files\Git\usr\bin\find.exe . -maxdepth 2 -type f | C:\Program Files\Git\usr\bin\sort.exe: PASS
- disposable clone export from committed patch: PASS
- negative check with tracked server.log: PASS; export failed as expected
- negative check with fake private-key marker: PASS; export failed as expected
- go test ./...: PASS
- git diff --cached --check: PASS
Notes:
- Added repository hygiene ignores, safe git archive export, archive path checks and private-key marker checks.
- Documented repeatable export flow and local-only development certificate/key generation.
- Remaining limitation: exact POSIX find command was not runnable in PowerShell; Git for Windows find.exe equivalent passed.
- Existing unrelated tools/tools.go deletion was outside TASK-004.
Unlocked tasks:
- none
```

```text
Task: TASK-005
Accepted: 2026-07-19
Commit: 70741d32
Reviewer: Mark
Verification:
- go test ./internal/config/... ./internal/router/...: PASS
- git grep -n "STORAGE_BACKEND\|DB_DSN" README.md internal/config: PASS
- go test ./...: PASS
- git diff --check -- README.md docs/security-policy.md internal/config/config_test.go internal/router/router_test.go: PASS
Notes:
- Documented the canonical default storage backend as STORAGE_BACKEND=postgres and aligned README DB_DSN/defaults with internal/config.
- Updated documented HTTP routes and optional integration conditions to match current router/main behavior.
- Added regression checks for default storage config and the absent v2 item route.
- Remaining limitation: PostgreSQL schema application is still manual; cmd/api has no built-in migration runner.
- Existing unrelated tools/tools.go deletion was outside TASK-005.
Unlocked tasks:
- TASK-018
- TASK-022
```

```text
Task: TASK-006
Accepted: 2026-07-19
Commit: 5fb87cf8
Reviewer: Mark
Verification:
- go test ./internal/config/...: PASS
- hostile env + go test -count=20 ./internal/config/...: PASS
- go test -count=20 ./internal/config/...: PASS
- go test ./...: PASS
- go test -shuffle=on -count=20 ./internal/config/...: PASS
- git diff --check -- internal/config/config_test.go: PASS
Notes:
- Reworked config tests so each test controls the full config.Load env surface.
- Added explicit default, valid-env and table-driven invalid-env coverage for duration, integer, URL, JWT, CORS, proxy, DB, TLS/gRPC and security-header validation.
- Remaining limitation: no telemetry config exists in internal/config; hostile telemetry/runtime env values are covered as unrelated env, not parsed config.
- Existing unrelated tools/tools.go deletion was outside TASK-006.
Unlocked tasks:
- none
```

```text
Task: TASK-007
Accepted: 2026-07-19
Commit: 915f8e8f
Reviewer: Mark
Verification:
- go version: PASS; go1.25.12 windows/amd64
- go test ./...: PASS
- go vet ./...: PASS
- go test -race ./...: FAIL; local CGO_ENABLED=0 rejects -race
- CGO_ENABLED=1 go test -race ./...: FAIL; local runtime/cgo could not find gcc in PATH
- go build -o .artifacts/pet-study.exe ./cmd/api: PASS
- govulncheck -version: PASS; govulncheck@v1.1.4
- govulncheck -mode binary .artifacts/pet-study.exe: PASS; 0 affected vulnerabilities
- go build -o .artifacts/pet-study ./cmd/api: PASS
- govulncheck -mode binary .artifacts/pet-study: PASS; 0 affected vulnerabilities
- staticcheck ./...: PASS
- gofmt -l cmd internal scripts: PASS
- git diff --check: PASS
Notes:
- Updated the project verification baseline from go1.25.8 to go1.25.12 so binary govulncheck no longer fails on Go standard-library vulnerabilities.
- Aligned go.mod, GitHub Actions and docs with the go1.25.12 baseline and recorded commands/results in docs/implementation/TOOLCHAIN_BASELINE.md.
- Remaining limitation: local Windows race tests need a C compiler; scripts/run-govulncheck.ps1 still pins go1.25.8 because scripts/** was outside TASK-007 allowed scope.
- Existing unrelated tools/tools.go deletion was outside TASK-007.
Unlocked tasks:
- TASK-008
- TASK-009
- TASK-013
- TASK-015
- TASK-078
```

```text
Task: P0-CORR-001
Accepted: 2026-07-20
Commit: 74b89e1c
Reviewer: Mark
Verification:
- .\scripts\install-tools.ps1: PASS
- go test ./scripts/...: PASS
- go -C tools mod tidy: PASS
- go generate ./...: PASS
- go generate ./...: PASS; second consecutive run produced no generated drift
- git diff --name-status -- internal/transport/pb: PASS; no generated protobuf diff
- pwsh -File ./scripts/check-clean-checkout.ps1: PASS; run in a disposable clean clone with the correction patch applied
- stale tools/go.mod mutation in disposable clone: PASS; clean-checkout gate failed as expected
- .\scripts\run-govulncheck.ps1: PASS; built with go1.25.12 and reported 0 affected vulnerabilities
- go test ./...: PASS
- go vet ./...: PASS
- staticcheck ./...: PASS
- gofmt -l cmd internal scripts: PASS
- git diff --check: PASS
- git diff --exit-code: FAIL; the correction diff was intentionally present in the main workspace before commit
- go test -race ./...: FAIL; local CGO_ENABLED=0 rejects -race
- CGO_ENABLED=1 go test -race ./...: FAIL; local runtime/cgo could not find gcc in PATH
Notes:
- Replaced substring tool-version checks with exact normalized token comparisons and post-install verification.
- Added nested tools module tidy to the clean-checkout gate and verified stale tools/go.mod drift fails the gate.
- Removed the local go1.25.8 govulncheck override so local binary scans use the repository-selected go1.25.12 baseline.
- Remaining limitations: local Windows race tests need a C compiler; historical documentation still mentions go1.25.8 as prior context; P0-CORR-002 and P0-CORR-003 remain unimplemented.
Unlocked tasks:
- P0-CORR-002
- P0-CORR-003
```

```text
Task: P0-CORR-002
Accepted: 2026-07-20
Commit: 47e054c1
Reviewer: Mark
Verification:
- go test -run ExportArchive -v ./scripts: PASS
- pwsh -File .\scripts\export-archive.ps1 -Force: PASS
- go test ./scripts/...: PASS
- go test ./...: PASS
- git diff --check: PASS
- git check-ignore -v secret.ppk secret.p12 secret.pfx project.iml .DS_Store Thumbs.db notes.swp: PASS
- intermediate go test ./scripts/...: FAIL then fixed; exposed top-level dotfile normalization issue for .DS_Store
- intermediate pwsh -File .\scripts\export-archive.ps1 -Force: FAIL then fixed; File.Replace needed explicit backup path
Notes:
- Export now writes to a temporary ZIP, validates paths/content, then publishes the final archive only after validation succeeds.
- Blocked archive paths now include *.ppk, *.p12, *.pfx, *.iml, .DS_Store, Thumbs.db and *.swp in addition to the existing local artifact rules.
- Large text entries are scanned incrementally for PEM private-key markers instead of being skipped over 1 MiB.
- Remaining limitation: this is targeted archive hardening, not a generic secret scanner.
Unlocked tasks:
- none
```

```text
Task: P0-CORR-003
Accepted: 2026-07-20
Commit: 6bd20117
Reviewer: Mark
Verification:
- go test ./internal/config/... ./internal/router/...: PASS
- go test -count=20 ./internal/config/...: PASS
- go test ./...: PASS
- git grep -n "Authorization: Bearer" README.md: PASS
- git grep -n "migrations" README.md docs: PASS
- git diff --check: PASS
- docker compose config --services: PASS
- README vs validConfigEnv comparison script: PASS; 58/58 documented
- docker-compose.yml service/env static check: PASS
Notes:
- Completed operational README documentation for config env vars, protected API examples and local PostgreSQL migration workflow.
- Added repository-owned P0 provenance notes for dependency security remediation, manual internal/db cleanup and TASK-006 telemetry N/A scope.
- Remaining limitation: migration commands were statically verified against Docker Compose names but not run against a live database in this task.
Unlocked tasks:
- none
```

```text
Task: P0-CORR-004
Accepted: 2026-07-20
Commit: a80f307a
Reviewer: Mark
Verification:
- go test -count=1 -run ExportArchive -v ./scripts: PASS
- go test -count=1 ./internal/config/...: PASS
- go test ./scripts/... ./internal/config/...: PASS
- go test ./...: PASS
- go vet ./...: PASS
- go test -race ./...: FAIL; local CGO_ENABLED=0 rejects -race
- CGO_ENABLED=1 go test -race ./...: FAIL; local runtime/cgo could not find gcc in PATH
- pwsh -File ./scripts/check-clean-checkout.ps1: FAIL; active workspace had the intentional uncommitted correction diff
- pwsh -File ./scripts/check-clean-checkout.ps1: PASS; run in a disposable clean clone with a temporary correction commit
- git diff --check: PASS
- git status --short: PASS
- staticcheck ./...: PASS
- go build -o .artifacts/pet-study ./cmd/api: PASS
- govulncheck -mode binary .artifacts/pet-study: PASS
- go test -count=1 -run TestReadmeConfigEnvironmentTableExactSet -v ./internal/config: PASS
- go test -count=1 -run TestCleanCheckoutGateFailsWhenToolsModuleDrifts -v ./scripts: PASS
- README mutation scenarios in C:\tmp: PASS; missing, extra and duplicate ENV rows failed as expected
Notes:
- Added committed regression guards for archive export success, reproducibility, blocked paths, private-key marker rejection and failed-publish atomicity.
- Added README/config exact-set validation for the 58-key config environment table and corrected PowerShell examples that used CMD caret continuations.
- Remaining limitation: local Windows race tests require CGO plus a C compiler; clean-checkout gate only passes from a committed clean tree, so it was verified in a disposable clone.
Unlocked tasks:
- none
```

```text
Task: TASK-008
Accepted: 2026-07-20
Commit: e62d6d04
Reviewer: Mark
Verification:
- go test ./internal/middleware/...: PASS
- go test -count=1 -run "TestStatusRecorder|TestMetricsAndLogger_RecordFirstWireStatus" -v ./internal/middleware: PASS
- go test -count=1 ./internal/middleware/...: PASS
- go test ./...: PASS
- go vet ./internal/middleware/...: PASS
- go test -race ./internal/middleware/...: FAIL; local CGO_ENABLED=0 rejects -race
- CGO_ENABLED=1 go test -race ./internal/middleware/...: FAIL; local runtime/cgo could not find gcc in PATH
- git diff --check: PASS
Notes:
- Fixed response status recording so the first committed status wins and implicit Write records 200.
- Added recorder and Logger+Metrics regressions proving wire status, logged status and metric status stay aligned.
- Remaining limitation: local Windows race verification requires CGO plus a C compiler.
Unlocked tasks:
- none
```

```text
Task: TASK-009
Accepted: 2026-07-20
Commit: c7e8632f
Reviewer: Mark
Verification:
- go test -count=1 ./internal/entity/... ./internal/service/...: PASS
- go test ./internal/entity/... ./internal/service/...: PASS
- go test ./...: PASS
- go vet ./internal/entity/... ./internal/service/...: PASS
- git diff --check: PASS
Notes:
- Defined explicit Job transition intents and terminal states.
- Added typed transition-conflict errors and service-level validation before repository transition calls.
- Remaining limitation: concrete repository CAS/expected-source-state enforcement is intentionally left for TASK-010.
Unlocked tasks:
- TASK-010
- TASK-011
```

```text
Task: TASK-010
Accepted: 2026-07-20
Commit: a9cf8383
Reviewer: Mark
Verification:
- go test -count=1 -v ./internal/store/jobrepo/...: PASS
- go test ./internal/store/jobrepo/...: PASS
- go test -race ./internal/store/jobrepo/...: FAIL; local CGO_ENABLED=0 rejects -race
- go test ./...: PASS
- git diff --check: PASS
- docker compose ps: FAIL; Docker daemon was not running
Notes:
- Enforced CAS Job transitions in memory repository under the write lock.
- Added SQL status-guarded transition updates and conflict-vs-not-found handling for SQLX job repository.
- Added shared memory and SQLX CAS tests for concurrent terminal transitions, immutable terminal states, missing jobs and FailActive terminal preservation.
- Remaining limitations: local Windows race verification requires CGO plus a C compiler; live PostgreSQL integration was not run because Docker daemon was unavailable.
Unlocked tasks:
- none
```

```text
Task: TASK-011
Accepted: 2026-07-20
Commit: 9ed6f131
Reviewer: Mark
Verification:
- go test -count=1 -v ./internal/workerpool/...: PASS
- go test ./internal/workerpool/...: PASS
- go test -race ./internal/workerpool/...: FAIL; local CGO_ENABLED=0 rejects -race
- CGO_ENABLED=1 go test -race ./internal/workerpool/...: FAIL; local runtime/cgo could not find gcc in PATH
- go test ./...: PASS
- go vet ./internal/workerpool/...: PASS
- git diff --check HEAD: PASS
- git diff --cached --check: PASS
Notes:
- Reworked WorkerPool lifecycle around immutable per-start generations.
- Removed shared mutable worker context and pool-level WaitGroup reuse.
- Prevented restart while a prior generation is still stopping after a timed-out Stop.
- Added regression tests for timeout-stop, fresh generation restart, concurrent Stop and idempotent concurrent Start.
- Remaining limitation: local Windows race verification requires CGO plus a C compiler.
Unlocked tasks:
- TASK-012
```

```text
Task: TASK-012
Accepted: 2026-07-21
Commit: 4d6b731b
Reviewer: Mark
Verification:
- go test -count=1 -v ./internal/workerpool/...: PASS
- go test ./internal/workerpool/... ./internal/store/jobrepo/...: PASS
- go test ./...: PASS
- go vet ./internal/workerpool/... ./internal/store/jobrepo/...: PASS
- go test -race ./internal/workerpool/... ./internal/store/jobrepo/...: FAIL; local CGO_ENABLED=0 rejects -race
- CGO_ENABLED=1 go test -race ./internal/workerpool/... ./internal/store/jobrepo/...: FAIL; local runtime/cgo could not find gcc in PATH
- git diff --check: PASS
Notes:
- Made WorkerPool Stop completion idempotent with cached StopOutcome.
- Removed the generation waiter goroutine by closing generation done from worker completion accounting.
- Split worker wait from terminal repair using detached bounded repair contexts.
- Replaced unbounded context.Background terminal failure writes with bounded repair context.
- Remaining limitation: local Windows race verification requires CGO plus a C compiler.
Unlocked tasks:
- none
```

```text
Task: TASK-013
Accepted: 2026-07-21
Commit: 98d08cca
Reviewer: Mark
Verification:
- go test -count=1 -v ./internal/queue/...: PASS
- go test -count=1 -run "TestDeleteJobAfterFailedEnqueueUsesDetachedBoundedContext|TestQueueOverflowFastFail" -v ./internal/routes: PASS
- go test ./internal/queue/... ./internal/routes/...: PASS
- go test ./...: PASS
- go vet ./internal/queue/... ./internal/routes/...: PASS
- go test -race ./internal/queue/...: FAIL; local CGO_ENABLED=0 rejects -race
- CGO_ENABLED=1 go test -race ./internal/queue/...: FAIL; local runtime/cgo could not find gcc in PATH
- git diff --check: PASS
Notes:
- Defined deterministic Queue admission semantics with canceled-context rejection before send.
- Documented StopAccepting as a strict post-return admission barrier without closing the producer channel.
- Added queue cancellation/barrier/no-close regressions.
- Made async enqueue rollback in v1/v2 users handlers use a detached bounded context.
- Remaining limitation: local Windows race verification requires CGO plus a C compiler.
Unlocked tasks:
- TASK-014
```

```text
Task: TASK-014
Accepted: 2026-07-21
Commit: 42359c14
Reviewer: Mark
Verification:
- gofmt -w internal/routes/users_handler.go internal/routes/users_handler_v2.go internal/testkit/testkit.go internal/stream/stream.go internal/stream/stream_test.go internal/routes/async_event_topology_test.go: PASS
- go test -count=1 ./internal/routes -run TestAsyncCreatePublishesMonotonicEventsToJobSSE: PASS
- go test -count=1 ./internal/stream -run TestHub: PASS
- go test -count=1 ./internal/routes/... ./internal/stream/... ./internal/testkit/...: PASS
- go test -race ./internal/routes/... ./internal/stream/... ./internal/testkit/...: FAIL; local CGO_ENABLED=0 rejects -race
- CGO_ENABLED=1 go test -race ./internal/routes/... ./internal/stream/... ./internal/testkit/...: FAIL; local runtime/cgo could not find gcc in PATH
- go test ./...: PASS
- staticcheck ./...: PASS
- go vet ./internal/routes/... ./internal/stream/... ./internal/testkit/...: PASS
- git diff --check: PASS
Notes:
- Corrected async event ordering so queued is published before a work item becomes visible to workers.
- Aligned testkit with production topology by using one Event Hub for async producers and SSE consumers.
- Added bounded Event Hub replay and full async-create-to-SSE coverage without manual publish.
- Remaining limitation: local Windows race verification requires CGO plus a C compiler.
- Remaining limitation: if Enqueue fails after durable Job Save, a short-lived queued event/count may exist for a job that is then rolled back.
Unlocked tasks:
- none
```

```text
Task: TASK-015
Accepted: 2026-07-21
Commit: 68c8899d
Reviewer: Mark
Verification:
- gofmt -w internal/transport/grpcserver/runtime.go internal/transport/grpcserver/runtime_test.go: PASS
- go test -count=1 -run Runtime -v ./internal/transport/grpcserver: PASS
- go test -count=1 ./internal/transport/grpcserver/...: PASS
- go test -race ./internal/transport/grpcserver/...: FAIL; local CGO_ENABLED=0 rejects -race
- CGO_ENABLED=1 go test -race ./internal/transport/grpcserver/...: FAIL; local runtime/cgo could not find gcc in PATH
- go test ./...: PASS
- go vet ./internal/transport/grpcserver/...: PASS
- staticcheck ./...: PASS
- git diff --check: PASS
Notes:
- Added explicit gRPC runtime state, single-use Start and idempotent Shutdown with one GracefulStop attempt plus one forced Stop fallback.
- Added Done and Err so the application owner can observe fatal Serve failures.
- Added runtime tests for bind failure, fatal Serve error, repeated Start and forced/repeated Shutdown.
- Remaining limitation: local Windows race verification requires CGO plus a C compiler.
- Remaining limitation: cmd/api does not yet consume Done/Err; that belongs to the application supervisor task.
Unlocked tasks:
- TASK-016
```

```text
Task: TASK-016
Accepted: 2026-07-21
Commit: 394a7f54
Reviewer: Mark
Verification:
- gofmt -w cmd/api/main.go internal/api/server.go internal/api/server_supervisor_test.go: PASS
- go test -count=1 ./internal/api/...: PASS
- go test -count=1 -run "TestRunReturnsGRPCFatalErrorAndStopsOwnedComponents|TestRunHTTPFailureUsesUnifiedCleanup" -v ./internal/api: PASS
- go test ./internal/api/... ./internal/transport/grpcserver/...: PASS
- go test -race ./internal/api/...: FAIL; local CGO_ENABLED=0 rejects -race
- CGO_ENABLED=1 go test -race ./internal/api/...: FAIL; local runtime/cgo could not find gcc in PATH
- go test ./...: PASS
- go vet ./internal/api/... ./internal/transport/grpcserver/...: PASS
- staticcheck ./...: PASS
- git diff --check: PASS
Notes:
- Moved WorkerPool and gRPC runtime startup under APIServer.Run supervision so main only constructs components.
- Added one cleanup path for signal shutdown, HTTP/HTTPS serve failure and gRPC runtime failure.
- Added regressions proving gRPC fatal Serve errors return from Run and unexpected HTTP failures stop owned components.
- Remaining limitation: local Windows race verification requires CGO plus a C compiler.
- Remaining limitation: shutdown still uses the existing per-component timeout model; one global shutdown budget belongs to TASK-017.
Unlocked tasks:
- TASK-017
- TASK-023
- TASK-025
- TASK-076
```

```text
Task: TASK-017
Accepted: 2026-07-21
Commit: a35da624
Reviewer: Mark
Verification:
- go test -count=1 -run TestRunShutdownUsesOneGlobalBudgetAndReportsForcedOutcomes -v ./internal/api: PASS
- go test -count=1 -run TestRepairContextRespectsEarlierParentDeadline ./internal/workerpool: PASS
- go test -count=1 ./internal/api/... ./internal/workerpool/...: PASS
- go test ./internal/api/...: PASS
- go test ./internal/workerpool/...: PASS
- go test ./...: PASS
- go vet ./internal/api/... ./internal/workerpool/...: PASS
- staticcheck ./...: PASS
- git diff --check: PASS
- go test -race ./internal/api/... ./internal/workerpool/...: FAIL; local CGO_ENABLED=0 rejects -race
- CGO_ENABLED=1 go test -race ./internal/api/... ./internal/workerpool/...: FAIL; local runtime/cgo could not find gcc in PATH
Notes:
- Added one global shutdown budget based on HTTP.ShutdownTimeout and derived component shutdown contexts from it.
- Stopped independent HTTP listeners concurrently and returned typed shutdown outcomes for graceful, forced, timed_out and failed component shutdowns.
- Capped detached workerpool repair contexts by the global shutdown deadline while preserving protection from parent cancellation.
- Remaining limitation: local Windows race verification requires CGO plus a C compiler.
Unlocked tasks:
- none
```

```text
Task: TASK-018
Accepted: 2026-07-21
Commit: df86b3b8
Reviewer: Mark
Verification:
- test -f docs/adr/*grpc* || true: FAIL; PowerShell has no POSIX test/true commands
- C:\Program Files\Git\bin\bash.exe -lc 'test -f docs/adr/*grpc* || true': PASS
- Test-Path docs/adr/*grpc*: PASS
- go test ./internal/config/...: PASS
- go test ./...: PASS
- git diff --check: PASS
- git diff --check -- README.md: PASS
Notes:
- Recorded direct gRPC as a private authenticated internal interface, not a public edge API.
- Chose mTLS plus application identity propagation, RBAC/owner checks and reflection disabled by default for the deployable model.
- Updated README so current plaintext direct gRPC examples are explicitly loopback/dev-only.
- Remaining limitation: runtime enforcement is intentionally left for TASK-019, TASK-020 and TASK-021.
Unlocked tasks:
- TASK-019
```

```text
Task: TASK-019
Accepted: 2026-07-21
Commit: eb8876d7
Reviewer: Mark
Verification:
- go test -count=1 ./internal/config/...: PASS
- go test -count=1 ./internal/transport/grpcserver/... ./internal/transport/grpcclient/...: PASS
- go test -count=3 ./internal/transport/grpcserver/... ./internal/transport/grpcclient/...: PASS
- go test ./internal/transport/grpcserver/... ./internal/transport/grpcclient/...: PASS
- go test ./internal/config/...: PASS
- go test ./...: PASS
- go test -count=1 ./...: PASS
- go vet ./internal/transport/grpcserver/... ./internal/transport/grpcclient/... ./internal/config/...: PASS
- go vet ./...: PASS
- staticcheck ./...: PASS
- git diff --check: PASS
- go test -race ./internal/transport/grpcserver/... ./internal/transport/grpcclient/...: FAIL; local CGO_ENABLED=0 rejects -race
- CGO_ENABLED=1 go test -race ./internal/transport/grpcserver/... ./internal/transport/grpcclient/...: FAIL; local runtime/cgo could not find gcc in PATH
Notes:
- Added gRPC mTLS transport configuration for the server and bridge client.
- Restricted plaintext gRPC to loopback/development listeners.
- Disabled reflection by default and allowed it only through explicit loopback configuration.
- Added temporary-certificate integration tests for mTLS, plaintext rejection, server-name validation, reflection policy and startup misconfiguration.
- Remaining limitation: gRPC authentication/RBAC is still intentionally left for TASK-020; local Windows race verification requires CGO plus a C compiler.
Unlocked tasks:
- TASK-020
```

```text
Task: TASK-020
Accepted: 2026-07-21
Commit: 35cb8646
Reviewer: Mark
Verification:
- gofmt -w internal/interceptors/auth.go internal/interceptors/auth_test.go internal/transport/grpcserver/grpc_job_service.go internal/transport/grpcserver/runtime.go internal/transport/grpcserver/grpc_test.go internal/transport/grpcserver/grpc_authz_test.go internal/transport/grpcserver/transport_security_test.go internal/transport/grpcserver/runtime_test.go internal/routes/job_handler.go internal/routes/job_handler_test.go cmd/api/main.go: PASS
- go test ./internal/interceptors/... ./internal/transport/grpcserver/...: PASS
- go test ./internal/routes/... ./cmd/api: PASS
- go test ./...: PASS
- .\scripts\run-staticcheck.ps1: PASS
- go vet ./...: PASS
- git diff --check: PASS
Notes:
- Added gRPC bearer authentication through the existing security.Verifier and mapped the verified Principal into request context.
- Enforced owner/admin authorization for JobsService.GetJob and verified missing, invalid, forbidden, owner and admin calls.
- Wired the application JWT verifier into the gRPC runtime and forwarded HTTP Authorization metadata through the HTTP-to-gRPC bridge.
- Remaining limitation: TASK-021 still owns metadata append semantics, request-ID sanitization/response metadata, bridge timeout and expanded gRPC/HTTP error taxonomy.
- Remaining limitation: post-acceptance local inspection shows commit 35cb8646 does not include internal/interceptors/auth.go, internal/interceptors/auth_test.go or internal/transport/grpcserver/grpc_authz_test.go; a clean checkout of that exact commit may not compile until those files are committed.
Unlocked tasks:
- TASK-021
```

```text
Task: TASK-021
Accepted: 2026-07-21
Commit: 26ed7b00
Reviewer: Mark
Verification:
- gofmt -w internal/interceptors/interceptors.go internal/interceptors/client_test.go internal/transport/grpcclient/grpcclient.go internal/transport/grpcclient/grpcclient_test.go internal/transport/grpcserver/grpc_job_service.go internal/transport/grpcserver/grpc_test.go internal/httputils/grpc_bridge_error.go internal/httputils/grpc_bridge_error_test.go internal/httputils/errmap.go internal/routes/job_handler.go internal/routes/job_handler_test.go: PASS
- go test ./internal/interceptors/... ./internal/transport/grpcclient/... ./internal/transport/grpcserver/... ./internal/httputils/... ./internal/routes/...: PASS
- go test ./internal/interceptors/... ./internal/transport/grpcclient/... ./internal/transport/grpcserver/...: PASS
- go test ./...: PASS
- go vet ./...: PASS
- .\scripts\run-staticcheck.ps1: PASS
- git diff --check: PASS
- go test -race ./internal/interceptors/... ./internal/transport/grpcclient/... ./internal/transport/grpcserver/...: FAIL; local CGO_ENABLED=0 rejects -race
- CGO_ENABLED=1 go test -race ./internal/interceptors/... ./internal/transport/grpcclient/... ./internal/transport/grpcserver/...: FAIL; local runtime/cgo could not find gcc in PATH
Notes:
- Normalized gRPC request-id handling so inbound metadata is sanitized and stable request-id metadata is returned to clients.
- Added unary gRPC client request-id and timeout baseline, preserved existing outgoing metadata, and bounded the HTTP-to-gRPC bridge call.
- Added deterministic bridge/server mapping for canceled, deadline, unavailable, permission, not-found and authentication outcomes.
- Remaining limitations: the bridge/client call timeout is fixed rather than runtime-configurable; local Windows race verification requires CGO plus a C compiler.
- Scope note: internal/routes/job_handler.go and internal/httputils/errmap.go were touched because the actual bridge call site and HTTP status mapping live there.
Unlocked tasks:
- TASK-061
```

```text
Task: TASK-022
Accepted: 2026-07-21
Commit: b92fac3b
Reviewer: Mark
Verification:
- go test ./internal/config: PASS
- go test ./internal/middleware: PASS
- go test ./internal/config/... ./internal/middleware/...: PASS
- go test ./...: PASS
- staticcheck ./...: PASS
- git diff --check: PASS
Notes:
- Added APP_ENV security-profile guards for JWT defaults.
- Added HTTP TLS minimum-version parsing and validation.
- Made Referrer-Policy use runtime config instead of the hardcoded no-referrer value.
- Added trusted-proxy guard enforcement in config and NewProxyAPI.
- Remaining limitation: new env keys are code-backed but README docs were not updated because TASK-022 scope did not include docs.
Unlocked tasks:
- TASK-068
- TASK-069
```

```text
Task: TASK-023
Accepted: 2026-08-04
Commit: 9872ca77
Reviewer: Mark
Verification:
- go test ./internal/config/...: PASS
- go test ./internal/middleware/... ./internal/outbound/... ./internal/interceptors/...: FAIL; first run caught a missing logger argument in the updated recover test before fix
- go test ./internal/middleware/... ./internal/outbound/... ./internal/interceptors/...: PASS
- go test ./...: PASS
- git diff --check: PASS
Notes:
- Added configurable logging through LOG_LEVEL, LOG_FORMAT and LOG_ADD_SOURCE.
- Normalized logger ownership with a component-free root logger and explicit component child loggers for main, HTTP access/recover, outbound Profile and gRPC server logging.
- Standardized logged method, route, status, request_id and duration_ms fields across HTTP access, outbound Profile and gRPC interceptor logs.
- Removed duplicate component attribution from the TASK-023 logging surfaces.
- Remaining limitations/risks: none known for the accepted TASK-023 scope.
Unlocked tasks:
- TASK-024
```

```text
Task: TASK-024
Accepted: 2026-08-04
Commit: 46fae392
Reviewer: Mark
Verification:
- go test ./internal/middleware/...: PASS
- go test ./internal/outbound/...: PASS
- go test ./internal/api/...: PASS
- go test ./internal/middleware/... ./internal/outbound/... ./internal/api/...: PASS
- go test ./...: PASS
- git diff --check: PASS
- git diff --cached --check: PASS
- git diff -- docs/implementation docs/audit: PASS; no forbidden docs changed
Notes:
- Documented the logging privacy policy, allowed fields, forbidden sensitive data and propagation contract.
- Added safe AuthN/AuthZ/CORS denial audit events, grouped Profile retry logs, SSE lifecycle/write-failure logs and one shutdown summary with trigger, outcome, duration and aggregate counts.
- Removed high-risk data from the TASK-024 logging surfaces: raw tokens, request/response bodies, raw CORS origins/header values, outbound user IDs and raw outbound errors.
- Remaining limitations/risks: legacy internal/httputils.AppHandler diagnostic logging was outside TASK-024 scope and was not changed.
- Remaining limitations/risks: component-specific APIServer server-error diagnostics may still include raw server errors; the final shutdown summary does not.
Unlocked tasks:
- none
```

```text
Task: TASK-025
Accepted: 2026-08-04
Commit: 5f98c7b4
Reviewer: Mark
Verification:
- go get go.opentelemetry.io/otel@v1.43.0 go.opentelemetry.io/otel/sdk@v1.43.0 go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.43.0 go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc@v1.43.0: PASS
- go mod tidy: PASS
- gofmt -w cmd/api/main.go internal/config/config.go internal/config/config_test.go internal/telemetry/runtime.go internal/telemetry/runtime_test.go: PASS
- go test ./internal/config/...: PASS
- go test ./internal/telemetry/...: FAIL; first run exposed an invalid assumption that the current OTel SDK returns invalid endpoint errors during exporter creation
- go test ./internal/telemetry/...: PASS
- go test ./internal/telemetry/... ./internal/config/...: PASS
- go test ./...: PASS
- gofmt -l cmd/api/main.go internal/config/config.go internal/config/config_test.go internal/telemetry/runtime.go internal/telemetry/runtime_test.go: PASS
- git diff --check: PASS
- git diff --name-only -- docs/implementation docs/audit: PASS; no forbidden docs changed
Notes:
- Added telemetry config with deterministic disabled default.
- Added one composition-root-owned internal telemetry runtime with shared Resource, TracerProvider, MeterProvider and W3C TraceContext propagator.
- Wired telemetry bootstrap in cmd/api/main.go and added pinned OpenTelemetry SDK/OTLP gRPC exporter dependencies.
- Remaining limitations/risks: TASK-026 still owns fail-open policy, ForceFlush, shutdown lifecycle verification and exporter failure surfacing.
- Remaining limitations/risks: TASK-028 through TASK-031 still own log correlation and HTTP/outbound/gRPC instrumentation.
- Remaining limitations/risks: Collector/Tempo/Prometheus pipeline configuration is not implemented in TASK-025.
- Remaining limitations/risks: README/env documentation for new TELEMETRY_* keys was not updated because docs outside allowed scope were forbidden.
Unlocked tasks:
- TASK-026
- TASK-027
- TASK-028
- TASK-029
- TASK-030
- TASK-031
- TASK-032
```
