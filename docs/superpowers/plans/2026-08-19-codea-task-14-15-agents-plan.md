# Codea V1 Task 14-15 Enterprise Agents Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans and superpowers:test-driven-development to implement this plan task-by-task.

**Goal:** Deliver the Code Reviewer and Unit Test Generator enterprise agents on top of Task 12 security controls and Task 13 deterministic custom tools, without bypassing those controls or expanding into Task 16.

**Architecture:** Enterprise-controlled Agent manifests and prompts orchestrate existing Task 13 tools. Reviewer is read-only and grounds findings in `collect_review_context` plus bounded repository reads. Unit Test Generator performs analyze → plan → generate → controlled write → controlled run → classify → bounded repair. Native `write/edit/bash` remain denied.

**Tech Stack:** OpenCode enterprise Agent resources, YAML manifests, JSON Schema, Go 1.26.5 E2E contract tests, existing TypeScript/Bun custom tools, Java/Spring/Maven fixture.

**Specification:** User-approved Batch 14-15 instructions from 2026-08-19 plus the authoritative `2026-07-30-codea-v1-plan.md` Task 14/15 definitions.

## Global Constraints

- Task 13 must be `completed` with `humanAccepted=true` before Task 14 starts.
- Do not modify `scripts/check-execution-state.sh` or its semantics.
- Keep exactly one formal active Task; Task 15 remains `pending` while pre-developed in the approved Batch exception.
- Do not modify OpenCode Core, AgentRuntime, Skill Manager, or Task 13 tools unless a verified blocker requires it.
- Enterprise agents may not bypass Task 12/13 with native `write`, `edit`, or `bash`.
- Use TDD: add E2E contract tests first, observe RED, then add the minimum agent artifacts needed for GREEN.

---

### Task 1: Close Task 12/13 State and Start Task 14

**Files:**
- Modify: `docs/execution-state.yaml`

- [x] Validate baseline Task 12 `awaiting_acceptance`, Task 13 `pending`.
- [x] Transition Task 12 to `completed` + `humanAccepted=true`; run validator.
- [x] Transition Task 13 to `awaiting_acceptance`; run validator.
- [x] Transition Task 13 to `completed` + `humanAccepted=true`; run validator.
- [x] Transition Task 14 to `in_progress`; keep Task 15 `pending`; run validator.

### Task 2: Code Reviewer Contract Tests (RED)

**Files:**
- Create: `tui/tests/e2e/code-review/review_test.go`

- [ ] Test manifest identity, required/optional skills, allowed tools, and native write/edit/bash deny.
- [ ] Test prompt requires scope-first `collect_review_context` flow and all six review scopes.
- [ ] Test bounded Java call-chain expansion with default max depth 3 and downstream-read-before-finding rule.
- [ ] Test finding contract: evidence, introducedByChange, severity taxonomy, confidence threshold >= 0.80, clean-diff behavior.
- [ ] Test Dify is optional knowledge only and cannot be the code-evidence source.
- [ ] Test JSON Schema validates required top-level sections and finding/reviewStats fields.
- [ ] Run targeted Go test and confirm RED because Task 14 artifacts do not exist.

### Task 3: Code Reviewer Agent (GREEN)

**Files:**
- Create: `distribution/agents/code-reviewer/agent.md`
- Create: `distribution/agents/code-reviewer/manifest.yaml`
- Create: `distribution/agents/code-reviewer/output-schema.json`

- [ ] Implement scope-first workflow around `collect_review_context`.
- [ ] Implement changed-behavior-only context expansion and Java/Spring call-chain depth 3.
- [ ] Implement introduced-by-change filtering, severity definitions, confidence gate, evidence requirements, and self-check.
- [ ] Implement clean-diff PASS behavior without fabricated findings.
- [ ] Implement Dify degradation and `businessKnowledgeUnavailable` reporting.
- [ ] Run targeted Go test to GREEN.

### Task 4: Unit Test Generator Contract Tests (RED)

**Files:**
- Create: `tui/tests/e2e/unit-test/ut_gen_test.go`

- [ ] Test manifest permissions, required skill, max repair attempts 3, and no native write/edit/bash.
- [ ] Test analyze-first workflow and unknown-framework stop behavior.
- [ ] Test Test Plan contract and representative existing-test style discovery.
- [ ] Test write-only-via-`write_test_file`, run-only-via-`run_project_test`, no `extraArgs`.
- [ ] Test smallest-scope run ordering: method → class → module.
- [ ] Test eight semantic error categories and repairable/non-repairable boundaries.
- [ ] Test current-run file ownership, no production/existing-test edits, no weakened assertions, PRODUCT_CODE_DEFECT_SUSPECTED stop.
- [ ] Run targeted Go test and confirm RED because Task 15 artifacts do not exist.

### Task 5: Unit Test Generator Agent (GREEN)

**Files:**
- Create: `distribution/agents/unit-test-generator/agent.md`
- Create: `distribution/agents/unit-test-generator/manifest.yaml`
- Create: `distribution/agents/unit-test-generator/error-categories.yaml`

- [ ] Implement analyze → plan → generate → controlled write → controlled run → classify → repair workflow.
- [ ] Default `overwrite=false`; permit repair overwrite only for files created by the current run.
- [ ] Bound repair attempts to 3 and stop on dependency/infrastructure/security failures.
- [ ] Prohibit production-code changes and test weakening.
- [ ] Base final report only on structured `run_project_test` output.
- [ ] Run targeted Go test to GREEN.

### Task 6: Combined Fixture and Batch Gates

**Files:**
- Modify only if required by tests: `tui/tests/e2e/fixtures/java-maven-project/**`

- [ ] Exercise a shared Java state-transition scenario proving Reviewer evidence and UT coverage semantics.
- [ ] Run Task 14 Reviewer E2E.
- [ ] Run Task 15 Unit Test E2E.
- [ ] Run `distribution/plugins`: `bun test`, `bun run build`.
- [ ] Run TUI: `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`.
- [ ] Cross-build Windows amd64 and Darwin amd64.
- [ ] Run plugin bundle, offline dependency, runtime boundary, execution-state validator, real Maven smoke.
- [ ] Run real OpenCode parity and require 17/17 if the locked runtime is executable in the verification environment.

### Task 7: Reports, Candidate Checkpoints, and Batch State

**Files:**
- Create: `docs/task-reports/task-14.md`
- Create: `docs/task-reports/task-15.md`
- Modify: `docs/execution-state.yaml`

- [ ] Record independent Task 14 and Task 15 implementation/checkpoint/Gate evidence.
- [ ] Keep Task 15 formally `pending` under the approved Batch exception while its implementation/report/Gate are complete.
- [ ] After both implementations verify, move only Task 14 to `awaiting_acceptance` with verification/taskGate pass and `humanAccepted=false`.
- [ ] Do not complete Task 14 or activate Task 15 before Batch human acceptance.
