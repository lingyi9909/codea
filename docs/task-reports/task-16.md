# Task 16 Report — API Documentation Agent

## Overview

Implementation checkpoint: `83d9d6ff8018883d31c584752fe67610834033ff`

Task 16 在 Task 13 已交付的 `extract_api_spec` / `validate_api_example` / `write_document` Custom Tools 之上，增加企业级 `api-documentation` Agent。Agent 采用 deterministic-first 策略：接口结构、DTO、校验和错误码 provenance 以 Tool/代码证据为准；模型只负责组织文档和可选业务语义补充，未知字段固定写为 `Not determined from code`。

## Delivered

- `distribution/agents/api-documentation/manifest.yaml`
  - enterprise-controlled
  - fail-closed tool surface: only `read` / `grep` / `glob` / `extract_api_spec` / `validate_api_example` / `write_document` / `dify-query` are allowed
  - native `write` / `edit` / `bash` denied
  - `noFabrication: true`
- `distribution/agents/api-documentation/agent.md`
  - `extract_api_spec` first
  - examples validated by `validate_api_example`
  - only `write_document` may persist output
  - Dify is optional business context and cannot override code evidence
  - exact Task 13 provenance preserved: `DECLARED` / `REFERENCED` / `INFERRED`
- `distribution/agents/api-documentation/output-template.md`
  - method/path/source/confidence
  - request/response/validation/error codes/examples/evidence
  - explicit uncertainty marker
- `distribution/skills/api-documentation/SKILL.md`
  - closes the requiredSkills dependency declared by the manifest
- `tui/tests/e2e/api-documentation/api_doc_test.go`
  - contract tests for bounded tools, deterministic-first workflow, traceability/uncertainty and required Skill

## Functional Workflow E2E remediation

The original runtime smoke only proved registration/permissions. The functional gap is now implemented as a real Task 16 workflow:

```text
APIDOCFLOW
→ extract_api_spec
→ validate_api_example
→ write_document
→ docs/api-demo.md
```

### Deterministic API-doc model

`tests/fixtures/real-parity/fake_api_doc_model.py` implements the workflow and deliberately consumes Tool Results:

- calls `extract_api_spec(controllerFile=DemoController.java)`
- reads the actual structured extraction Tool Result
- passes that exact `spec` object into `validate_api_example`
- renders Markdown from extracted endpoints, DTO fields, validation annotations and error-code provenance
- preserves only `DECLARED` / `REFERENCED` / `INFERRED`
- renders unresolved semantics as `Not determined from code`
- calls `write_document(docs/api-demo.md)` with that rendered output

### Shared full-regression fake model

A review found that the combined `run-real-agent-smoke.sh` still starts `tests/fixtures/real-parity/fake_model.py`, while `TestRealAPIDocEvidence` now sends `APIDOCFLOW`. The dedicated Task 16 model already implemented APIDOCFLOW, but the shared Reviewer/UT fake model did not, so the combined Task 14–16 regression would terminate without the three API-doc Tool calls.

This is fixed at checkpoint `83d9d6ff8018883d31c584752fe67610834033ff`:

- shared `fake_model.py` explicitly recognizes `APIDOCFLOW`
- it delegates to `fake_api_doc_model.decide(prompt, messages)`
- therefore focused Task 16 smoke and combined Agent regression use the **same Tool-Result-driven workflow implementation**, rather than two duplicated/hard-coded Markdown generators
- existing Reviewer/UT state machines remain unchanged

### Runtime assertions

`tui/tests/parity/real_api_doc_smoke_test.go` requires:

- `extract_api_spec` called once and succeeded
- `validate_api_example` called once and succeeded
- `write_document` called once and succeeded
- persisted Markdown contains code-derived `DemoController`, `GET /api/users/{id}`, `POST /api/users`, `CreateUserRequest`, `@NotBlank`, `@Email`, `@Min(1)`, `@Max(120)`
- error provenance remains `DECLARED` / `REFERENCED` / `INFERRED` and rejects `REFERRED`
- unresolved semantics remain `Not determined from code`
- example validation is recorded as PASS

`scripts/run-real-api-doc-smoke.sh` starts the dedicated deterministic API-doc model plus real locked OpenCode v1.18.11 and requires fresh `api-doc-agent-evidence.json` to be exactly 9/9 PASS.

`scripts/run-real-agent-smoke.sh` continues to exercise the combined Reviewer/UT/API Documentation runtime regression; its shared fake model can now execute the same APIDOCFLOW semantics.

## Verification status

The implementation blocker is fixed, but this report does **not** claim fresh runtime PASS yet. The current execution sandbox cannot run the repository with Go 1.26.5/Bun/OpenCode v1.18.11. No authoritative completed workflow/native evidence is currently available through the connected execution interface, so no evidence is fabricated.

Fresh acceptance evidence must include:

```bash
cd tui
GOTOOLCHAIN=local go test ./tests/e2e/api-documentation/ -count=1
GOTOOLCHAIN=local go test ./... -count=1
GOTOOLCHAIN=local go test -race ./... -count=1
GOTOOLCHAIN=local go vet ./...
GOTOOLCHAIN=local go build ./...

cd ../distribution/plugins
bun test
bun run build

cd ../..
OPENCODE_BIN=<opencode-v1.18.11> ./scripts/run-real-agent-smoke.sh
OPENCODE_BIN=<opencode-v1.18.11> ./scripts/run-real-api-doc-smoke.sh
```

Required committed artifact:

```text
tui/tests/parity/evidence/api-doc-agent-evidence.json
```

It must show OpenCode `1.18.11`, `failedChecks=0`, and `passedChecks=totalChecks=9`, including:

- `workflowExtractSucceeded=true`
- `workflowValidateSucceeded=true`
- `workflowWriteSucceeded=true`
- `workflowDocumentValid=true`

Until fresh evidence exists, Task 16 remains `blocked` and must not become `awaiting_acceptance` or `completed`.

## Scope boundary

No Task 18 upgrade/rollback implementation is included or started.
