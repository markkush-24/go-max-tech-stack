# P0-CORR-004 — Add final regression guards and close P0 bookkeeping

## Goal

Close the final review gaps so P0 is protected by real automated tests rather than successful commands that execute zero relevant tests.

## Findings

- CLOSURE-F1 — `go test -run ExportArchive -v ./scripts` reports `[no test files]`; archive hardening has no committed regression suite.
- CLOSURE-F2 — README currently lists all 58 ENV variables, but no automated test protects README/config consistency.
- CLOSURE-F3 — PowerShell code blocks for SSE/gRPC/Range still use CMD `^` continuation characters.
- CLOSURE-F4 — P0-CORR-003 remains `READY` and has no accepted completion entry.

## Allowed scope

- `scripts/*_test.go`
- focused test helpers/fixtures under `scripts/testdata/**`
- `internal/config/*_test.go` or a small shared config-manifest helper
- `README.md`
- after human acceptance only:
  - `docs/implementation/TASK_INDEX.md`
  - `docs/implementation/COMPLETION_LOG.md`

## Forbidden scope

- no application runtime behavior changes;
- no new secret-management platform;
- no P1 implementation work;
- no unrelated documentation rewrite.

## Requirements

1. Add committed automated tests that invoke `scripts/export-archive.ps1` in disposable Git repositories.
2. Tests must prove:
   - valid export succeeds;
   - two exports of one treeish are byte-identical;
   - every blocked extension/path from P0-CORR-002 is rejected;
   - small and >1 MiB private-key marker files are rejected;
   - failed validation leaves no new final archive;
   - `-Force` preserves an existing valid archive when the replacement candidate is invalid.
3. Tests must fail clearly or skip with an explicit reason when `pwsh` is unavailable. CI must run them on a runner with `pwsh`.
4. Add an automated exact-set test between the README ENV table and the 58-item config test manifest or a production-owned shared manifest.
5. The ENV test must fail for missing, extra, and duplicate README rows.
6. Replace CMD `^` line continuations in fenced PowerShell examples with PowerShell backticks or single-line commands.
7. Do not mark the correction accepted yourself. Return a completion report for human review.

## Acceptance criteria

- `go test -run ExportArchive -v ./scripts` executes named tests, not `[no test files]`;
- all archive negative/atomicity scenarios pass;
- README/config exact-set test passes;
- a deliberate README ENV mutation makes the test fail;
- no `powershell` example uses `^` as a line continuation;
- full Go 1.25.12 CI is green;
- after human acceptance, P0-CORR-003 and P0-CORR-004 are recorded as DONE.

## Verification

```text
go test -count=1 -run ExportArchive -v ./scripts
go test -count=1 ./internal/config/...
go test ./scripts/... ./internal/config/...
go test ./...
go vet ./...
go test -race ./...
pwsh -File ./scripts/check-clean-checkout.ps1
git diff --check
git status --short
```

Run mutation scenarios in disposable clones and report their expected failures.

## Completion report

Report:

- test names actually executed;
- all archive scenarios and results;
- README/config mutation-test result;
- corrected README blocks;
- changed files;
- Go/pwsh versions;
- verification commands and results;
- remaining limitations.
