# Codex Prompt Template

Use this prompt for every task execution:

```text
Read and follow docs/implementation/AGENT_EXECUTION_PROTOCOL.md.

Execute exactly one task:
docs/implementation/tasks/TASK-XXX-<slug>.md

Before editing:
1. Inspect the current repository state and the files listed in the task.
2. Restate the goal, constraints, dependencies and planned files.
3. Report BLOCKED instead of inventing a missing architecture decision.

During implementation:
- Stay inside the task scope.
- Add tests that prove the audited failure and the desired behavior.
- Do not begin another task.

At the end, return the required completion report from AGENT_EXECUTION_PROTOCOL.md.
```

For a continuing Codex chat, the first sentence may be shortened to: `Continue following the existing Agent Execution Protocol.`
