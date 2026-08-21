# Task 16 Report — API Documentation Agent

## Overview

Implementation checkpoint: `c9796e2600129dfb0ce476f6192222c403964d48`

Task 16 在 Task 13 已交付的 `extract_api_spec` / `validate_api_example` / `write_document` Custom Tools 之上，增加企业级 `api-documentation` Agent。Agent 采用 deterministic-first 策略：接口结构、DTO、校验和错误码 provenance 以 Tool/代码证据为准；模型只负责组织文档和可选业务语义补充，未知字段固定写为 `Not determined from code`。

## Delivered

- `distribution/agents/api-documentation/manifest.yaml`
  - enterprise-controlled
  - fail-closed tool surface: only `read` / `grep` / `glob` / `extract_api_spec` / `validate_api_example` / `write_document` / `dify-query` are allowed
  - native `write` / `edit` / `bash` denied
  - `noFabrication: true`
- `distribution/agents/api-documentation/agent.md`
  - `extract_api_spec` first
  - bounded evidence expansion only
  - examples validated by `validate_api_example`
  - only `write_document` may persist output
  - Dify is optional business context and cannot override code evidence
  - exact Task 13 error provenance preserved: `DECLARED` / `REFERENCED` / `INFERRED`
- `distribution/agents/api-documentation/output-template.md`
  - method/path/source/confidence
  - request/response/validation/error codes/examples/evidence
  - explicit uncertainty marker
- `distribution/skills/api-documentation/SKILL.md`
  - closes the requiredSkills dependency declared by the manifest
- `tui/tests/e2e/api-documentation/api_doc_test.go`
  - contract tests for bounded tools, deterministic-first workflow, traceability/uncertainty and required Skill
- `tui/tests/parity/real_api_doc_smoke_test.go`
  - real locked-runtime gate for agent listing, read allow, `write_document` allow, cross-tool `write_test_file` deny, and actual docs artifact creation
- `scripts/run-real-agent-smoke.sh`
  - extended to run both existing Task 14/15 evidence and Task 16 API Documentation runtime evidence against OpenCode v1.18.11

## Verification performed in current environment

A minimal isolated Go mirror of the Task 16 contract test was executed because the current sandbox cannot clone the repository or run the repository Go 1.26.5 toolchain. The API Documentation contract suite passed 4/4 before the provenance spelling correction; after inspection against the authoritative Task 13 tool contract, all `REFERRED` occurrences were corrected to `REFERENCED` in prompt/template/test. This report does **not** treat that earlier mirror run as a fresh full Task 16 Gate.

## Verification still required before `awaiting_acceptance`

The following must be executed in a repository environment with Go 1.26.5, Bun and the locked OpenCode binary:

```bash
cd tui
GOTOOLCHAIN=local go test ./tests/e2e/api-documentation/ -count=1
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
go build ./...

cd ../distribution/plugins
bun test
bun run build

cd ../..
OPENCODE_BIN=<opencode-v1.18.11> ./scripts/run-real-agent-smoke.sh
./scripts/check-execution-state.sh
```

The real runtime run must generate fresh `tui/tests/parity/evidence/api-doc-agent-evidence.json` with all checks passing. Until that evidence exists, Task 16 must not be marked `awaiting_acceptance` or `completed`.

## Scope boundary

No Task 18 upgrade/rollback implementation is included or started.
