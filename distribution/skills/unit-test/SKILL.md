---
name: unit-test
description: Enterprise unit test generator with protected writes and machine verification.
---

# Unit Test Generator

Generate focused unit tests for the target module. Follow the project's existing test conventions and keep assertions tight and deterministic.

## Persistent planning protocol

- Inspect evidence first with `analyze_test_project`, `read`, `grep`, and `glob`; read-only analysis may happen before a plan.
- Before the first mutation or command execution, call `task_plan` with a bounded **3–7** step plan. `write_test_file` and `run_project_test` require that persisted plan.
- Before working a planned step, call `task_step` with status `in_progress`; keep only one active step.
- Call `task_step` with status `completed` only with concrete evidence from the generated file or fresh structured test result.
- If a step cannot proceed, mark it `blocked` with concise evidence and use `task_status` to reread persisted state when needed.
- Never fabricate tool output or plan evidence.

## Required completion flow

1. **Analyze target and risk** using `analyze_test_project` and a small representative set of nearby tests. Do not guess framework/dependency choices.
2. **Plan or refresh** the bounded persisted `task_plan` before any mutation or command execution.
3. **Protected test write** through `write_test_file` only. Preserve production code and arbitrary existing tests.
4. **Machine verification** after the latest mutation: run `verify_project`. Completion requires machine-observable verification evidence from the fresh call. Focused `run_project_test` evidence is useful but does not replace this root-task completion gate.
5. **Report from machine evidence** using the latest structured results. Never claim the mutating task passed/completed from memory, prose, or a verification that predates the latest mutation.

If fresh `verify_project` fails and the verification repair budget remains, perform one bounded repair from machine evidence and run `verify_project` again. `NOT_CONFIGURED` and `TIMEOUT` are unverified outcomes, never PASS; if fresh PASS is still unavailable, stop and report the unverified state.

When Codea sends an internal `verification-control` continuation, treat it as the same root task, not a new user task; it must not reset the plan epoch. Continue the existing persisted plan instead of starting a new ordinary task.

The prose test cases you design do not replace `task_plan`. Use only controlled Codea test-writing and test-execution tools, preserve existing production code, and never claim focused tests pass without fresh `run_project_test` evidence. Root-task completion after mutation is determined only by fresh `verify_project` machine-observable verification evidence.
