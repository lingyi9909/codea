---
name: unit-test
description: Enterprise unit test generator.
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

The prose test cases you design do not replace `task_plan`. Use only controlled Codea test-writing and test-execution tools, preserve existing production code, and never claim tests pass without fresh `run_project_test` evidence.
