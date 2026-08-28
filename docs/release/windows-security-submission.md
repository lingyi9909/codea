# Windows Security / 360 False-Positive Submission Guide

Task 27 只负责生成可复核的可信发布证据；360 软件开放平台、企业白名单或其他 EDR 厂商的实际提交、受理和解除误报属于 **external evidence**，不得在 CI、Task report 或 execution-state 中伪造为已完成。

## 360 官方通道

2026-08-28 核对的 360 软件开放平台官方流程：

- 软件开发者可在 360 软件开放平台提交软件检测；
- 发生误报时可使用开发者误报反馈页面提交 Windows 文件与误报截图；
- 处理结果通过填写的邮箱返回；
- 软件版本更新后建议重新上传检测；
- 若通过检测后仍拦截，应对比被报文件与平台提交文件是否为同一哈希，并提供误报弹窗截图、具体被报文件和软件 ID；
- 官方联系邮箱为 `opensoft@360.cn`。

外部平台字段、页面和处理规则可能变化，正式提交前必须再次以 360 软件开放平台当前页面为准。

## 每个 Windows Release 必须保留的 evidence

由 `packaging/platform/windows/build-security-evidence.ps1` 生成的 JSON 至少包含：

- `codeaVersion`
- `releaseFile`
- `releaseSize`
- `releaseSha256`
- `gitCommit`
- `releaseTag`
- `openCodeVersion`
- `openCodeChecksum`
- `signatureStatus`
- `signerSubject`
- `signerThumbprint`

这些字段用于证明“被报文件”和“我们实际发布文件”是同一构建产物，并将 Codea Publisher 与内置 OpenCode Runtime 的供应商二进制区分开。

## 提交材料清单

每次出现 360/EDR 误报时，至少准备：

1. 最终发布 ZIP（不要重新压缩或修改后再提交）；
2. release ZIP SHA256，即 evidence 中的 `releaseSha256`；
3. 被报 `codea.exe` 文件；
4. Authenticode `signerSubject` / `signerThumbprint` / `signatureStatus`；
5. Git release tag 与 commit；
6. Codea 版本；
7. bundled OpenCode v1.18.11 版本与官方锁定 checksum；
8. 360 拦截/报毒完整截图，包含具体文件路径与检测名称；
9. 复现步骤：安装包来源、安装路径、fresh/upgrade/rollback 情况；
10. Task 27 Windows installed-package lifecycle Gate 的 run ID / 结果，用于证明正式包能正常安装并启动 Runtime。

## 发布纪律

- Preview 包若没有受信任的生产 Authenticode 证书，必须明确标记为 unsigned preview；不得伪造 Publisher reputation。
- Stable Windows release 必须配置真实 Code Signing PFX，并通过 fail-closed Authenticode 验签。
- CI 的 ephemeral self-signed certificate 仅证明签名工具链机械可用，不代表公众信任或 360 白名单。
- 不得在仓库、artifact、日志或 evidence JSON 中保存 PFX 密码、私钥或其他 secret。
- 360 实际“已受理 / 已解除 / 已加入白名单”只能由外部平台回执证明，Task 27 自动 Gate 不得自行写 PASS。
