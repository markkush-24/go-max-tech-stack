# P0-CORR-001 — Make codegen, clean-checkout and toolchain checks exact

## Goal

Close the remaining correctness gaps in the P0 quality system so local and CI verification use the same toolchain and detect version/module drift exactly.

## Review findings

- P0-02-F1 — substring tool-version comparison accepts wrong versions.
- P0-02-F2 — clean-checkout does not tidy the nested `tools` module.
- P0-05-F1 — local vulnerability scan forces Go 1.25.8 instead of the Go 1.25.12 baseline.

## Allowed scope

- `scripts/**`
- `.github/workflows/**`
- `tools/**`
- `docs/implementation/TOOLCHAIN_BASELINE.md`
- `README.md`
- focused tests for scripts/generator behavior

## Forbidden scope

- no application behavior changes;
- no dependency upgrade unless required by a newly reproduced vulnerability and explicitly reported;
- no unrelated CI redesign;
- no P1 task work.

## Requirements

1. Replace substring version checks with exact normalized token comparison.
2. Reject near-match versions such as:
   - required `v1.36.11`, actual `v1.36.110`;
   - required `1.6.1`, actual `1.6.10`.
3. Re-check the installed version after `go install`; fail if it still does not match.
4. Add `go -C tools mod tidy` to the clean-checkout gate before the clean-tree assertion.
5. Add a regression scenario proving drift in `tools/go.mod` or `tools/go.sum` fails the gate.
6. Remove the `go1.25.8+auto` override from `run-govulncheck.ps1`.
7. Ensure local vulnerability scans build with the toolchain selected by the repository baseline.
8. Keep CI, README and `TOOLCHAIN_BASELINE.md` consistent.

## Acceptance criteria

- exact expected tool versions pass;
- suffix/prefix near-matches fail;
- stale `tools/go.mod` fails clean-checkout;
- `run-govulncheck.ps1` does not contain `go1.25.8`;
- local and CI binary scans use Go 1.25.12 baseline semantics;
- two consecutive generation runs produce no diff;
- clean-checkout ends with a clean tree.

## Required verification

```text
go test ./scripts/...
go -C tools mod tidy
go generate ./...
go generate ./...
git diff --exit-code
pwsh -File ./scripts/check-clean-checkout.ps1
pwsh -File ./scripts/run-govulncheck.ps1
git grep -n "go1.25.8" -- scripts README.md docs/implementation .github || true
go test ./...
go vet ./...
```

Run the clean-checkout mutation tests in a disposable clone.

## Required completion report

Report:

- exact version parser behavior;
- nested-module drift test result;
- toolchain used to build the scanned binary;
- all changed files;
- all commands and results;
- remaining limitations.
