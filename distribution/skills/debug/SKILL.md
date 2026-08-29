---
name: debug
description: Evidence-driven enterprise debugging with controlled fixes and fresh verification.
---

# Debug

Use this skill for application failures, test failures, build failures, regressions, and unexpected runtime behavior.

## Persistent planning protocol

- Inspect evidence first; read-only investigation may happen before a plan.
- Before the first mutation or command execution, call `task_plan` with a bounded **3–7** step plan.
- Before working a planned step, call `task_step` with status `in_progress`; keep only one active step.
- Call `task_step` with status `completed` only with concrete fresh evidence.
- If progress cannot continue, mark the step `blocked` with concise evidence and use `task_status` to reread persisted state when needed.
- Never fabricate tool output or plan evidence.

## Required loop

1. **Collect evidence** from the reported failure, logs, stack traces, tests, code, and workspace state. Separate facts from assumptions.
2. **Reproduce** the problem with the smallest relevant check when feasible. If command execution is required, establish the machine plan first. If it cannot be reproduced, record why and what evidence is still available.
3. **Root cause** the failure by tracing backward from the symptom to the earliest supported cause. Prefer fixing the cause over masking the symptom.
4. **Controlled fix** with the smallest scoped change. Mutating files or executing commands remains subject to the Task 29 plan gate, Codea approval, project path policy, command policy, DLP, audit, and offline constraints.
5. **Fresh verification** after the change: rerun the focused failing check, then the relevant regression checks.
6. Summarize evidence, root cause, files changed, verification performed, and remaining risks.

Do not claim success merely because code was edited. A fix is only complete when fresh verification supports it. If the plan gate, approval, or security policy blocks a required action, report the block and preserve the controls.
