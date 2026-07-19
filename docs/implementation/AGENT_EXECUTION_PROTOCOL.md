# Agent Execution Protocol

## Purpose

This protocol is the permanent instruction set for Codex or another coding agent working inside the `pet-study` repository. The agent receives this file once and exactly one task card per execution cycle.

## Source of truth

1. The active task card is the implementation source of truth.
2. `TASK_INDEX.md` controls ordering and dependencies.
3. `IMPLEMENTATION_ROADMAP.md` explains phase goals, not extra scope.
4. Audit reports are evidence only. Read a linked report when the task card is insufficient or a decision is disputed.
5. Existing project invariants remain binding unless the task explicitly changes them.

## One-task rule

- Work on exactly one `TASK-xxx` card.
- Do not start a dependent or adjacent task.
- Do not perform unrelated refactoring, renaming, formatting or dependency upgrades.
- Do not silently resolve architecture questions outside the task. Report a blocker instead.

## Required workflow

1. Read this protocol and the active task card.
2. Inspect the listed files and current repository state.
3. Restate the task goal, constraints and planned files before editing.
4. Add or update tests that prove the audited failure and desired behavior.
5. Implement the smallest coherent change satisfying the card.
6. Run every verification command that is possible in the environment.
7. Review the diff for scope creep, generated drift and security regressions.
8. Return the completion report below.

## Safety and project invariants

- Preserve `http.ServeMux.ServeHTTP`, `r.Pattern` and `PathValue` behavior.
- Preserve request-ID trust rules unless the active task changes them.
- Never log tokens, secrets, full request bodies or unreviewed PII.
- Do not put request, trace, user or job identifiers into metric labels.
- Do not put `context.Context` into durable or queued payloads.
- Do not close producer-facing queue channels as a shortcut.
- Do not weaken auth, RBAC, CORS, TLS or readiness to make tests pass.
- Do not replace typed errors with string matching.
- Do not mark a task complete when required verification was skipped without an explicit reason.

## Generated files and dependencies

- Use only repository-pinned generators and commands.
- Never use `@latest` in implementation instructions or scripts.
- Do not edit generated files manually unless the task explicitly says so.
- Explain and justify every new dependency.
- Run generation and verify the working tree contains only expected changes.

## Required completion report

```text
Task: TASK-xxx
Status: DONE | PARTIAL | BLOCKED

Summary:
- ...

Files changed:
- path: reason

Tests/commands:
- command: PASS | FAIL | NOT RUN — reason

Acceptance criteria:
- criterion: PASS | FAIL

Design decisions:
- ...

Remaining risks/blockers:
- ...

Unexpected scope changes:
- none | explanation
```

## Review rule

A task becomes `DONE` only after its diff and acceptance criteria are reviewed. Agent self-report alone is not acceptance.
