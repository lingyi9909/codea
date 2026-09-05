# Task 31 — Agent Checkpoint & Restore Verification Report

## Status

- Task: **31 — Agent Checkpoint & Restore**
- Production baseline: `75fc1feef9fd5e0363eaec5cb63fc6d2a3d3869d`
- Fresh Task 31 Gate: **GitHub Actions `33974093998` — PASS**
- Fresh Task 20-21 real dual-runtime parity: **GitHub Actions `33974093996` — PASS**
- Fresh Task 26 V1.1 integration / G3-G14 parity: **GitHub Actions `33974093989` — PASS**
- Linux: **PASS**
- Windows: **PASS**
- macOS: **PASS**
- Automated verification: **PASS**
- Task Gate: **PASS**
- Human acceptance: **PENDING**
- Task 32: **not started**

## Delivered behavior

1. **Codea-owned shadow Git checkpoints**
   - Checkpoints live under `CODEA_HOME/checkpoints/<workspace-hash>/git`, outside the project repository.
   - Every checkpoint Git operation uses an explicit shadow `--git-dir` and project `--work-tree`; project `.git` is not used as the checkpoint index/ref store.
   - Project branch, HEAD, index and refs are mechanically verified unchanged by checkpoint create/restore operations.
   - Plain non-Git project folders are supported when the local Git executable is available.

2. **Deterministic local snapshots**
   - Eligible modify/add/delete/rename state is captured in the shadow tree.
   - Candidate path staging uses NUL-separated pathspec stdin rather than a shell command or giant argv.
   - Identical trees can reuse the previous shadow commit while still recording a distinct checkpoint event.
   - Metadata is persisted atomically and corrupt metadata fails closed instead of silently resetting checkpoint history.

3. **Safe restore and recovery**
   - User/model checkpoint IDs must resolve through trusted Codea metadata; free-form Git revisions are not accepted.
   - A `safety` checkpoint is mandatory immediately before applying a restore target.
   - Restore reproduces exact eligible bytes for added/modified/deleted paths and the safety checkpoint can restore the pre-restore state.
   - Interrupted restore persists safe recovery IDs in `restore-state.json` and surfaces recovery guidance on reopen.
   - Restore never uses `git reset --hard` against the user's project repository.
   - Before any changed path is written or deleted, restore inspects its existing ancestor chain and fails closed on symlink ancestors; Windows additionally rejects reparse-point ancestors such as junctions.
   - A safety-snapshot skipped directory protects both the directory and every descendant path (`dir/**`) from restore mutation.

4. **Agent lifecycle integration**
   - An Agent engineering prompt schedules/reuses an asynchronous baseline checkpoint before Runtime prompt execution.
   - A fresh mutating verification PASS schedules a final checkpoint.
   - Checkpoint failure is a separate truth dimension: it is shown to the user but does not rewrite a valid Task 30 verification PASS into failure.
   - Git/checkpoint unavailability does not block ordinary Codea startup or fabricate checkpoint protection.

5. **Local checkpoint commands**
   - `/checkpoint`, `/checkpoints`, and `/restore <checkpoint-id>` are protected built-ins.
   - They are local workspace actions and never route through the model as prompts.
   - Restore is blocked while streaming, while approval is pending, or while another checkpoint operation is in flight.
   - Every completed restore attempt invalidates Repo Context, including an interrupted restore that has already partially mutated disk state, so subsequent Agent prompts cannot reuse a stale pre-restore Repo Map.

6. **Retention and degradation**
   - Default history keeps the newest 20 checkpoint records while preserving protected active baseline/safety or recovery references.
   - Git-unavailable behavior returns `CHECKPOINT_UNAVAILABLE` and does not write false checkpoint metadata.

## Snapshot exclusions and limits

Task 31 checkpoints are **local shadow snapshots, not project Git commits, branches, stash entries, or refs**.

The deterministic snapshot policy excludes generated/sensitive paths including at least:

```text
.git/
target/
build/
dist/
node_modules/
.codea/
.env
.env.*
*.pem
*.key
*.p12
*.pfx
credentials*
secrets*
```

Files above the default 5 MiB size limit and configured large/binary thresholds are skipped with evidence. A skipped sensitive/generated/large file is not represented as protected by the checkpoint. Existing Codea DLP/path policy remains the primary security boundary.

The checkpoint subsystem uses the local Git CLI only; its checkpoint control path does not require public network access.

## Final regression repairs discovered during Task 31 certification

Task 31 itself was already functionally complete, but fresh exact-head full-repository certification exposed forward-compatibility regressions introduced by Task 29's plan-before-mutation contract. They were repaired without weakening production safety gates:

1. **Plan-gated enterprise agents** — `unit-test-generator`, `api-documentation`, and `debug` expose `task_plan`, `task_step`, and `task_status` before their existing mutation/execute capabilities. Production write/bash/edit policies were not relaxed.
2. **API Documentation parity fixture** — real-runtime document writes follow `task_plan -> write_document`; Task 16 real API Documentation workflow smoke is green.
3. **Release parity Approval/Reject fixture** — the candidate creates a valid Task 29 plan before the approval-gated `bash`, while the vanilla baseline preserves its original direct-bash behavior.
4. **Parity collector continuation** — an intermediate planning `step.finished` can no longer truncate the later `approval.resolved` / final tool result. Replying to an approval opens a fresh continuation and clears only the stale pre-approval terminal wait state.

Fresh Task 20-21 run `33974093996` confirms the real distinct baseline/candidate parity remains green after the final Task 31 P0 repairs; full Go and plugin regression also pass.

## P0 re-verification — restore fail-safe boundaries

The final review found two restore fail-safe gaps, both repaired narrowly without changing previously accepted checkpoint behavior.

1. **Symlink / junction ancestor escape**
   - Restore now performs a fail-closed ancestor-chain preflight before any changed path is applied and repeats the check immediately before each write/delete.
   - On Unix-like systems, an existing symlink ancestor blocks restore.
   - On Windows, a symlink or `FILE_ATTRIBUTE_REPARSE_POINT` ancestor blocks restore, covering junctions.
   - The target file itself may be absent; the check walks existing ancestors rather than relying on `EvalSymlinks` of the final path.
   - Safety `Skipped` protection is subtree-aware, so a skipped `src` protects `src/**`.
   - Permanent tests prove outside sentinels remain unchanged and outside files are neither created nor deleted.

2. **Interrupted restore Repo Context invalidation**
   - Service regression injects failure on the second blob after the first file was restored, proves `CodeRestoreInterrupted`, `FilesChanged == 1`, restored file A, and unchanged file B.
   - Application handling invalidates Repo Context for every completed restore attempt before branching on success/error, so an interrupted partially-applied restore cannot leave a stale Task 28 Repo Map.

Permanent acceptance markers added for these blockers:

```text
RESTORE_SYMLINK_ESCAPE_BLOCKED PASS
RESTORE_JUNCTION_ESCAPE_BLOCKED PASS
INTERRUPTED_RESTORE_INVALIDATES_REPO_CONTEXT PASS
```

## Formal mechanical acceptance

Fresh Task 31 Gate run: **GitHub Actions `33974093998`** on exact production baseline `75fc1feef9fd5e0363eaec5cb63fc6d2a3d3869d`.

Permanent acceptance scenarios emit these markers:

```text
TASK31_AGENT_CHECKPOINT PASS
SHADOW_GIT_ISOLATION PASS
BASELINE_RESTORE PASS
SAFETY_RESTORE PASS
PROJECT_INDEX_UNCHANGED PASS
WINDOWS_SPACES_UNICODE PASS
GIT_UNAVAILABLE_FAIL_VISIBLE PASS
WINDOWS_WORKFLOW_FAIL_FAST PASS
RESTORE_SYMLINK_ESCAPE_BLOCKED PASS
RESTORE_JUNCTION_ESCAPE_BLOCKED PASS
INTERRUPTED_RESTORE_INVALIDATES_REPO_CONTEXT PASS
```

### Linux — Ubuntu 24.04

- Execution-state validation: **PASS**
- Task 31 mechanical checkpoint acceptance: **PASS**
- `RESTORE_SYMLINK_ESCAPE_BLOCKED`: **PASS**
- `INTERRUPTED_RESTORE_INVALIDATES_REPO_CONTEXT`: **PASS**
- Checkpoint package + Application + command + composition focused tests: **PASS**
- Full `GOTOOLCHAIN=local go test ./... -count=1`: **PASS**
- `GOTOOLCHAIN=local go build ./cmd/codea`: **PASS**
- `CGO_ENABLED=0 GOOS=windows GOARCH=amd64` release cross-build: **PASS**

### Windows — Windows Server 2025

- Native PowerShell fail-fast contract: **PASS**
- Native `git.exe` checkpoint lifecycle including project/home paths with spaces and non-ASCII (`中文`): **PASS**
- Native junction/reparse-point escape regression: **PASS**
- `RESTORE_JUNCTION_ESCAPE_BLOCKED`: **PASS**
- `INTERRUPTED_RESTORE_INVALIDATES_REPO_CONTEXT`: **PASS**
- Baseline restore + mandatory safety restore: **PASS**
- Project HEAD/branch/index/refs isolation: **PASS**
- Full Go regression with `CGO_ENABLED=0`: **PASS**
- `go build ./cmd/codea`: **PASS**

### macOS — macOS 15

- Native Task 31 mechanical checkpoint acceptance: **PASS**
- Symlink-ancestor restore fail-closed regression: **PASS**
- Full Go regression: **PASS**
- `go build ./cmd/codea`: **PASS**

## Full-repository release confidence

Fresh Task 26 Gate `33974093989` on production baseline `75fc1feef9fd5e0363eaec5cb63fc6d2a3d3869d` is green across Linux, Windows, and macOS. On Linux all pre-release checks, full Go race/vet/build, enterprise plugin regression/build, Debug contract, offline packaging, all supported release targets, Windows cross-build, locked OpenCode v1.18.11 download, and **V1 release parity G3 through G14** passed.

Fresh Task 20-21 Gate `33974093996` is also green and independently confirms the distinct vanilla-baseline versus Codea-candidate real-runtime parity did not regress.

## Gate conclusion

The two final Task 31 review blockers are repaired and mechanically verified at production baseline `75fc1feef9fd5e0363eaec5cb63fc6d2a3d3869d`. Task 31 Gate `33974093998` passes on Linux, Windows, and macOS; Task 20-21 real dual-runtime parity `33974093996` passes; and Task 26 full V1.1 integration / G3-G14 parity `33974093989` passes.

Restore now fails closed before following symlink/junction/reparse ancestors outside the lexical project tree, skipped directory protection covers descendants, and interrupted restore results always invalidate Repo Context. Previously accepted Task 31 capabilities were not reworked.

Task 31 remains **awaiting human acceptance**. Human acceptance is intentionally not marked true. Task 32 has not started.
