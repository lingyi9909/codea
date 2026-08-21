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
- `tui/tests/e2e/api-documentation/api_doc_test.go`

## Functional Workflow E2E remediation

The original runtime smoke only proved registration/permissions. The functional gap is now implemented as:

```text
APIDOCFLOW
→ extract_api_spec
→ validate_api_example
→ write_document
→ docs/api-demo.md
```

`tests/fixtures/real-parity/fake_api_doc_model.py` consumes the actual extraction Tool Result, passes that exact spec to `validate_api_example`, renders Markdown from the structured result, preserves only `DECLARED` / `REFERENCED` / `INFERRED`, keeps unresolved semantics as `Not determined from code`, and finally calls `write_document`.

The combined regression model `tests/fixtures/real-parity/fake_model.py` also recognizes `APIDOCFLOW` and delegates to the same `fake_api_doc_model.decide(...)`, so focused Task 16 smoke and combined Task 14–16 regression do not use divergent workflow semantics.

`tui/tests/parity/real_api_doc_smoke_test.go` requires:

- `extract_api_spec` called once and succeeded
- `validate_api_example` called once and succeeded
- `write_document` called once and succeeded
- persisted Markdown contains code-derived `DemoController`, `GET /api/users/{id}`, `POST /api/users`, `CreateUserRequest`, `@NotBlank`, `@Email`, `@Min(1)`, `@Max(120)`
- error provenance remains `DECLARED` / `REFERENCED` / `INFERRED` and rejects `REFERRED`
- unresolved semantics remain `Not determined from code`
- example validation is recorded as PASS

`scripts/run-real-api-doc-smoke.sh` requires fresh `api-doc-agent-evidence.json` to be exactly 9/9 PASS against real locked OpenCode v1.18.11.

## Task 16 acceptance boundary

Task 16 acceptance is intentionally independent from Task 17 platform-release evidence.

Task 16 requires exactly these evidence groups:

1. **API Documentation functional/runtime evidence**
   - `run-real-api-doc-smoke.sh`
   - real OpenCode v1.18.11
   - `api-doc-agent-evidence.json` = 9/9 PASS
2. **Direct regression**
   - Go test / race / vet / build
   - Bun test / build
3. **Existing runtime regression**
   - Task 14/15 agent smoke
   - OpenCode parity/runtime regression

The following Task 17 evidence does **not** block Task 16:

- three-platform release build
- macOS native offline install/runtime smoke
- Windows x64 native offline install/runtime smoke

Those remain exclusively under Task 17.

## Verification status

Fresh Task 16 evidence is still required before `awaiting_acceptance`:

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

It must show OpenCode `1.18.11`, `failedChecks=0`, `passedChecks=totalChecks=9`, and:

- `workflowExtractSucceeded=true`
- `workflowValidateSucceeded=true`
- `workflowWriteSucceeded=true`
- `workflowDocumentValid=true`

Until these Task 16-specific fresh gates pass, Task 16 remains `blocked`. Task 17 native install evidence is not part of this blocker.

## Scope boundary

Task 17 release/native evidence is tracked separately. Task 18 remains untouched and pending.
