# Task Review Checklist

Use this after Codex returns a completion report.

## Scope

- [ ] Only the active task was implemented.
- [ ] No unrelated refactor, dependency upgrade or contract change was added.
- [ ] Generated files were produced through pinned commands.

## Correctness

- [ ] Every acceptance criterion is backed by code or a test.
- [ ] The original audited failure has a regression test.
- [ ] Error, cancellation and concurrency behavior was reviewed.
- [ ] Public security and API invariants were preserved.

## Verification

- [ ] Required narrow tests passed.
- [ ] `go test ./...` passed, or the blocking reason is recorded.
- [ ] Race/integration/generation commands required by the card were run.
- [ ] The working tree contains only expected changes.

## Decision

- [ ] `DONE` — accepted and dependencies may be unlocked.
- [ ] `REVIEW` — changes exist but evidence is incomplete.
- [ ] `BLOCKED` — a prerequisite or architecture decision is missing.
- [ ] `PARTIAL` — split/rework is required before continuing.
