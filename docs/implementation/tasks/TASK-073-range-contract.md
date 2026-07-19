# TASK-073 — Complete Range validators, conditional requests and HEAD behavior

## Metadata

- Phase: `P5`
- Priority: `P2`
- Remediation group: `RG-HTTP-CONTRACT`
- Gap: `GAP-HTTP-CONTRACT-073`
- Dependencies: TASK-064
- Initial status: `BACKLOG`

## Goal

Complete Range validators, conditional requests and HEAD behavior. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A09-F19: The Range endpoint is not a meaningful large-object streaming workload
- A12-F20: Range works, but the export representation lacks validators and consistent HEAD

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/routes/**`
- `internal/httpapi/users.go`
- `internal/routes/streaming_range_test.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Add stable ETag and/or Last-Modified for the export representation.
2. Support consistent HEAD metadata.
3. Test If-Range and unsatisfiable ranges through `ServeContent`.
4. Keep the endpoint scoped as a protocol demo unless large-object storage is added.

## Acceptance criteria

- [ ] HEAD and GET expose consistent validators/length metadata.
- [ ] Conditional Range behavior is deterministic.
- [ ] No custom unsafe Range parser is introduced.

## Required verification

- `go test ./internal/routes/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
