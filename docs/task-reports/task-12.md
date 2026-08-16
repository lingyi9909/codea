# Task 12 Report — 安全规则、DLP、Dify、审计 + Plugin 工程骨架

## Overview

Checkpoint: `ef8207576f9c3aaf328d8436e01f6a619c58559d`

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
- 状态：`in_progress`（Batch Exception，暂不进入 awaiting_acceptance，继续 Task 13 开发）
