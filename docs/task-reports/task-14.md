# Task 14 Report — Code Reviewer Agent

## Overview

Checkpoint: `c9064c20642702fd9b2fd0a577b8acbf802f353f`

在 Task 13 的 `collect_review_context` Tool 之上，交付企业级 Code Reviewer Agent（enterprise-controlled），以 Manifest + Prompt + JSON Schema 定义结构化、可追溯、只读的代码审查。原生 `write`/`edit`/`bash` 在 permissions 中 deny，审查证据只能来自 `collect_review_context` 与受限仓库读取。

核心边界（本 Task 不可违反）：

- **只读**：`write`/`edit`/`bash` 全部 deny，Reviewer 无任何写/执行能力。
- **Scope-first**：必须先 `collect_review_context` 收集变更上下文，再进行仓库扩展；收集必须先于 grep/glob 展开。
- **调用链展开有界**：Java/Spring 调用链默认最大深度 3（Controller → Service → Repository → Mapper → DTO → Domain），在下结论前必须读取下游实现。
- **finding 契约**：每条 finding 必须含 `file`/`lineRange`/`severity`/`title`/`description`/`evidence`/`introducedByChange`/`confidence`/`recommendation`；严重级别 Critical/Major/Minor/Suggestion。
- **confidence 门槛**：低于 `0.80` 的观察进入 `uncertainObservations`，不得作为正式 finding。
- **clean-diff 行为**：无有效变更时返回 `findings=[]`，禁止虚构。
- **Dify 仅作知识**：`dify-query` 只提供业务知识，`must not be used as code evidence`，降级时 `review must continue` 并标记 `businessKnowledgeUnavailable`。

## 交付物

| 文件 | 说明 |
|------|------|
| `distribution/agents/code-reviewer/agent.md` | 审查工作流、严重级别、证据要求、confidence 门槛、introducedByChange 规则、clean-diff 与 Dify 降级 |
| `distribution/agents/code-reviewer/manifest.yaml` | name/version/mode/requiredSkills/optionalSkills/tools 白名单 + deny + constraints（maxCallChainDepth:3, minimumFindingConfidence:0.80, readOnly:true） |
| `distribution/agents/code-reviewer/output-schema.json` | JSON Schema：summary/scope/findings/observations/uncertainObservations/reviewStats/businessKnowledgeUnavailable |
| `tui/tests/e2e/code-review/review_test.go` | 契约 E2E：manifest 只读与工具边界、scope-first 与调用链、finding 质量与 clean-diff、Dify 降级、JSON Schema |

## 契约 E2E

`GOTOOLCHAIN=local go test ./tests/e2e/code-review/ -count=1` → 5/5 PASS：

- TestReviewerManifestIsReadOnlyAndToolBounded
- TestReviewerWorkflowIsScopeFirstAndExpandsBoundedCallChains
- TestReviewerFindingQualityAndCleanDiffRules
- TestReviewerDifyIsOptionalKnowledgeOnly
- TestReviewerOutputSchemaIsJSONSchema

## Full Gate Verification

| Gate | Result |
|------|--------|
| `go test ./tests/e2e/code-review/ ./tests/e2e/unit-test/`（契约） | PASS（11 tests） |
| `bun test`（distribution/plugins） | PASS（241 tests，0 fail） |
| `bun run build`（bundle） | PASS（dist/index.js 0.52 MB） |
| `./scripts/check-plugin-bundle.sh` | PASS（bundle 自包含，offline-safe） |
| `./scripts/run-plugin-smoke.sh` | PASS（8-tool adapter guard chain，零公网） |
| `./scripts/check-runtime-boundary.sh` | PASS（no vendor DTO leakage） |
| `go build ./...` | PASS |
| `go vet ./...` | clean |
| `go test ./... -count=1` | PASS（24 packages） |
| `go test -race ./... -count=1` | PASS（无竞态） |
| `GOOS=windows GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `GOOS=darwin GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `./scripts/run-real-maven-smoke.sh` | PASS（真实 Maven fixture JUnit 绿） |
| `OPENCODE_BIN=… ./scripts/run-real-parity-smoke.sh` | PASS（17/17，v1.18.11） |
| `OPENCODE_BIN=… ./scripts/run-real-plugin-smoke.sh` | PASS（serve load + 8/8 tool 注册） |
| `./scripts/check-execution-state.sh` | PASS（state valid） |
