# Task 16 Report — API Documentation Agent

## Overview

Implementation checkpoint: `3feb08ba0a086af501574ca1b09dc0216add08cc`

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

The previous runtime smoke only proved registration/permissions. This gap is now implemented as a separate Task 16 functional smoke so Task 14/15's accepted deterministic model remains isolated.

New files/changes:

- `tests/fixtures/real-parity/fake_api_doc_model.py`
  - deterministic `APIDOCFLOW`
  - `extract_api_spec(controllerFile=DemoController.java)`
  - consumes the **actual structured Tool Result**
  - passes that exact spec into `validate_api_example`
  - renders Markdown from the extracted endpoints/DTO/validation/error provenance
  - calls `write_document(docs/api-demo.md)`
  - does not hard-code endpoint method/path/DTO/provenance as the final document source
- `tui/tests/parity/real_api_doc_smoke_test.go`
  - requires all three workflow tools to be called and succeed through the real runtime
  - checks the persisted Markdown contains code-derived `DemoController`, `GET /api/users/{id}`, `POST /api/users`, `CreateUserRequest`, `@NotBlank`, `@Email`, `@Min(1)`, `@Max(120)`
  - checks error provenance remains `DECLARED` / `REFERENCED` / `INFERRED` and explicitly rejects the historical typo `REFERRED`
  - checks unresolved semantics remain `Not determined from code`
  - checks example validation is recorded as PASS
- `scripts/run-real-api-doc-smoke.sh`
  - starts deterministic fake API-doc model
  - starts real locked OpenCode v1.18.11
  - materializes real enterprise agents
  - loads the real plugin bundle
  - executes `TestRealAPIDocEvidence`
  - requires fresh `api-doc-agent-evidence.json` to be exactly 9/9 PASS
- `scripts/run-real-agent-smoke.sh`
  - restored to the Task 14/15-only 15/15 regression so Task 16 cannot perturb the accepted Reviewer/UT fake-model lifecycle
- `.github/workflows/task16-17-gates.yml`
  - automated runner entry for Go/plugin regression, Task 14/15 15/15 regression and Task 16 real workflow smoke

## Verification status

The required functional E2E is now encoded, but this report does **not** claim fresh runtime PASS yet. The current ChatGPT execution sandbox cannot run the repository with Go 1.26.5/Bun/OpenCode v1.18.11. A GitHub Actions gate entry was committed, but no completed workflow run/evidence is currently visible through the connected execution interface, so it is not counted as verification evidence.

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
