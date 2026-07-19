# TASK-004 — Define repository and archive hygiene rules

## Metadata

- Phase: `P0`
- Priority: `P2`
- Remediation group: `RG-REPRO`
- Gap: `GAP-REPRO-004`
- Dependencies: None
- Initial status: `BACKLOG`

## Goal

Define repository and archive hygiene rules. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A01-R4: Archive contains sensitive/local artifacts

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `.gitignore`
- `scripts/**`
- `docs/**`
- `README.md`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Exclude IDE state, logs, request scratch files, private keys and accidental files from shareable archives.
2. Provide a safe archive/export command.
3. Document that development certificates and keys must be generated locally.
4. Add a lightweight secret/artifact check to the export path.

## Acceptance criteria

- [ ] The project export contains no private key or runtime log.
- [ ] The export process is documented and repeatable.
- [ ] Local development remains possible after regenerating ignored artifacts.

## Required verification

- `git check-ignore certs/localhost-key.pem server.log .idea || true`
- `find . -maxdepth 2 -type f | sort`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
