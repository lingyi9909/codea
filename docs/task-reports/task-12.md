# Task 12 Report — 安全规则、DLP、Dify、审计 + Plugin 工程骨架

## Overview

Checkpoint: `be133eda923c104aa924b617c4b6e54136bc01da`

建立 `distribution/plugins/` Plugin 工程骨架（TypeScript + Bun + 自包含 ESM bundle，非 npm+esbuild），并实现安全基础：Command Policy（风险分级）、4 层 DLP、路径策略（canonicalize/realpath）、Permissions（General 与 3 个企业 Agent 分离）、Dify 熔断器、审计日志、Runtime Security Guard。

核心边界（本 Task 不可违反）：

- **工具链**：严格 TypeScript + target Bun + `bun.lock` + `bun test` + `bun build` + self-contained ESM；开发机安装 Bun 不违反离线约束（仅开发期，运行时无 bun/npm 依赖）。实际锁定 `bun 1.3.14`。
- **不侵入 OpenCode Core**：全部能力在 `distribution/plugins/` 内实现，零 OpenCode 仓库 patch。
- **Command Policy 不做「前缀安全=整体安全」**：按 shell 元字符 tokenize，危险 token → deny，组合（pipe/redirect/chain/subcmd）→ ask，白名单 → safe，其余 → ask。
- **DLP 4 层**：secret block（Authorization/Bearer/PAT/AWS key/API key/password/token/secret）、敏感路径 redact、内容 redact、输出最小化。
- **Path Policy 不做 `startsWith(projectRoot)`**：canonicalize + realpath 后验证 `isWithin`，覆盖 `../` traversal、绝对路径、Windows drive/UNC、symlink escape、大小写/分隔符绕过。
- **Permissions**：企业 Agent（code-reviewer / unit-test-generator / api-documentation）的 write/edit/bash 必须 deny，仅通过受控 custom tool 获得 write/exec；General 保留原生能力（write/edit/bash=ask）。
- **Dify 熔断器**：3 次连续失败 → open 60s → degraded → half-open → closed；降级结果 `degraded=true` 不抛异常。
- **审计日志**：禁止记录完整源码、完整 Prompt、完整 Tool Output、Token/API Key、绝对用户路径、password、Authorization；相对路径化 + DLP redact；`log()` 永不抛异常。

## 语义实现

### Command Policy（`src/security/command-policy.ts`）

`DANGEROUS_COMMANDS`（sudo/doas/su/curl/wget/nc/netcat/telnet/rm/rmdir/del/sh/bash/zsh/cmd/powershell/pwsh/remove-item/invoke-webrequest/invoke-expression/start-process 等）、`SAFE_COMMANDS` 白名单（git status/diff/log/show/rev-parse/branch/ls-files/shortlog、ls/pwd/cat/head/tail/grep/find 等）、`tokenize()`（按 `[\s;&|()<>"'\\]+` 拆分）、`analyzeCommand()`。危险 token → `deny`；`hasPipe/hasRedirect/hasSubCmd/hasChain` → `ask`；白名单 → `safe`；其余 → `ask`。**不实现前缀匹配**。

### DLP（`src/security/dlp.ts`）

`DLP_RULES` block（authorization-header / bearer-token / private-key / github-pat / github-fine-pat / aws-access-key / api-key / password / passwd / token / secret-value）+ redact（env-file / ssh-key / pem-file / credentials-file）。`scanDlp(input, ctx)` → `DlpResult{allowed, redacted, findings}`；`MASK_SECRET=[REDACTED]`、`MASK_PATH=[REDACTED-PATH]`。

### Path Policy（`src/security/path-policy.ts`）

`resolveProjectPath`、`assertWithinRoot`、`assertWithinAllowedRoots`、`toRelativePath`；`WINDOWS_DRIVE=/^[a-zA-Z]:[\\/]/`、`WINDOWS_UNC=/^\\\\/`；`normalizeSeparators`（`\`→`/`）；`isWithin`；`realpathExisting`（deepest existing ancestor realpath + 重拼未存在尾部）。symlink escape 通过 realpath 检测。

### Permissions（`src/permissions.ts` + `config/opencode/permissions.json`）

`validatePermissions(config)` → `ValidationIssue[]`；`ENTERPRISE_AGENTS` 三者的 write/edit/bash 必须 deny；`general` 三者不得 deny；受控 tool 集合。`permissions.json`：general=ask；企业 Agent 只 allow 各自受控 tool（collect_review_context / analyze_test_project / write_test_file / run_project_test / extract_api_spec / validate_api_example / write_document / dify-query），write/edit/bash=deny。

### Dify 熔断器（`src/dify-query.ts`）

`CircuitBreaker`（threshold=3、resetTimeoutMs=60000、可注入 `now()`）、`CircuitOpenError`、`DifyHttpError`、`DifyInvalidResponseError`；`DifyClient.query()` 熔断/错误 → `degraded`；`fetchQuery` 用 AbortController 超时；`classifyDifyError` → timeout/http-4xx/http-5xx/invalid-response/network-error；`difyConfigFromEnv`（DIFY_BASE_URL/DIFY_API_KEY）。

### Audit Log（`src/audit-log.ts`）

`AuditEntry{timestamp, sessionId, agent, tool, action, result, duration, relativePath?, errorCategory?}`；`AuditLogger.log()` 返回 `{ok, error?}` 永不 throw；`sanitize()` = DLP redact + 绝对路径→项目相对路径。

### Runtime Security Guard（`src/runtime-security-guard.ts`）

`before(input)`：写 action + targetPath → path policy → command policy（若有 command）→ DLP input → `{decision, reason, redactedInput}`；`after(input)`：DLP output + audit。`WRITE_ACTIONS={write,create,overwrite,append,edit}`。

### 入口（`src/index.ts`）

re-export security/types、command-policy、dlp、path-policy、dify-query、audit-log、runtime-security-guard、permissions（tools 导出属 Task 13，暂未加入）。

## Full Gate Verification

| Gate | Result |
|------|--------|
| `bun test`（distribution/plugins） | PASS（94 tests，无失败） |
| `bun run build`（bundle） | PASS（dist/index.js 18.11KB，611 行） |
| `./scripts/check-plugin-bundle.sh` | PASS（bundle 自包含，无非 builtin import） |
| `./scripts/run-plugin-smoke.sh` | PASS（guard + audit + dify degradation，零公网） |
| `GOTOOLCHAIN=local go build ./...` | PASS |
| `GOTOOLCHAIN=local go test ./... -count=1` | PASS（22 packages，无失败） |
| `GOTOOLCHAIN=local go test -race ./... -count=1` | PASS（22 packages，无竞态） |
| `GOTOOLCHAIN=local go vet ./...` | clean |
| `GOOS=windows GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `GOOS=darwin GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `./scripts/check-runtime-boundary.sh` | PASS（no vendor DTO leakage） |
| `./scripts/check-execution-state.sh` | valid（Task 12 Step 1 in_progress） |
| `tests/execution-state/state_validator_test.sh` | PASS |

## Task 12 Gate Checklist

- [x] Plugin 工程骨架：TS + Bun 1.3.14 + bun.lock + 自包含 ESM，零 OpenCode Core patch
- [x] Command Policy 风险分级（safe/ask/deny），无「前缀安全=整体安全」
- [x] DLP 4 层（secret block / sensitive path redact / content redact / output minimize）
- [x] Path Policy canonicalize + realpath，覆盖 traversal/absolute/Windows drive/UNC/symlink/case-separator
- [x] Permissions 分离 General 与 3 企业 Agent（企业 write/edit/bash deny，仅受控 tool）
- [x] Dify 熔断器（3 连败 → open 60s → degraded → half-open → closed）
- [x] 审计日志不记录 source/prompt/output/token/绝对路径/password/Authorization
- [x] Runtime Security Guard before/after hook
- [x] bundle 自包含 + 离线安全（无非 builtin import，零公网 smoke）
- [x] 全量 Gate 通过；Windows/darwin 交叉编译不退化

## Files Changed

| File | Action |
|------|--------|
| `distribution/plugins/package.json` | Create（bun@1.3.14、scripts、@types/bun） |
| `distribution/plugins/tsconfig.json` | Create（ESNext、moduleResolution bundler、strict） |
| `distribution/plugins/bun.lock` | Create（@types/bun 锁文件） |
| `distribution/plugins/src/security/types.ts` | Create（CommandRisk/DlpContext/DlpResult 等） |
| `distribution/plugins/src/security/command-policy.ts` | Create |
| `distribution/plugins/src/security/dlp.ts` | Create |
| `distribution/plugins/src/security/path-policy.ts` | Create |
| `distribution/plugins/src/dify-query.ts` | Create（熔断器 + DifyClient） |
| `distribution/plugins/src/audit-log.ts` | Create |
| `distribution/plugins/src/runtime-security-guard.ts` | Create |
| `distribution/plugins/src/permissions.ts` | Create（validatePermissions） |
| `distribution/plugins/src/index.ts` | Create（re-export 安全模块） |
| `distribution/plugins/tests/*.test.ts` | Create（command-policy/dlp/path-policy/dify-query/audit-log/runtime-security-guard/permissions，94 tests） |
| `distribution/plugins/tests/bundle-smoke.ts` | Create（真实 bundle 执行 smoke） |
| `distribution/config/opencode/permissions.json` | Create（General vs 3 企业 Agent 权限分离） |
| `scripts/check-plugin-bundle.sh` | Create（bundle 自包含校验） |
| `scripts/run-plugin-smoke.sh` | Create（build + smoke） |
| `.gitignore` | Modify（加 node_modules/） |

## Batch Implementation Exception（Task 12/13）

Task 12/13 按连续 Batch 执行，不修改 execution-state validator（保持「单 active Task + completed 必须 humanAccepted=true」）。

- Task 12 本报告撰写时正式状态仍为 `in_progress`（不提前进入 awaiting_acceptance）。
- Task 12 代码 + Gate 已全绿并提交（`ef82075`），允许继续开发 Task 13 代码，但不提前改变 Task 13 正式状态（仍 `pending`）。
- Task 12+13 代码和证据全部完成后，Task 12 才进入 `awaiting_acceptance`；两个 Task 一起提交人工 Review；Review 通过后按 validator 顺序纯状态收口：Task 12 completed → Task 13 awaiting_acceptance → Task 13 completed。

## Gate 结论

- verification：pass（Task 12 独立 Gate 全绿）
- Task Gate：pass
- 状态：`awaiting_acceptance`（Batch Exception，随 Task 13 一起提交 Batch Human Review）

## 验收整改（Batch 12-13 Remediation）

Checkpoint：`9cbd5e64b04b82e494119a08d89376f202f8327f`（整改后全量 Gate 复跑证据）

人工验收指出的安全整改，本 Task 侧（Command Policy / Runtime Security Guard）修复：

1. **Safe Command 不得跳过 DLP/path**：`runtime-security-guard.ts` 的 `before()` 去掉「RiskSafe 早退」，safe command 现在仍进入 DLP input 扫描（command 字符串 + tool input 合并扫描），携带 secret 的 `grep password=...` 会命中 `dlp-blocked`。
2. **参数级安全控制**：`command-policy.ts` 新增 `findDangerousGitOption()`（`-c/--config/--config-env/-C/--directory/--git-dir/--work-tree/--output/--upload-pack/--receive-pack/--pager/--exec-path` → deny，阻止只读 git 升级为代码执行/目录逃逸/写文件）与 `findSensitivePath()`（绝对路径/`~`/Windows drive/`..` traversal/`.env`/`.ssh`/`.aws`/`.gnupg`/ssh-key/credentials 文件 → deny）。`git -c core.pager=sh log` 与 `cat .env` 不再读作 safe。
3. **负向测试**：新增 12 个 CommandPolicy 用例 + 2 个 Guard 用例（safe command 带敏感路径 → deny、safe command 带 secret → DLP-block）。

整改后 Gate 复跑（Task 12 侧）：`bun test` 178 pass（原 155）、`bun run build` 61.84 KB、`./scripts/check-plugin-bundle.sh` PASS、`./scripts/run-plugin-smoke.sh` PASS、`GOTOOLCHAIN=local go test ./... -count=1` 22 packages PASS、`-race` PASS、`go vet` clean、`go build` PASS、Windows/darwin 交叉编译 PASS、`./scripts/check-runtime-boundary.sh` PASS、`OPENCODE_BIN=<abs> ./scripts/run-real-parity-smoke.sh` 17/17 PASS（v1.18.11）。

## 验收整改（Round 2 — OpenCode Plugin Adapter + 输出 DLP 生效 + 动态 shell 降级）

人工二次验收指出的 Blocking 项，本 Task 侧修复：

1. **OpenCode v1.18.11 Plugin Adapter/Entry**：新增 `src/opencode/types.ts`（OpenCode Plugin SDK 契约的 type-only 镜像）与 `src/opencode/entry.ts`（真正 default-export `{id: "codea-enterprise", server}`）。`server(input)` 返回 `Hooks.tool`：注册 7 个企业 Custom Tool（Task 13）+ `dify-query`，把 OpenCode `ToolContext`（`sessionID/agent/directory/ask`）映射为 Codea `ToolContext`，并挂 Guard：`guard.before` deny → throw 中止；write/execute action → `ctx.ask` 进入 permission 流程；`guard.guardOutput` 作用于返回给模型的输出。`src/index.ts` 增加 `export { plugin, plugin as default }`。args 使用真实 zod schema（与 OpenCode 锁定的 zod 4.1.8 一致），保证 `fromPlugin` 走 `zodJsonSchema` 得到正确的 required/optional。
2. **输出 DLP 真正作用于返回模型的数据**：`runtime-security-guard.ts` 新增 `guardOutput(output)`，返回 `{output, blocked, rule}`；adapter 在每个 tool 执行后对序列化结果调用 `guardOutput`，layer-1 secret 整体 block、普通敏感值原地 redact，并以 `dlpBlocked/dlpRule` metadata 透传。
3. **safe command 动态 shell 表达式降级 ask**：`command-policy.ts` 新增 `hasDynamicExpansion()`（`* ? [ ]` glob、`$VAR`/`${VAR}` 变量展开），白名单命中后仍检测到动态展开则降级 `ask`，避免只读命令经 glob/变量升级为任意执行。

新增 Real OpenCode Plugin Smoke：`tests/plugin-smoke.ts` 加载 bundle 的 default export（`readV1Plugin` 契约）→ 调用 `server()` 得到 `Hooks.tool`（`fromPlugin` 契约）→ 驱动 8 个 tool 走完整 Guard 链（path deny / DLP input deny / write permission ask / output DLP block / dify degraded，零公网）。`scripts/run-real-plugin-smoke.sh` 额外用真实 `opencode serve`（v1.18.11）注册插件并断言 health。

Round 2 Gate 复跑（Task 12 侧）：`bun test` 201 pass、`bun run build` 0.52 MB（zod 内联）、`./scripts/check-plugin-bundle.sh` PASS、`./scripts/run-plugin-smoke.sh` PASS、`OPENCODE_BIN=<abs> ./scripts/run-real-plugin-smoke.sh` PASS、`OPENCODE_BIN=<abs> ./scripts/run-real-parity-smoke.sh` 17/17 PASS（v1.18.11）、`GOTOOLCHAIN=local go test ./... -count=1` 22 packages PASS、`-race` PASS、`go vet` clean、`go build` PASS、Windows/darwin 交叉编译 PASS、`./scripts/check-runtime-boundary.sh` PASS、`./scripts/check-execution-state.sh` valid。

## 验收整改（Round 3 — 原生 tool 输出 DLP + 插件运行时注册）

人工第二轮「有条件通过」指出的 Blocking 项，本 Task 侧修复：

1. **原生 Tool 输出 DLP**：此前 `guardOutput` 只包裹 8 个注册 custom tool，未覆盖 OpenCode 原生 `read/grep/glob/bash`。`src/opencode/entry.ts` 新增 `tool.execute.after` 钩子（`src/opencode/types.ts` 补 `tool.execute.after` 契约签名）：对 `NATIVE_OUTPUT_DLP_TOOLS = {read,grep,glob,bash}` 的输出跑 `guardOutput`，layer-1 secret 整体 block、普通敏感值原地 redact，并以 `dlpBlocked/dlpRule` metadata 透传。同时 `tool.execute.before` 对原生 `read/grep/glob` 的 `filePath`/`path` 跑 `findSensitivePath`（`command-policy.ts` 改为 export），绝对路径/traversal/`.env`/`.ssh` 等在文件读取前 deny。
2. **Runtime 插件注册**：此前 bootstrap 从未把插件 bundle 写进 OpenCode 配置，插件形同未加载。`tui/cmd/codea/main.go` 新增 `pluginBundlePath()`（`CODEA_PLUGIN_BUNDLE` 可覆盖，默认取 `../distribution/plugins/dist/index.js`）与 `writePluginConfig(cfgDir)`：在 skill sync 之后、`bootstrapRuntime` 之前，把 Codea 自有的 `opencode.json`（`"plugin": ["file://…/dist/index.js"]`）写入受控 config dir。bundle 缺失（未构建/覆盖路径失效）时降级 General 模式，绝不注册死插件 URL。`main_test.go` 新增 `TestWritePluginConfigRegistersBundle` / `TestWritePluginConfigMissingBundleNoop`。

Round 3 Gate 复跑（Task 12 侧）：`bun test` 213 pass、`bun run build` 0.52 MB、`./scripts/check-plugin-bundle.sh` PASS、`./scripts/run-plugin-smoke.sh` PASS、`OPENCODE_BIN=<abs> ./scripts/run-real-plugin-smoke.sh` PASS（`/experimental/tool/ids` 断言 8/8 企业 tool 注册）、`OPENCODE_BIN=<abs> ./scripts/run-real-parity-smoke.sh` 17/17 PASS（v1.18.11）、`GOTOOLCHAIN=local go test ./... -count=1` 22 packages PASS、`-race` PASS、`go vet` clean、`go build` PASS、Windows/darwin 交叉编译 PASS、`./scripts/check-runtime-boundary.sh` PASS、`./scripts/check-execution-state.sh` valid、`tests/execution-state/state_validator_test.sh` PASS。

## 验收整改（Round 4 — 原生 read/grep/glob 绝对路径误判修复）

人工第三轮「有条件通过」指出的唯一代码 Blocking，本 Task 侧（Path Policy + 原生 tool 适配）修复：

1. **原生 `read/grep/glob` 绝对路径误判**：此前 `tool.execute.before` 对原生 `read/grep/glob` 的 `filePath`/`path` 直接跑 `findSensitivePath(targetPath)`，任何绝对路径（`/...`、`C:\...`）都被判为 `absolute-path` → deny。但 OpenCode v1.18.11 原生 `read` 的 `filePath` 契约定义为「文件/目录的绝对路径」，项目内绝对路径（如 `/project/src/main/java/Foo.java`）应正常放行，否则企业 Agent 连项目内文件都读不了。`src/security/path-policy.ts` 新增 `validateNativeReadPath(projectRoot, targetPath)`：原始 path → 分隔符归一 → 绝对/相对解析（Windows drive/UNC 走 `path.win32`，大小写折叠）→ 是否仍在 projectRoot 内（否 → `outside-project`）→ 敏感目标检查（`.env`/`.ssh`/`.aws`/`.gnupg`/credentials/ssh-key/pem → `sensitive-file:*`/`sensitive-dir`）→ POSIX symlink escape 检测（`symlink-escape`/`unresolvable`）。`entry.ts` 原生路径钩子改用 `validateNativeReadPath(input.directory, targetPath)`，错误码由 `sensitive-path:` 改为 `native-path:`。`findSensitivePath` 收窄为 `command-policy.ts` 私有（仅 shell 命令分析内部使用），不再对外导出。

2. **测试**：`path-policy.test.ts` 新增 `validateNativeReadPath` 单元测试（绝对路径在项目内 → allow、项目外 → deny、相对路径、空路径、`.env`/`.env.production`/`.ssh`/credentials/id_rsa 敏感目标、symlink escape、Windows drive/UNC/前向斜杠/不同盘符/大小写不敏感 containment）；`opencode-entry.test.ts` 新增「绝对路径在项目内 → allow、项目外 → deny、Windows 覆盖」集成断言；`tests/plugin-smoke.ts` 补绝对路径在项目内 allow + 项目外 deny 断言。

Round 4 Gate 复跑（Task 12 侧）：`bun test` 234 pass、`bun run build` 0.52 MB、`./scripts/check-plugin-bundle.sh` PASS、`./scripts/run-plugin-smoke.sh` PASS、`OPENCODE_BIN=<abs> ./scripts/run-real-plugin-smoke.sh` PASS（`/experimental/tool/ids` 断言 8/8 企业 tool 注册）、`OPENCODE_BIN=<abs> ./scripts/run-real-parity-smoke.sh` 17/17 PASS（v1.18.11）、`GOTOOLCHAIN=local go test ./... -count=1` 22 packages PASS、`-race` PASS、`go vet` clean、`go build` PASS、Windows/darwin 交叉编译 PASS、`./scripts/check-runtime-boundary.sh` PASS、`./scripts/check-execution-state.sh` valid、`tests/execution-state/state_validator_test.sh` PASS。

## 验收整改（Round 5 — Windows 相对路径 + symlink/junction realpath + 敏感文件名大小写）

人工第四轮「有条件通过」指出的 1 个 Windows path-policy Blocking，本 Task 侧（Path Policy）修复：

1. **Windows 路径风格不再只看 targetPath**：此前 `validateNativeReadPath` 用 `WINDOWS_DRIVE.test(targetPath) || WINDOWS_UNC.test(targetPath)` 决定 Windows 模式，导致 `root=C:\code\project` + `targetPath=src/main/java`（OpenCode `grep`/`glob` 合法允许的相对 `path`）落入 `path.posix` 分支。现改为 `windowsStyle = process.platform === "win32" || isWindowsPath(root) || isWindowsPath(targetPath)`；Windows root + 相对 target → `path.win32.resolve(root, target)`。新增 `isWindowsPath()` 复用 `WINDOWS_DRIVE`/`WINDOWS_UNC`。

2. **路径风格与「当前主机是否可 realpath」分离**：此前只对 `!windows` 做 `realpathSync`，真实 Windows 主机上 Windows 绝对路径仅有 lexical containment，项目内 symlink/junction 指向外部不会被真实路径检查拦截。现改为 `canRealpath = (process.platform === "win32") === windowsStyle`：POSIX 主机 + POSIX path、Windows 主机 + Windows path 都执行 `fs.realpath` 的 symlink/junction escape 检测；POSIX 主机上模拟的 `C:\...`（单测）仅做 lexical containment（主机文件系统无法解析，跳过 realpath）。realpath 结果比较同样做大小写折叠。

3. **敏感文件名大小写不敏感**：`sensitiveSegment` 的 `base` 统一 `toLowerCase()`，`credentials`/`.git-credentials`/`.npmrc`/`.netrc`/`id_rsa`/`id_ed25519`/`id_ecdsa`/`id_dsa` 命中 `Credentials`/`ID_RSA` 等（Windows 文件系统通常大小写不敏感）。

4. **测试**：`path-policy.test.ts` 新增 Windows root + 相对路径在项目内 → allow、Windows root + 相对 traversal → deny（`..\..` 与 `../..` 两种分隔符）、Windows 大小写不敏感敏感文件名 → deny。真实 Windows 主机 junction/symlink escape（需 `process.platform === "win32"` 下对 `C:\...` 跑 `fs.realpath`）无法在 POSIX 测试主机上执行，该分支代码已就位（与 POSIX symlink escape 同构），真实 Windows junction 测试按 Windows wrapper 同样延后至发行验收（Task 17/18 / Task 21）。

Round 5 Gate 复跑（Task 12 侧）：`bun test` 239 pass、`bun run build` 0.52 MB、`./scripts/check-plugin-bundle.sh` PASS、`./scripts/run-plugin-smoke.sh` PASS、`OPENCODE_BIN=<abs> ./scripts/run-real-plugin-smoke.sh` PASS（`/experimental/tool/ids` 断言 8/8 企业 tool 注册）、`OPENCODE_BIN=<abs> ./scripts/run-real-parity-smoke.sh` 17/17 PASS（v1.18.11）、`GOTOOLCHAIN=local go test ./... -count=1` 22 packages PASS、`-race` PASS、`go vet` clean、`go build` PASS、Windows/darwin 交叉编译 PASS、`./scripts/check-runtime-boundary.sh` PASS、`./scripts/check-execution-state.sh` valid、`tests/execution-state/state_validator_test.sh` PASS。
