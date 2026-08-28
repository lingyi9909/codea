# Task 27 Windows Trust Chain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不降低 Windows Runtime 启动可靠性、安全边界和进程树清理能力的前提下，降低 360/EDR 误报风险并建立可签名、可验证、可申诉的 Windows 可信发布链。

**Architecture:** 保留 Installer manifest 校验、安装后 `Unblock-File`、Runtime `prepareRuntimeBinary()` MOTW self-heal 与仅针对 `ERROR_ACCESS_DENIED` 的 bounded retry。先把 Windows Job Object 的 `OpenProcess` 权限从 `PROCESS_ALL_ACCESS` 收敛为 `PROCESS_SET_QUOTA | PROCESS_TERMINATE`；新增正式 release zip 的 installed-package lifecycle smoke；新增 Authenticode signing/verification 脚本与 CI 机械验证；生成 360 白名单/误报申诉 evidence bundle。Task 27 第一版不删除 Runtime MOTW self-heal。

**Tech Stack:** Go 1.26.5, PowerShell 7, Windows 2025 GitHub runner, GitHub Actions, Authenticode/SignTool, OpenCode v1.18.11.

**Spec:** 用户在 2026-08-28 会话中批准的 Task 27 Windows 启动可靠性补充要求；基线设计继续继承 `docs/superpowers/specs/2026-08-26-codea-v1.1-agent-workspace-design.md` 的 Runtime/Windows/Offline 约束。

## Global Constraints

- Windows 启动可靠性 > 安全边界 > 降低杀软误报。
- `prepareRuntimeBinary()` 第一版保留；没有真实 Windows installed-package lifecycle 证据不得删除。
- Installer manifest SHA256/size 校验与安装后 `Unblock-File` 保留。
- `ERROR_ACCESS_DENIED` bounded retry 保留：只 retry `ERROR_ACCESS_DENIED`，次数与总等待时间有限，其他错误立即返回。
- Process Tree Cleanup/Job Object 能力不得退化。
- Runtime 固定 OpenCode v1.18.11；Windows 不依赖 WSL。
- Vendor/OpenCode DTO 不得泄漏到 Application/TUI。
- 360 企业白名单/误报申诉属于外部操作；仓库交付可复现 evidence bundle 与提交清单，不伪造外部平台已受理/已通过。

---

### Task 1: Task 27 状态与 Gate Contract

**Files:**
- Modify: `docs/execution-state.yaml`
- Create: `tests/task27_windows_trust_contract_test.py`
- Create: `.github/workflows/task27-windows-trust-gates.yml`

**Interfaces:**
- Consumes: Task 26 completed/humanAccepted=true baseline.
- Produces: Task 27 fail-closed acceptance contract and exact Windows Gate entrypoint.

- [ ] **Step 1: Write RED contract** validating Task 26 accepted, Task 27 active, MOTW self-heal/retry protections retained, `PROCESS_ALL_ACCESS` absent, Task27 workflow includes real installed-package lifecycle and Authenticode mechanical verification.
- [ ] **Step 2: Run contract and confirm RED** because Task 27 state/workflow/minimal rights are not implemented.
- [ ] **Step 3: Update execution state** to append Task 27 and set `current.task=27`, `status=in_progress`, `humanAccepted=false`.
- [ ] **Step 4: Add Task27 workflow skeleton** with Linux contract job and Windows jobs, without weakening existing Task26/Windows release gates.
- [ ] **Step 5: Re-run contract until only intended production/lifecycle gaps remain.**

### Task 2: Windows Job Object 最小进程权限

**Files:**
- Modify: `tui/internal/supervisor/process_windows.go`
- Modify: `tui/internal/supervisor/process_windows_test.go`

**Interfaces:**
- Consumes: Win32 `AssignProcessToJobObject` requirement.
- Produces: `processJobAccess = PROCESS_SET_QUOTA | PROCESS_TERMINATE` used by `OpenProcess`.

- [ ] **Step 1: Add RED test** asserting the access mask equals `0x0100 | 0x0001` and does not include unrelated VM/query/full-access bits.
- [ ] **Step 2: Run native/cross Windows focused test and confirm RED.**
- [ ] **Step 3: Replace `processAllAccess`** with explicit minimal constants `processSetQuota`, `processTerminate`, `processJobAccess`.
- [ ] **Step 4: Run focused Windows tests** including Access Denied retry, process-tree Stop, force-kill fallback.

### Task 3: Authenticode Signing & Verification Chain

**Files:**
- Create: `packaging/platform/windows/sign-release.ps1`
- Create: `packaging/platform/windows/verify-signature.ps1`
- Create: `tests/release/task27_authenticode_contract_test.py`
- Modify: `.github/workflows/task27-windows-trust-gates.yml`
- Modify: `.github/workflows/windows-release.yml`

**Interfaces:**
- Consumes: built `codea.exe`, certificate/PFX supplied by CI secret in real release.
- Produces: signed `codea.exe`, fail-closed signature verification, mechanical CI proof using ephemeral self-signed Code Signing certificate.

- [ ] **Step 1: Add RED contract** requiring SHA256 digest, Code Signing EKU, optional RFC3161 timestamp URL, verification via `Get-AuthenticodeSignature`, and no plaintext private key in repository.
- [ ] **Step 2: Confirm RED.**
- [ ] **Step 3: Implement signing script** that imports a supplied PFX into CurrentUser certificate store, selects the exact thumbprint, signs requested binaries with SHA256 and optional RFC3161 timestamp, removes imported private-key material from the store in `finally`.
- [ ] **Step 4: Implement verification script** requiring `Status=Valid`, expected signer thumbprint/subject when supplied, and failing on Unsigned/UnknownError/HashMismatch/NotTrusted for production verification mode.
- [ ] **Step 5: Add CI mechanical proof** creating an ephemeral self-signed Code Signing certificate on Windows, signing a copied test executable and verifying cryptographic signature mechanics without claiming public trust.
- [ ] **Step 6: Wire real release** so signing occurs only when signing secrets are configured; stable channel must fail closed if trusted signing credentials are absent, preview remains explicitly marked preview.

### Task 4: Windows Installed-Package Real Lifecycle Smoke

**Files:**
- Create: `tests/release/task27-windows-installed-lifecycle.ps1`
- Modify: `.github/workflows/task27-windows-trust-gates.yml`

**Interfaces:**
- Consumes: exact Windows release ZIP artifact built by `packaging/scripts/build-release.sh`, `install.ps1`, bundled `codea.exe`, bundled OpenCode v1.18.11.
- Produces: real Windows evidence for fresh/MOTW/spaces/immediate-start/upgrade/rollback/runtime health.

- [ ] **Step 1: Add RED workflow contract** requiring a real release ZIP artifact to be downloaded into Windows runner, not a synthetic fake package.
- [ ] **Step 2: Implement PowerShell lifecycle harness** with helpers to extract package, apply MOTW to package/executable, run `install.ps1`, resolve installed version/current pointer, execute installed `codea.exe doctor`, and require Runtime/OpenCode health success.
- [ ] **Step 3: Fresh install scenario**: normal path -> install -> immediate `codea.exe doctor` -> PASS.
- [ ] **Step 4: MOTW scenario**: add `Zone.Identifier` before install -> install -> assert installed `opencode.exe` no longer carries Zone.Identifier -> immediate doctor/health PASS.
- [ ] **Step 5: Spaces scenario**: `CODEA_HOME` path contains spaces -> install -> immediate doctor/health PASS.
- [ ] **Step 6: Upgrade scenario**: maintain two installed version directories/current pointers using the real packaged binaries and update platform switch semantics; switch forward then immediately run doctor/health PASS.
- [ ] **Step 7: Rollback scenario**: switch current pointer back to previous installed version through Codea update platform switch semantics, then immediate doctor/health PASS.
- [ ] **Step 8: Assert no `Access is denied`/CreateProcess/runtime-start failure** in lifecycle logs.
- [ ] **Step 9: Run existing Task26 Windows native full regression in same Windows job** so lifecycle cannot pass while core Windows tests regress.

### Task 5: 360 Whitelist / False-Positive Evidence Bundle

**Files:**
- Create: `packaging/platform/windows/build-security-evidence.ps1`
- Create: `docs/release/windows-security-submission.md`
- Modify: `.github/workflows/windows-release.yml`

**Interfaces:**
- Consumes: final release ZIP, checksum, Authenticode signature metadata, runtime version metadata.
- Produces: `codea-<version>-windows-security-evidence.json` plus human submission checklist.

- [ ] **Step 1: Add RED contract** requiring release SHA256, file size, tag/commit, Codea version, OpenCode version/checksum, signer subject/thumbprint/status, download filename, and no secrets.
- [ ] **Step 2: Implement evidence generator** with deterministic JSON ordering and fail-closed missing artifact/checksum handling.
- [ ] **Step 3: Add release workflow artifact upload** for evidence JSON.
- [ ] **Step 4: Document exact 360 enterprise allowlist / false-positive appeal fields** and explicitly mark actual submission/approval as external evidence, never automated PASS.

### Task 6: Fresh Task 27 Verification & Closeout

**Files:**
- Create: `docs/task-reports/task-27.md`
- Modify: `docs/execution-state.yaml`

**Interfaces:**
- Consumes: Tasks 1–5.
- Produces: Task 27 implementation checkpoint and automated Gate evidence.

- [ ] **Step 1: Run Task27 Windows Trust Gate** and require Windows installed-package lifecycle PASS.
- [ ] **Step 2: Require** fresh install, MOTW, spaces, immediate start, upgrade, rollback, OpenCode health, bounded Access Denied retry, process tree cleanup, Task26 Windows native regression all PASS.
- [ ] **Step 3: Run full Go regression, vet/build, architecture boundary and existing Task26 Gate regressions.**
- [ ] **Step 4: Record exact checkpoint/run IDs and signing/evidence status**; do not claim 360 external approval or publicly trusted signing unless real credentials/evidence exist.
- [ ] **Step 5: Set Task 27 to `awaiting_acceptance`, `humanAccepted=false`** after all automated gates pass.
