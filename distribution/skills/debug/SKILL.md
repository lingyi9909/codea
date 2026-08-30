---
name: debug
description: Evidence-driven enterprise debugging with controlled fixes and machine verification.
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
2. **Reproduce** the problem from the smallest defensible evidence. If actual command execution is required, obey the persisted planning gate before executing it.
3. **Plan or refresh** the bounded machine `task_plan` before mutation or command execution, incorporating the reproduction evidence.
4. **Root cause** the failure by tracing backward from the symptom to the earliest supported cause. Prefer fixing the cause over masking the symptom.
5. **Controlled fix** with the smallest scoped change. Mutating files or executing commands remains subject to the Task 29 plan gate, Codea approval, project path policy, command policy, DLP, audit, and offline constraints.
6. **Machine verification** after the latest mutation: run `verify_project`. Completion requires machine-observable verification evidence from this fresh call; an edit, memory, assistant prose, or an older check cannot establish success.
7. **Bounded repair** only when fresh verification fails: make one bounded repair from the machine evidence, then run `verify_project` again. `NOT_CONFIGURED` and `TIMEOUT` are unverified outcomes, never PASS. Stop automatic repair if fresh PASS is still unavailable.
8. **Report from machine evidence** with the root cause, changed files, latest verification result, and remaining risks. Do not label a mutating task verified/completed without fresh PASS after the latest mutation.

When Codea sends an internal `verification-control` continuation, treat it as the same root task, not a new user task; it must not reset the plan epoch. Continue the existing persisted plan and use the continuation only for machine verification or its bounded repair.

Do not claim success merely because code was edited. A mutating fix is only complete when fresh `verify_project` machine-observable verification evidence supports it. If the plan gate, approval, or security policy blocks a required action, report the block and preserve the controls.
