# Task 13 Report — 7 个企业 Custom Tools + 统一 Tool 基础设施 + 真实 Java Maven Fixture + E2E

## Overview

Checkpoint: `8b2e9ce6aaf753feda0227680f22261677475f17`

在 Task 12 的 Plugin 工程与安全基础之上，实现 7 个企业 Custom Tool（代码审查上下文收集 / 测试工程分析 / 测试文件写入 / 测试运行 / API 规范提取 / API 示例校验 / 文档写入），并建立统一 Tool 基础设施（ToolContext / ToolError / ToolResult / 9 类错误分类 / JSON Schema 校验 / 受控 exec / 受控文件系统），配真实 java-maven fixture 与 E2E（3 流程），每个 write/exec tool 配安全负向 Gate。

核心边界（本 Task 不可违反）：

- **写/执行能力仅经受控 Custom Tool**：企业 Agent 的原生 write/edit/bash 在 permissions 中 deny，write_test_file / run_project_test / write_document 是唯一获得写/执行能力的通道。
- **禁止任意 shell**：所有命令用 argv 数组（`node:child_process execFile`，无 shell），`extraArgs` 白名单 + 拒绝元字符/危险命令；run_project_test 禁止 curl/wget/sudo/rm/powershell 任意脚本、禁止 extraArgs shell 注入。
- **写文件严格限根**：write_test_file 只允许 test roots，默认禁止覆盖（`overwrite=true` 才可）；write_document 只允许 docs/doc/api-docs；两者都走 path-policy canonicalize + realpath，阻止 `../../src/main/...`、`C:\Windows\...`、symlink escape、生产源码目录。
- **API 规范禁止 hallucination**：`extract_api_spec` 是有限语法解析器（非 Java 编译器），不能确定一律标记 `Not determined from code`；错误码保留 DECLARED/REFERENCED/INFERRED。
- **审计不泄密**：所有 tool 的 guard.after 只记录 action/targetPath 相对路径/错误 category，不记录源码/Prompt/Output/Token/绝对路径。

## 统一 Tool 基础设施（`src/tools/`）

### 类型与结果（`types.ts`）

`ToolErrorCategory`（9 类）：`INVALID_INPUT / PATH_VIOLATION / PERMISSION_DENIED / DLP_BLOCKED / TIMEOUT / COMMAND_FAILED / PARSE_FAILED / NOT_SUPPORTED / INTERNAL_ERROR`。`ToolContext{sessionId, agent, projectRoot, audit, guard}`；`ToolResult<T> = {ok:true,data} | {ok:false,error}`；`ok()/err()` 构造器。

### 错误（`errors.ts`）

`ToolError extends Error`（`readonly category`、`override readonly cause?`、`toJSON()`）；工厂函数 `invalidInput/pathViolation/permissionDenied/dlpBlocked/timeoutError/commandFailed/parseFailed/notSupported/internalError`。

### 失败分类（`failure-classifier.ts`）

`classifyError`（ToolError→category、PathViolationError→PATH_VIOLATION、killed/ETIMEDOUT→TIMEOUT、其余→INTERNAL_ERROR）、`toToolError`（非 ToolError 包装）、`classifyCommandFailure`（exit code→COMMAND_FAILED / TIMEOUT）。

### Schema 校验（`schemas.ts`）

轻量 JSON Schema 校验器（`validateSchema/isValid/JsonSchema/SchemaProperty`），支持 type/required/additionalProperties/enum/minLength。每个 tool 用声明式 SCHEMA 校验入参。

### 受控 exec（`exec.ts`）

`execCommand(argv, {cwd, timeoutMs, env, maxBuffer})`：argv 数组 + `execFile`（无 shell）+ 显式 timeout（默认 30s）+ maxBuffer（默认 10MB）；`displayCommand`；`ensureCommandSucceeded`（非零/超时→classified ToolError）。

### 受控文件系统（`filesystem.ts`）

`resolveInRoot`（path-policy canonicalize + realpath，返回绝对路径或抛 PATH_VIOLATION）、`readTextFile/listDir/ensureDir/writeFileAtomic`（path policy → allowedRoots → overwrite 检查 → DLP → 原子 tmp+rename）。

## 7 个 Custom Tool

| Tool | 文件 | 能力 | 安全负向 Gate |
|------|------|------|---------------|
| `collect_review_context` | `collect-review-context.ts` | git diff 上下文（staged/unstaged/base-branch/commit/range/file-path），解析 unified diff→files/hunks/行号 | 只读 git 白名单命令 + 30s 超时 + 5MB 上限 + file-path 先 resolveInRoot |
| `analyze_test_project` | `analyze-test-project.ts` | 探测 build system（maven/gradle）/ test framework（junit5/junit4/mockito）/ test roots / wrapper / 既有 test pattern | 只读，无写入 |
| `write_test_file` | `write-test-file.ts` | 在 test root 写测试文件 | allowedRoots=testRoots、默认禁止覆盖、DLP、原子写、阻止生产源码目录/traversal |
| `run_project_test` | `run-project-test.ts` | 运行测试（mvnw/gradlew 优先），解析 surefire/gradle 汇总 | argv 数组、EXTRA_ARG_FORBIDDEN 白名单、禁 curl/wget/sudo/rm/powershell/元字符注入、超时+输出上限 |
| `extract_api_spec` | `extract-api-spec.ts` | Spring MVC controller→routes/params/DTO/enum/错误码 | 只读、有限语法解析、不可确定标 `Not determined from code`、不 fabricate |
| `validate_api_example` | `validate-api-example.ts` | 校验请求/响应示例 vs 提取的 spec（required/enum/Min/Max/type） | 只读、unknown field→warning 不报错 |
| `write_document` | `write-document.ts` | 写文档到 docs/doc/api-docs | allowedRoots=docs/doc/api-docs、禁 src/.git/项目外/绝对路径、overwrite=true |

## Fixture（`tui/tests/e2e/fixtures/java-maven-project/`）

真实 java-maven 工程：`pom.xml`（spring-boot-starter-web 3.2.0 + junit-jupiter 5.10.0 + mockito-core 5.7.0 + surefire 3.2.2）、`DemoController.java`（4 endpoint + @ExceptionHandler）、`UserDto.java`、`CreateUserRequest.java`、`UserStatus.java` enum、`UserService.java`、`UserServiceTest.java`、`mvnw` stub（可执行，输出确定性 Surefire 汇总 `Tests run: 3, Failures: 0, Errors: 0, Skipped: 0`）。

## E2E（`tests/e2e.test.ts`，3 流程）

1. **UT 流程**：analyze_test_project → write_test_file（test root）→ run_project_test（mvnw，解析汇总）。
2. **API Doc 流程**：extract_api_spec（DemoController，4 endpoint + DTO + enum + 错误码）→ validate_api_example（合法/非法示例）。
3. **Code Review 流程**：collect_review_context（真实 git 仓库 staged/unstaged diff）。

安全负向 Gate：write_test_file 写 `src/main/...`、`../../`、覆盖（overwrite=false）→ PATH_VIOLATION / 拒绝；run_project_test extraArgs 注入 `; curl`、`rm -rf`、`powershell` → 拒绝；write_document 写 `src/`、`.git/`、项目外 → PATH_VIOLATION。

## Full Gate Verification

| Gate | Result |
|------|--------|
| `bun test`（distribution/plugins） | PASS（155 tests，0 fail，17 files） |
| `bun run build`（bundle） | PASS（dist/index.js 59.47 KB） |
| `./scripts/check-plugin-bundle.sh` | PASS（bundle 自包含，无非 builtin import） |
| `./scripts/run-plugin-smoke.sh` | PASS（guard + audit + dify degradation，零公网） |
| `GOTOOLCHAIN=local go test ./... -count=1` | PASS（22 packages，无失败） |
| `GOTOOLCHAIN=local go test -race ./... -count=1` | PASS（22 packages，无竞态） |
| `GOTOOLCHAIN=local go vet ./...` | clean |
| `GOTOOLCHAIN=local go build ./...` | PASS |
| `GOOS=windows GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `GOOS=darwin GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `./scripts/check-runtime-boundary.sh` | PASS（no vendor DTO leakage） |
| `./scripts/check-execution-state.sh` | valid |
| `tests/execution-state/state_validator_test.sh` | PASS |
| **`OPENCODE_BIN=<abs> ./scripts/run-real-parity-smoke.sh`** | **PASS（real OpenCode v1.18.11，17/17 checks，failedChecks=0）** |

> Task 13 改了 Plugin/Tool 能力，故按 Batch 要求重新跑真实 OpenCode parity（非沿用 Task 11 evidence），结果 17/17 全绿。

## Task 13 Gate Checklist

- [x] 7 个企业 Custom Tool 全部实现（collect_review_context / analyze_test_project / write_test_file / run_project_test / extract_api_spec / validate_api_example / write_document）
- [x] 统一 ToolContext / ToolError / ToolResult + 9 类错误分类
- [x] failure-classifier（分类 + toToolError + classifyCommandFailure）
- [x] 真实 java-maven fixture（pom + controller + DTO + enum + service + test + mvnw stub）
- [x] E2E 3 流程全绿
- [x] 每个 write/exec tool 配安全负向 Gate（traversal/覆盖/注入/越根/生产目录）
- [x] argv 数组执行（无 shell），extraArgs 白名单 + 注入拒绝
- [x] 写文件严格限根 + 默认禁止覆盖 + DLP + 原子写
- [x] extract_api_spec 有限语法解析 + 不 fabricate（`Not determined from code`）
- [x] 全量 Gate 通过 + 真实 OpenCode parity 17/17

## Files Changed

| File | Action |
|------|--------|
| `distribution/plugins/src/tools/types.ts` | Create（ToolContext/ToolError/ToolResult/9 类 category） |
| `distribution/plugins/src/tools/errors.ts` | Create（ToolError + 工厂函数） |
| `distribution/plugins/src/tools/failure-classifier.ts` | Create |
| `distribution/plugins/src/tools/schemas.ts` | Create（轻量 JSON Schema 校验） |
| `distribution/plugins/src/tools/exec.ts` | Create（argv 数组 execFile，超时/maxBuffer） |
| `distribution/plugins/src/tools/filesystem.ts` | Create（resolveInRoot/readTextFile/writeFileAtomic） |
| `distribution/plugins/src/tools/collect-review-context.ts` | Create |
| `distribution/plugins/src/tools/analyze-test-project.ts` | Create |
| `distribution/plugins/src/tools/write-test-file.ts` | Create |
| `distribution/plugins/src/tools/run-project-test.ts` | Create |
| `distribution/plugins/src/tools/extract-api-spec.ts` | Create |
| `distribution/plugins/src/tools/validate-api-example.ts` | Create |
| `distribution/plugins/src/tools/write-document.ts` | Create |
| `distribution/plugins/src/index.ts` | Modify（re-export 7 tools + types/errors/failure-classifier） |
| `distribution/plugins/tests/helpers.ts` | Create（temp root + redacting audit context） |
| `distribution/plugins/tests/*.test.ts` | Create（schemas/failure-classifier/collect-review-context/analyze-test-project/write-test-file/run-project-test/extract-api-spec/validate-api-example/write-document/e2e，共 155 tests） |
| `tui/tests/e2e/fixtures/java-maven-project/**` | Create（真实 java-maven fixture） |
| `scripts/check-plugin-bundle.sh` | Modify（ALLOWED_BUILTINS 加 node:fs/path/os/child_process/bun:test） |

## Batch Implementation Exception（Task 12/13）

Task 12/13 按连续 Batch 执行，不修改 execution-state validator（保持「单 active Task + completed 必须 humanAccepted=true」）。

- Task 12 与 Task 13 的代码 + Gate + 证据已全部完成并分别提交（`ef82075` / `68f49e9`）。
- 按 validator「单 active Task」约束，本报告撰写时 Task 12 进入 `awaiting_acceptance`，Task 13 保持 `pending`（verification=pass / taskGate=pass 已就绪）。
- Task 12 + Task 13 两个 report 一起提交人工 Batch Review；Review 通过后按 validator 顺序纯状态收口：Task 12 completed（humanAccepted=true）→ Task 13 awaiting_acceptance → Task 13 completed。
- 不开始 Task 14。

## Gate 结论

- verification：pass
- Task Gate：pass
- 真实 OpenCode parity：17/17 PASS（v1.18.11）
- 状态：`pending`（Batch Exception，随 Task 12 一起提交人工 Review）

## 验收整改（Batch 12-13 Remediation）

Checkpoint：`9cbd5e64b04b82e494119a08d89376f202f8327f`（整改后全量 Gate 复跑证据）

人工验收指出的安全整改，本 Task 侧（7 个 Custom Tool）修复：

1. **write_test_file 删除调用方可控 testRoots**：移除 `WriteTestFileInput.testRoots` 与 SCHEMA 字段；test roots 一律来自 `analyze_test_project` 的 `detectTestRoots(ctx.projectRoot)`（detectTestRoots 改为 export）。无 test roots → `NOT_SUPPORTED`。新增 2 个负向测试（caller testRoots → INVALID_INPUT、无 test roots → NOT_SUPPORTED）。
2. **write_document 删除调用方可控 docsRoot**：移除 `WriteDocumentInput.docsRoot`；allowedRoots 固定为 `DEFAULT_DOCS_ROOTS`（docs/doc/api-docs）。新增 1 个负向测试。
3. **run_project_test 删除 extraArgs**：移除 `extraArgs` 字段、`EXTRA_ARG_FORBIDDEN` 与 build 注入，禁止 Maven/Gradle 扩展机制绕过受控测试执行。原 2 个 extraArgs 注入测试改为断言 `INVALID_INPUT`。
4. **collect_review_context ref validation / option-injection 防护**：新增 `validateRef()`（`GIT_REF_RE` 首字符必须字母数字、拒绝 `..`/`@{`/元字符），对 baseBranch/commit/rangeFrom/rangeTo 校验。新增 4 个负向测试（`--output`、`--upload-pack`、`--git-dir`、shell 元字符 → INVALID_INPUT）。
5. **Real Maven Integration Smoke**：新增 `scripts/run-real-maven-smoke.sh`，拷贝 fixture 到临时目录、删除 mvnw stub、`mvn -B test` 真实编译执行 JUnit，断言 `BUILD SUCCESS` + `Tests run: N, Failures: 0, Errors: 0, Skipped: 0`。fixture 补 `spring-boot-starter-validation` 依赖 + jakarta.validation imports（@NotBlank/@Email/@Min/@Max）使真实 Maven 编译通过。

整改后 Gate 复跑（Task 13 侧）：`bun test` 178 pass（原 155）、`bun run build` 61.84 KB、`./scripts/check-plugin-bundle.sh` PASS、`./scripts/run-plugin-smoke.sh` PASS、`./scripts/run-real-maven-smoke.sh` PASS（fixture 真实编译运行绿）、`GOTOOLCHAIN=local go test ./... -count=1` 22 packages PASS、`-race` PASS、`go vet` clean、`go build` PASS、Windows/darwin 交叉编译 PASS、`./scripts/check-runtime-boundary.sh` PASS、`OPENCODE_BIN=<abs> ./scripts/run-real-parity-smoke.sh` 17/17 PASS（v1.18.11）。

## 验收整改（Round 2 — 4 个 Tool 修复 + 真实 OpenCode Plugin 注册）

人工二次验收指出的 Blocking 项，本 Task 侧（Custom Tool）修复：

1. **unknown API example field → invalid（非 warning）**：`validate-api-example.ts` 对 `example` 中不存在于提取 DTO 的字段由 warning 改为 `unknown field "..." (not in extracted DTO)` error，杜绝「无中生有」的字段通过校验。
2. **Windows wrapper 支持**：`run-project-test.ts` 新增 `WRAPPERS`（maven: `mvnw`/`mvnw.cmd`；gradle: `gradlew`/`gradlew.bat`）+ `detectWrapper()`，`buildCommand()` 优先使用平台正确 wrapper，`.cmd`/`.bat` 在 Windows 下可执行。
3. **API endpoint path 合并 class basePath**：`extract-api-spec.ts` 新增 `joinPaths(basePath, endpointPath)`，`/api/users` + `/{id}` → `/api/users/{id}`；`execute()` 对每个 endpoint 应用合并后的 path。
4. **package-aware DTO 查找 + params 逗号解析（泛型）+ method-scoped 错误码**：`extract-api-spec.ts` 新增 `findJavaFileByImport()`（按 import 的 package 路径 + 类名在项目内定位 DTO）、`splitTopLevel()`（按 `<`/`>` 深度拆分逗号，正确处理 `Map<String, List<Foo>>` 泛型）、`balancedBlock()`（提取方法体）。错误码拆分 DECLARED（`@ExceptionHandler`/`@ResponseStatus`，全类）与 REFERENCED（`throw new XxxException`，方法体 scope）。
5. **test root 标准布局安全推导**：`analyze-test-project.ts` 的 `detectTestRoots()` 在物理存在的 test root 之外，对 Maven/Gradle 项目按标准布局（`src/test/java`，gradle 追加 `src/test/kotlin`）推导，未测试过的工程也有约定目标。

OpenCode Plugin Adapter（跨 Task 12/13）注册上述 7 个 tool + dify-query，见 Task 12 Round 2。

Round 2 Gate 复跑（Task 13 侧）：`bun test` 201 pass、`bun run build` 0.52 MB、`./scripts/check-plugin-bundle.sh` PASS、`./scripts/run-plugin-smoke.sh` PASS、`./scripts/run-real-maven-smoke.sh` PASS、`OPENCODE_BIN=<abs> ./scripts/run-real-plugin-smoke.sh` PASS、`OPENCODE_BIN=<abs> ./scripts/run-real-parity-smoke.sh` 17/17 PASS（v1.18.11）、`GOTOOLCHAIN=local go test ./... -count=1` 22 packages PASS、`-race` PASS、`go vet` clean、`go build` PASS、Windows/darwin 交叉编译 PASS、`./scripts/check-runtime-boundary.sh` PASS、`./scripts/check-execution-state.sh` valid。

## 验收整改（Round 3 — Windows wrapper 代码级验证 + 批处理参数防护）

人工第二轮「有条件通过」指出的 Blocking 项，本 Task 侧修复：

1. **Windows `.cmd`/`.bat` 代码级验证（非真实 Windows 主机执行）**：此前 `.cmd`/`.bat` 仍走 `execFile()`，Windows 无法直接 spawn 批处理。`src/tools/exec.ts` 新增 `resolveExecArgv(argv, platform)`：`argv[0]` 匹配 `\.(cmd|bat)$` 且 platform 为 `win32` 时，路由为 `cmd.exe /d /s /c <joined>`（argv 数组，非 `shell:true`，不引入 POSIX shell）；`execCommand` 改为先经 `resolveExecArgv` 再 `execFile`。`displayCommand` 负责安全 join 单个命令行参数。**验证方式为 `resolveExecArgv(..., "win32")` 单元测试（`tests/exec.test.ts`）模拟 win32 平台路由；真实 Windows 主机上的 `.cmd`/`.bat` 端到端执行未在本轮验证，延后至发行验收（Task 17/18 离线发行包 / Task 21 Release Parity Certification）在真实 Windows x64 环境复核。**
2. **批处理路径注入防护**：`src/tools/run-project-test.ts` 新增 `UNSAFE_BUILD_ARG = /[\s&|<>^%!"'`();]/` 与 `assertSafeBuildArgs()`，对调用方 `module`/`testClass`/`testMethod`/`profiles` 在 batch 路径生效前拒绝 shell/cmd 元字符，保证 `/c` 命令行参数无活元字符。新增负向测试（`extraArgs` 移除、`testClass`/`module`/`profiles` 元字符 → `INVALID_INPUT`）。
3. **wrapper 检测补 Windows-only 工程**：`analyze-test-project.ts` 的 `detectWrapper` 与 `run-project-test.ts` 的 `WRAPPERS` 同时识别 `mvnw.cmd`/`gradlew.bat`，`buildCommand()` 对仅有 `.cmd`/`.bat` 的 checkout 也能正确选择批处理 wrapper。新增 `tests/exec.test.ts`（`resolveExecArgv` 路由）与 analyze-test-project/run-project-test 的 wrapper 测试。

Round 3 Gate 复跑（Task 13 侧）：`bun test` 213 pass、`bun run build` 0.52 MB、`./scripts/check-plugin-bundle.sh` PASS、`./scripts/run-plugin-smoke.sh` PASS、`./scripts/run-real-maven-smoke.sh` PASS、`OPENCODE_BIN=<abs> ./scripts/run-real-plugin-smoke.sh` PASS（`/experimental/tool/ids` 断言 8/8）、`OPENCODE_BIN=<abs> ./scripts/run-real-parity-smoke.sh` 17/17 PASS（v1.18.11）、`GOTOOLCHAIN=local go test ./... -count=1` 22 packages PASS、`-race` PASS、`go vet` clean、`go build` PASS、Windows/darwin 交叉编译 PASS、`./scripts/check-runtime-boundary.sh` PASS、`./scripts/check-execution-state.sh` valid、`tests/execution-state/state_validator_test.sh` PASS。
