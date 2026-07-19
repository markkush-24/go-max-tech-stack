# TASK-009 — Define an explicit immutable-terminal Job state machine

## Metadata

- Phase: `P1`
- Priority: `P0`
- Remediation group: `RG-JOB-CORRECTNESS`
- Gap: `GAP-JOB-CORRECTNESS-009`
- Dependencies: TASK-007
- Initial status: `BACKLOG`

## Goal

Define an explicit immutable-terminal Job state machine. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A10-F10: Job transitions consume work without guaranteed terminality
- A10-F2: Active worker can overwrite shutdown terminal state

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/entity/job.go`
- `internal/service/jobService.go`
- `internal/service/jobRepository.go`
- `internal/apperr/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Define allowed transitions and immutable terminal states.
2. Introduce a typed transition-conflict error.
3. Separate transition intent from arbitrary status assignment.
4. Document concurrency semantics expected from repositories.

## Acceptance criteria

- [ ] Allowed transitions are unambiguous.
- [ ] `succeeded` and `failed` cannot transition further.
- [ ] Service interfaces expose expected-source-state semantics.

## Required verification

- `go test ./internal/entity/... ./internal/service/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
