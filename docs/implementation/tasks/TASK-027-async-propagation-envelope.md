# TASK-027 — Add broker-compatible async propagation envelope and per-job context

## Metadata

- Phase: `P2`
- Priority: `P1`
- Remediation group: `RG-OTEL`
- Gap: `GAP-OTEL-027`
- Dependencies: TASK-013, TASK-011, TASK-025
- Initial status: `BACKLOG`

## Goal

Add broker-compatible async propagation envelope and per-job context. Complete this outcome without implementing adjacent roadmap tasks.

## Audit evidence

- A02-E3: Async work loses request/trace correlation at enqueue
- A03-L1: Async worker logs cannot correlate to the initiating request
- A05-T14: Security identity remains request-local and stops at integration boundaries
- A05-T15: SSE connection cancellation is correct, but event causality is absent
- A05-T2: Async enqueue is a hard propagation break
- A05-T3: Workers use one shared lifecycle context rather than a per-job operation context
- A05-T4: `markJobFailed(context.Background())` severs all operation context
- A05-T5: Job persistence has no correlation or propagation metadata

## Context

The audit found the behavior above in the uploaded working tree. This card converts those findings into one bounded engineering outcome. Inspect the current implementation before changing it; filenames and exact APIs may have evolved since the audit.

## Allowed scope

- `internal/queue/**`
- `internal/entity/job.go`
- `internal/routes/users_handler*.go`
- `internal/workerpool/**`
- `internal/requestid/**`

## Forbidden scope

- Do not start another task card.
- Do not perform unrelated architectural rewrites or formatting sweeps.
- Do not change public API/security behavior unless this card explicitly requires it.
- Do not suppress an error, disable a test, weaken a limit or broaden trust merely to satisfy verification.
- Do not introduce unpinned tools or generators.

## Implementation requirements

1. Do not store `context.Context` in queued work.
2. Carry request ID, trace carrier, enqueue time and bounded actor metadata explicitly.
3. Create a per-job context derived from worker lifecycle context.
4. Preserve accepted work after the producer request is canceled.

## Acceptance criteria

- [ ] Worker logs/traces can link to the producer request.
- [ ] The envelope is serializable and broker-ready.
- [ ] HTTP cancellation after 202 does not cancel accepted work.

## Required verification

- `go test -race ./internal/queue/... ./internal/workerpool/... ./internal/routes/...`

Also run the narrowest relevant `go test` command after every meaningful change. Run `go test ./...` before completion when the environment permits it.

## Required agent response

Use the completion-report format from `docs/implementation/AGENT_EXECUTION_PROTOCOL.md`. Report any criterion that could not be verified as `FAIL` or `NOT RUN`; do not silently assume success.
