# Task 32 — Model Qualification & Adaptive Agent Strategy Verification Report

## Status

- Task: **32 — Model Qualification & Adaptive Agent Strategy**
- Production baseline: `cbf008b8f239a3273062c059c43bdfa2b4cbf713`
- Fresh Task 32 Gate: **GitHub Actions `34005073941` — PASS**
- Linux: **PASS**
- Windows: **PASS**
- macOS: **PASS**
- Automated verification: **PASS**
- Task Gate: **PASS**
- Human acceptance: **PENDING**
- Task 33: **not started**

## Delivered behavior

1. **Local model capability profile**
   - Profiles are keyed by provider + model and stored under `CODEA_HOME/model-profiles/<sha256>.json`.
   - The persisted profile contains bounded capability metadata only; it does not store user prompts, repository context, provider output, credentials, endpoints, or chain-of-thought.
   - Writes are atomic; malformed profile state fails visibly; stale profile versions are treated as unqualified.
   - Missing, stale, invalid, or aborted qualification falls back to the conservative `medium` strategy.

2. **Bounded qualification probes**
   - Qualification uses exactly four non-project-mutating probes: tool call, structured output, patch following, and planning.
   - Machine score is derived from observed probe completion/attempt order rather than model self-report or free-form reasoning text.
   - All first-attempt passes produce `strong`; a second-attempt recovery without any twice-failed probe produces `medium`; any probe that fails twice produces `weak`.
   - Interrupted qualification does not persist a completed profile.

3. **Isolated evaluator and `/model-check`**
   - `model-evaluator` is a utility Agent with only the four probe tools allowed; read/grep/glob/write/edit/bash/project verification/Dify tools remain denied.
   - `/model-check` is a protected local action and uses an internal qualification session instead of the user's visible chat session.
   - Internal qualification sessions are filtered from normal session/chat presentation.
   - Explicit selected model wins; when no explicit selection exists, qualification can use a unique Runtime default. Ambiguous model identity does not fabricate a profile.

4. **Adaptive strategy**
   - `strong`: Repo Map max 12,000 chars; planning remains machine-valid at 3–7 steps; verification continuation limit 2.
   - `medium`: Repo Map max 8,000 chars; guidance prefers 3–5 steps; verification continuation limit 2.
   - `weak`: Repo Map max 4,000 chars; guidance prefers 3 concise steps; verification continuation limit 1; sequential/single-mutation-group guidance is enabled.
   - The verification continuation budget is frozen when the root task starts, so changing model mid-task cannot silently change the root task's verification budget.

5. **Safety and truth invariance**
   - Model strength never grants tools, bypasses approval, weakens command/path/DLP policy, or enables network access.
   - Mechanical tests prove the same denied command, network command, path escape, DLP secret write, and approval-required write have identical decisions under weak/medium/strong profiles.
   - The same engineering prompt keeps the same Agent route across profiles.
   - Task 30 verification truth remains authoritative and unchanged by model profile.
   - Task 31 checkpoint package regression remains green; adaptive strategy does not weaken checkpoint/restore protection.

## Task 32 mechanical acceptance

Fresh dedicated Task 32 Gate: **GitHub Actions `34005073941`** on exact production baseline `cbf008b8f239a3273062c059c43bdfa2b4cbf713`.

Permanent acceptance emits:

```text
TASK32_MODEL_QUALIFICATION PASS
STRONG_PROFILE PASS
MEDIUM_PROFILE PASS
WEAK_PROFILE PASS
MODEL_PROFILE_ISOLATION PASS
STRATEGY_SECURITY_INVARIANT PASS
V12_AGENT_RELIABILITY_REGRESSION PASS
```

### Linux — Ubuntu 24.04

- V1.2 execution-state validation: **PASS**
- Task 32 mechanical acceptance: **PASS**
- Task 28 Repo Intelligence regression: **PASS**
- Task 29 Agent Planning regression: **PASS**
- Task 30 Verification Loop regression: **PASS**
- Task 31 Checkpoint regression: **PASS**
- Model strategy security invariance: **PASS**
- Full enterprise plugin regression + build: **PASS**
- Full `GOTOOLCHAIN=local go test ./... -count=1`: **PASS**
- `GOTOOLCHAIN=local go build ./cmd/codea`: **PASS**
- `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea`: **PASS**

### Windows — Windows Server 2025

- PowerShell native fail-fast self-test: **PASS**
- Native probe/evaluator/security tests: **PASS**
- Native model profile + adaptive-strategy tests: **PASS**
- `CODEA_HOME` with spaces and non-ASCII (`Codea Home 中文 Model Profiles`): **PASS**
- Native Task 31 checkpoint package regression: **PASS**
- Full enterprise plugin regression + build: **PASS**
- Full Go regression with `CGO_ENABLED=0`: **PASS**
- Native `go build ./cmd/codea`: **PASS**

### macOS — macOS 15

- Native Task 32 mechanical acceptance: **PASS**
- Full enterprise plugin regression + build: **PASS**
- Full Go regression: **PASS**
- `go build ./cmd/codea`: **PASS**

## Private intranet acceptance boundary

Task 32 itself introduces no external benchmark, telemetry upload, or public API dependency. Public CI verifies local qualification, profile storage, adaptive strategy, security invariance, Windows no-CGO compatibility, and three-platform regressions.

A real company-intranet run against the private deployed model is environment-specific evidence and **cannot be certified by public GitHub Actions**. No private endpoint, credential, provider payload, or fabricated intranet result is recorded in this report. That evidence remains a separate manual acceptance item when the private environment is available.

## Gate conclusion

Task 32 automated implementation and certification are complete at production baseline `cbf008b8f239a3273062c059c43bdfa2b4cbf713`. Dedicated Gate `34005073941` passes on Linux, Windows, and macOS, including full Go/plugin regressions, native Windows spaces/non-ASCII profile paths, and the Windows no-CGO build contract.

The adaptive strategy changes context/planning/retry support only. Agent routing, security permission decisions, verification truth, and checkpoint protection remain authoritative and invariant.

Task 32 is ready for **human acceptance** and must remain `awaiting_acceptance` with `humanAccepted: false` until explicitly approved. Task 33 has not started.
