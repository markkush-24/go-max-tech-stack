# TASK-075 — Replace O(n) in-memory email lookup with an indexed structure

## Metadata

- Phase: `P5`
- Priority: `P2`
- Remediation group: `RG-PERFORMANCE`
- Gap: `GAP-PERFORMANCE-075`
- Dependencies: None
- Initial status: `BACKLOG`

## Goal

Replace O(n) in-memory email lookup with an indexed structure. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A09-F7: Memory repository email uniqueness is O(n)
- A10-F18: Memory repositories have good locking and copy discipline

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/store/userrepo/memory_user_repo.go`
- `internal/store/userrepo/memory_user_repo_test.go`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Maintain an email-to-ID index under the same lock as the user map.
2. Preserve uniqueness and copy discipline.
3. Update index on all supported mutations.
4. Add scale-oriented correctness benchmarks/tests.

## Acceptance criteria

- [ ] ExistsByEmail and duplicate detection are O(1) average.
- [ ] Index cannot drift from the user map.
- [ ] Race tests pass.

## Required verification

- `go test -race ./internal/store/userrepo/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
