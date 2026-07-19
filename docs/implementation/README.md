# Implementation Package

This directory converts the completed 14-pass audit into **98** bounded Codex tasks.

## What to give Codex

Codex already has the repository open in IntelliJ IDEA. For each execution cycle, send:

1. `docs/implementation/AGENT_EXECUTION_PROTOCOL.md` — once per Codex chat/session.
2. Exactly one task card from `docs/implementation/tasks/`.
3. No full audit bundle unless the task card or a disputed decision requires it.

Suggested prompt:

```text
Read docs/implementation/AGENT_EXECUTION_PROTOCOL.md.
Execute only docs/implementation/tasks/TASK-xxx-....md.
Do not start another task. Return the required completion report.
```

## How to select work

- Open `TASK_INDEX.md`.
- Select a task marked `READY` whose dependencies are `DONE`.
- Use P0 before P1, and earlier phases before later phases unless the dependency graph explicitly permits parallel work.
- After Codex reports completion, review the diff and acceptance criteria before marking `DONE`.

## What not to do

- Do not paste all audit reports into every Codex request.
- Do not ask Codex to choose the next task or redesign the roadmap.
- Do not batch unrelated task cards in one request.
- Do not mark a task done solely because compilation succeeds.

## Validation checkpoints

A fresh independent review should be performed after:

- P1: critical correctness/security checkpoint;
- P2/P3: observability end-to-end checkpoint;
- P4/P5: resilience and API-contract checkpoint;
- P6: quality-system checkpoint;
- P7: final project re-audit.

The final re-audit should use a new project archive or repository snapshot plus the updated `TASK_INDEX.md` statuses and agent completion reports.
