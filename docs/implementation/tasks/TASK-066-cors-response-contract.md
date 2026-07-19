# TASK-066 — Expose required browser headers and complete CORS Vary behavior

## Metadata

- Phase: `P5`
- Priority: `P1`
- Remediation group: `RG-CORS`
- Gap: `GAP-CORS-066`
- Dependencies: None
- Initial status: `BACKLOG`

## Goal

Expose required browser headers and complete CORS Vary behavior. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A12-F5: Browser clients cannot read important response headers
- A12-F6: CORS Vary coverage is incomplete on denial paths

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/middleware/cors.go`
- `internal/middleware/*cors*test.go`
- `internal/config/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Configure Access-Control-Expose-Headers for Location, ETag, X-Request-Id, Retry-After, Content-Range and reviewed headers.
2. Emit complete Vary on successful and denied simple/preflight paths.
3. Preserve deny-by-default and credentials rules.
4. Add browser-oriented contract tests.

## Acceptance criteria

- [ ] Browser JavaScript can read required response metadata.
- [ ] Cache variance is correct on denial paths.
- [ ] Wildcard+credentials remains rejected.

## Required verification

- `go test ./internal/middleware/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
