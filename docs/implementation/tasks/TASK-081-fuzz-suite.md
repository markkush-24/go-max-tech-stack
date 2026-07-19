# TASK-081 — Add fuzz/property tests for protocol parsers and state transitions

## Metadata

- Phase: `P6`
- Priority: `P1`
- Remediation group: `RG-QA`
- Gap: `GAP-QA-081`
- Dependencies: None
- Initial status: `BACKLOG`

## Goal

Add fuzz/property tests for protocol parsers and state transitions. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A13-Q8: There are no fuzz targets

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/httputils/**/*_test.go`
- `internal/requestid/**/*_test.go`
- `internal/middleware/**/*_test.go`
- `internal/store/jobrepo/**/*_test.go`
- `internal/outbound/**/*_test.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Fuzz JSON, Accept, Content-Type, Authorization, request ID, ETag, XFF and query parsing.
2. Fuzz backoff arithmetic and Job transition sequences.
3. Assert no panic, bounded work and stable error classification.
4. Seed with known audit regressions.

## Acceptance criteria

- [ ] At least one fuzz target exists for each high-risk parser family.
- [ ] Known malformed cases remain in seed corpus.
- [ ] Short CI fuzz smoke runs are reproducible.

## Required verification

- `go test ./... -run=^$ -fuzz=Fuzz -fuzztime=10s`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
