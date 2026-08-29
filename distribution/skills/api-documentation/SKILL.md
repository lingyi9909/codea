---
name: api-documentation
description: Enterprise API documentation generation from deterministic code evidence.
---

# API Documentation

Generate API documentation from deterministic extraction first. Treat code and Custom Tool output as authoritative evidence, keep business enrichment optional, and enforce no fabrication: unresolved fields must be marked `Not determined from code`.

## Persistent planning protocol

- Inspect evidence first with `extract_api_spec`, `validate_api_example`, `read`, `grep`, and `glob`; read-only analysis may happen before a plan.
- Before the first mutation or command execution, call `task_plan` with a bounded **3–7** step plan. For this skill the mutation is normally `write_document`.
- Before working a planned step, call `task_step` with status `in_progress`; keep only one active step.
- Call `task_step` with status `completed` only with concrete evidence from deterministic extraction, validation, or the successful document write.
- If a step cannot proceed, mark it `blocked` with concise evidence and use `task_status` to reread persisted state when needed.
- Never fabricate tool output or plan evidence.

Do not create a plan for purely read-only/explanatory API analysis. When a document write is required, the persisted machine plan is mandatory and does not override code evidence, path/DLP policy, approval, or the no-fabrication rule.
