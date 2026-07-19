# TASK-002 — Pin protobuf and quality-tool versions

## Metadata

- Phase: `P0`
- Priority: `P0`
- Remediation group: `RG-REPRO`
- Gap: `GAP-REPRO-002`
- Dependencies: TASK-001
- Initial status: `BACKLOG`

## Goal

Pin protobuf and quality-tool versions. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A01-R1: Clean checkout is not reproducible
- A13-Q2: Generated protobuf code is required but untracked

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `tools/**`
- `scripts/**`
- `Makefile`
- `Taskfile.yml`
- `go.mod`
- `README.md`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Pin protoc plugin and quality tool versions; do not use `@latest`.
2. Provide one reproducible generation command for protobuf artifacts.
3. Record required external tool versions such as `protoc`.
4. Make generation deterministic across supported developer environments.

## Acceptance criteria

- [ ] Two consecutive generation runs produce no diff.
- [ ] Tool versions are reviewable in the repository.
- [ ] README commands match the actual entrypoint.

## Required verification

- `go generate ./...`
- `git diff --exit-code`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
