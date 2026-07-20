# Toolchain Baseline

Last verified: 2026-07-20

## Target

- Go toolchain: `go1.25.12`
- Module directive: `go 1.25.0`
- CI setup-go version: `1.25.12`

The baseline is `go1.25.12` because `govulncheck -mode binary` on binaries built
with `go1.25.8` reports Go standard-library vulnerabilities. The newest fixed
version required by the reported findings is `go1.25.12`.

## Commands

Run from the repository root:

```powershell
go version
go test ./...
go vet ./...
go test -race ./...
go -C tools mod tidy
go generate ./...
go generate ./...
gofmt -l cmd internal scripts
staticcheck ./...
New-Item -ItemType Directory -Force -Path .artifacts | Out-Null
go build -o .artifacts/pet-study.exe ./cmd/api
govulncheck -mode binary .artifacts/pet-study.exe
pwsh -File ./scripts/check-clean-checkout.ps1
pwsh -File ./scripts/run-govulncheck.ps1
```

On Linux CI the binary path is `.artifacts/pet-study`.

## Recorded Results

- `go version`: PASS; `go version go1.25.12 windows/amd64`
- `.\scripts\install-tools.ps1`: PASS; exact version tokens matched `govulncheck@v1.1.4`, `v0.7.0`, `v1.36.11` and `1.6.1`
- `go test ./scripts/...`: PASS
- `go -C tools mod tidy`: PASS; no `tools/go.mod` or `tools/go.sum` drift
- `go generate ./...`: PASS
- `go generate ./...`: PASS; second consecutive run produced no generated drift
- `go test ./...`: PASS
- `go vet ./...`: PASS
- `go test -race ./...`: BLOCKED locally; default `CGO_ENABLED=0` rejects `-race`
- `CGO_ENABLED=1 go test -race ./...`: BLOCKED locally; `runtime/cgo` cannot find `gcc` in `PATH`
- `gofmt -l cmd internal scripts`: PASS; no output
- `staticcheck ./...`: PASS
- `pwsh -File ./scripts/check-clean-checkout.ps1`: PASS in a disposable clean clone with the correction patch applied
- `go build -o .artifacts/pet-study.exe ./cmd/api`: PASS
- `go build -o .artifacts/pet-study ./cmd/api`: PASS
- `govulncheck -version`: PASS; `govulncheck@v1.1.4`, DB updated `2026-07-08`
- `govulncheck -mode binary .artifacts/pet-study.exe`: PASS; affected vulnerabilities: `0`
- `govulncheck -mode binary .artifacts/pet-study`: PASS; affected vulnerabilities: `0`
- `.\scripts\run-govulncheck.ps1`: PASS; built the scanned binary with `go version go1.25.12 windows/amd64`

## Notes

- Race testing remains part of CI on Ubuntu with the pinned Go toolchain.
- The local Windows environment needs a C compiler before `go test -race ./...`
  can run.
- `scripts/run-govulncheck.ps1` does not set `GOTOOLCHAIN`; it uses the
  repository-selected Go toolchain, matching the CI baseline semantics.
- Codegen and pinned-tool checks compare normalized version tokens exactly, so
  prefix/suffix near-matches do not satisfy the baseline.
