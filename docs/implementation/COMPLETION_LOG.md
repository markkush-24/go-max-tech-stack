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
