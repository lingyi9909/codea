# Task 15 Report — Unit Test Generator Agent

## Overview

Checkpoint: `c9064c20642702fd9b2fd0a577b8acbf802f353f`

在 Task 13 的 `analyze_test_project` / `write_test_file` / `run_project_test` / failure-classifier 之上，交付企业级 Unit Test Generator Agent（enterprise-controlled），实现 JUnit 5 测试生成与受控自动修复。原生 `write`/`edit`/`bash` 全部 deny，写与执行能力只经 `write_test_file` / `run_project_test` 通道。

核心边界（本 Task 不可违反）：

- **analyze-first**：7 步工作流 `analyze → plan → generate → write → run → classify → repair`；`testFramework = unknown` 时 `do not guess`，直接停止。
- **写只经 `write_test_file`**：默认 `overwrite=false`，仅允许覆盖 `filesCreatedByCurrentRun` 记录的本轮文件；不得编辑生产代码、不得改动已有测试、不得用 `@Disabled`/弱化断言。
- **跑只经 `run_project_test`**：最小范围顺序 method → class → module；禁止 `extraArgs` 注入。
- **修复有界**：最多 3 次修复（Attempt 0 → Repair 1/2/3 → STOP），遇 dependency/infrastructure/安全类失败立即停。
- **8 类错误分类**：COMPILE_ERROR / DEPENDENCY_ERROR / TEST_FAILURE / ASSERTION_FAILURE / MOCK_CONFIGURATION / TEST_RUNTIME_ERROR / TIMEOUT / INFRASTRUCTURE_ERROR，其中 repairable=true 的才进入 repair。
- **产品缺陷止损**：`PRODUCT_CODE_DEFECT_SUSPECTED` 触发 stop，禁止改业务代码。
- **真实结果收口**：最终报告只基于 `run_project_test` 的结构化输出（passed/failed/errors/skipped/duration/failureDetails/exitCode/category），`Never claim tests pass without run_project_test`。

## 交付物

| 文件 | 说明 |
|------|------|
| `distribution/agents/unit-test-generator/agent.md` | 7 步工作流、Test Plan 契约、所有权与覆盖安全、修复有界、最终报告规则 |
| `distribution/agents/unit-test-generator/manifest.yaml` | name/version/mode/requiredSkills/tools 白名单 + deny + constraints（maxRepairAttempts:3, neverOverwriteExisting:true） |
| `distribution/agents/unit-test-generator/error-categories.yaml` | 8 类失败 + repairable 边界 + PRODUCT_CODE_DEFECT_SUSPECTED terminalOutcome |
| `tui/tests/e2e/unit-test/ut_gen_test.go` | 契约 E2E：manifest 权限、workflow、Test Plan 与所有权安全、修复有界、错误分类、真实结果收口 |

## 契约 E2E

`GOTOOLCHAIN=local go test ./tests/e2e/unit-test/ -count=1` → 6/6 PASS：

- TestUTManifestEnforcesControlledTools
- TestUTWorkflowAnalyzePlanGenerateRunRepair
- TestUTPlanAndOwnershipSafety
- TestUTRepairIsBounded
- TestUTErrorCategories
- TestUTFinalReportUsesRealToolResult

## 真实 Maven Fixture

`./scripts/run-real-maven-smoke.sh` → PASS：真实 Maven（非 mvnw stub）编译并执行 `tui/tests/e2e/fixtures/java-maven-project` 的 JUnit，Surefire 报绿。

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
