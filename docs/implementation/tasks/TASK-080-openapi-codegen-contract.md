# TASK-080 — Add pinned OpenAPI validation, codegen and runtime conformance checks

## Metadata

- Phase: `P6`
- Priority: `P1`
- Remediation group: `RG-OPENAPI`
- Gap: `GAP-OPENAPI-080`
- Dependencies: TASK-079, TASK-002
- Initial status: `BACKLOG`

## Goal

Add pinned OpenAPI validation, codegen and runtime conformance checks. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A12-F24: No machine-readable HTTP contract exists
- A13-Q2: Generated protobuf code is required but untracked
- A13-Q20: Previously discovered defects lack regression tests
- A13-Q7: No OpenAPI or runtime contract test exists

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `api/openapi/**`
- `internal/transport/httpgenerated/**`
- `tools/**`
- `scripts/**`
- `.github/workflows/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Pin validator/codegen versions.
2. Choose a bounded generated artifact such as typed client/models without replacing the architecture wholesale.
3. Validate actual responses against the spec in integration tests.
4. Fail CI on generated/spec drift and breaking changes.

## Acceptance criteria

- [ ] Generation is deterministic.
- [ ] At least core success and Problem responses pass runtime validation.
- [ ] A deliberate schema/status mismatch fails CI.

## Required verification

- `go generate ./...`
- `git diff --exit-code`
- `go test ./...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
