# Task 15 Report — Unit Test Generator Agent

## Overview

Checkpoint: `d4333bccf09a1c397d59d69e9776cf1d66755943`

在 Task 13 的 `analyze_test_project` / `write_test_file` / `run_project_test` / failure-classifier 之上，交付企业级 Unit Test Generator Agent（enterprise-controlled），实现 JUnit 5 测试生成与受控自动修复。原生 `write`/`edit`/`bash` 全部 deny，写与执行能力只经 `write_test_file` / `run_project_test` 通道。

核心边界（本 Task 不可违反）：

- **analyze-first**：7 步工作流 `analyze → plan → generate → write → run → classify → repair`；`testFramework = unknown` 时 `do not guess`，直接停止。
- **写只经 `write_test_file`**：默认 `overwrite=false`，仅允许覆盖 `filesCreatedByCurrentRun` 记录的本轮文件；不得编辑生产代码、不得改动已有测试、不得用 `@Disabled`/弱化断言。
- **跑只经 `run_project_test`**：最小范围顺序 method → class → module；禁止 `extraArgs` 注入。
- **修复有界**：最多 3 次修复（Attempt 0 → Repair 1/2/3 → STOP），遇 dependency/infrastructure/安全类失败立即停。
- **8 类错误分类**：COMPILE_ERROR / DEPENDENCY_ERROR / TEST_FAILURE / ASSERTION_FAILURE / MOCK_CONFIGURATION / TEST_RUNTIME_ERROR / TIMEOUT / INFRASTRUCTURE_ERROR，其中 repairable=true 的才进入 repair。
- **产品缺陷止损**：`PRODUCT_CODE_DEFECT_SUSPECTED` 触发 stop，禁止改业务代码。
- **真实结果收口**：最终报告只基于 `run_project_test` 的结构化输出（passed/failed/errors/skipped/duration/failureDetails/exitCode/category），`Never claim tests pass without run_project_test`。

## Enterprise Runtime Integration（最后一公里）

`manifest.yaml` + `agent.md` 由 `tui/internal/agent` 物化为 `OPENCODE_CONFIG_DIR/agents/unit-test-generator.md`，在 runtime 启动前由 `main.go` 冷启动写入，使 unit-test-generator 作为真实的一等 agent 出现在 `/agent` 列表中，权限由 OpenCode 服务端强制执行。

**Tool Whitelist fail-closed（server-side）**：materializer 完整解析 manifest 的 tools map，生成 `permission: {"*": deny, <allow-tool>: allow}`。未列入 allow 的 tool（含 `write`/`edit`/`bash`/`write_document`）一律继承 `deny`，因此 UT 只允许 `read`/`grep`/`glob`/`analyze_test_project`/`write_test_file`/`run_project_test`/`dify-query`。真实 runtime 断言：unit-test-generator → `write_test_file` ALLOW、`write_document` DENY、`collect_review_context` DENY。

**写入所有权（server-side，非 Prompt 约束）**：`write_test_file` 的 `overwrite` 不再是 Prompt 提示，而是由 Plugin 服务端基于 `(sessionID + agent)` 维护的 `createdFiles` Set 强制执行：

- 已有测试文件 + `overwrite=true` → **DENY**（权限错误，工具不成功）
- 本轮刚创建文件 + `overwrite=true` → **ALLOW**（修复工作流覆盖自己刚写的测试）
- 其他 session 创建的文件 + `overwrite` → **DENY**
- 生产代码路径 → **DENY**

所有权记录与覆盖判定位于 `writeFileAtomic` 内，`write_test_file` 是唯一获得 `ownershipFactory` 的写工具；模型即便尝试 `overwrite` 也只会收到 deny。

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

## 真实 Agent Workflow E2E（真实 Maven）

`run-real-agent-smoke.sh` 现在丢弃 `mvnw` stub，使 unit-test-generator 的 `run_project_test` 回退到真实 `mvn test`：

- **成功链路**：`analyze_test_project → write_test_file(GeneratedFlowTest.java) → run_project_test(真实 mvn test)`，断言 `GeneratedFlowTest.java` 落盘、Surefire 逐类报告存在（证明被真实编译执行）、exitCode=0、Tests run≥1、Failures=0、Errors=0。
- **最终结论来自 run_project_test**：fake model 读取 `run_project_test` 的 Tool Result（`category`/`exitCode`/`passed`/`failed`/`errors`）生成结论，不再硬编码 `"UT workflow complete"`。
- **确定性失败链路**：`UTFLOW_FAIL` 写入故意失败的 JUnit，`run_project_test` 返回 exitCode≠0，Agent 最终结论必须 FAIL（不得输出 PASS）。

## Full Gate Verification

| Gate | Result |
|------|--------|
| `go test ./tests/e2e/code-review/ ./tests/e2e/unit-test/`（契约） | PASS（12 tests） |
| `bun test`（distribution/plugins） | PASS（245 tests，0 fail） |
| `bun run build`（bundle） | PASS（dist/index.js 0.52 MB） |
| `./scripts/check-plugin-bundle.sh` | PASS（bundle 自包含，offline-safe） |
| `./scripts/run-plugin-smoke.sh` | PASS（8-tool adapter guard chain，零公网） |
| `./scripts/check-runtime-boundary.sh` | PASS（no vendor DTO leakage） |
| `go build ./...` | PASS |
| `go vet ./...` | clean |
| `go test ./... -count=1` | PASS（23 packages） |
| `go test -race ./... -count=1` | PASS（无竞态） |
| `GOOS=windows GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `GOOS=darwin GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `./scripts/run-real-maven-smoke.sh` | PASS（真实 Maven fixture JUnit 绿） |
| `OPENCODE_BIN=… ./scripts/run-real-parity-smoke.sh` | PASS（17/17，v1.18.11） |
| `OPENCODE_BIN=… ./scripts/run-real-plugin-smoke.sh` | PASS（serve load + 8/8 tool 注册） |
| `OPENCODE_BIN=… ./scripts/run-real-agent-smoke.sh` | PASS（code-reviewer + unit-test-generator 出现在 /agent；read allow、write deny + Custom Tool 白名单 fail-closed cross-tool deny + Reviewer/UT 工作流真实 E2E（UT 用真实 Maven + 结论来自 run 结果 + 失败链路），15/15） |
| `./scripts/check-execution-state.sh` | PASS（state valid） |
