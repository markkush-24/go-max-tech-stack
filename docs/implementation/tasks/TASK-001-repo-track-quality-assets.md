# TASK-001 — Track CI, scripts, tools and generated-artifact policy

## Metadata

- Phase: `P0`
- Priority: `P0`
- Remediation group: `RG-REPRO`
- Gap: `GAP-REPRO-001`
- Dependencies: None
- Initial status: `READY`

## Goal

Track CI, scripts, tools and generated-artifact policy. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A01-R1: Clean checkout is not reproducible
- A01-R2: CI and local quality tooling are not committed
- A13-Q1: The quality system is not part of the tracked repository
- A13-Q2: Generated protobuf code is required but untracked

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `.github/**`
- `scripts/**`
- `tools/**`
- `internal/transport/pb/**`
- `.gitignore`
- `README.md`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Choose and document whether generated `*.pb.go` files are committed or generated before every build.
2. Add the currently local CI/scripts/tools assets to the tracked repository after reviewing them for local-only content.
3. Remove ambiguity between the working tree and a clean checkout.
4. Document the canonical source files for generated artifacts.

## Acceptance criteria

- [ ] A fresh clone contains the declared quality system.
- [ ] The generated-artifact policy is explicit and internally consistent.
- [ ] No required build input exists only as an untracked local file.

## Required verification

- `git ls-files .github scripts tools internal/transport/pb`
- `git status --short`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
