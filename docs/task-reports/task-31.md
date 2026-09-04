# Task 31 — Agent Checkpoint & Restore Verification Report

## Status

- Task: **31 — Agent Checkpoint & Restore**
- Production baseline: `0057614d0617248efdf40c2420f0303d1417e498`
- Fresh Task 31 Gate: **GitHub Actions `33842705894` — PASS**
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

4. **Agent lifecycle integration**
   - An Agent engineering prompt schedules/reuses an asynchronous baseline checkpoint before Runtime prompt execution.
   - A fresh mutating verification PASS schedules a final checkpoint.
   - Checkpoint failure is a separate truth dimension: it is shown to the user but does not rewrite a valid Task 30 verification PASS into failure.
   - Git/checkpoint unavailability does not block ordinary Codea startup or fabricate checkpoint protection.

5. **Local checkpoint commands**
   - `/checkpoint`, `/checkpoints`, and `/restore <checkpoint-id>` are protected built-ins.
   - They are local workspace actions and never route through the model as prompts.
   - Restore is blocked while streaming, while approval is pending, or while another checkpoint operation is in flight.
   - Successful restore invalidates Repo Context so subsequent Agent prompts cannot reuse stale pre-restore source structure.

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

## Formal mechanical acceptance

Fresh Task 31 Gate run: **GitHub Actions `33842705894`** on exact production baseline `0057614d0617248efdf40c2420f0303d1417e498`.

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
```

### Linux — Ubuntu 24.04

- Task 31 mechanical checkpoint acceptance: **PASS**
- Full `GOTOOLCHAIN=local go test ./... -count=1`: **PASS**
- `GOTOOLCHAIN=local go build ./cmd/codea`: **PASS**
- `CGO_ENABLED=0 GOOS=windows GOARCH=amd64` release cross-build: **PASS**

### Windows — Windows Server 2025

- Native PowerShell fail-fast contract: **PASS**
- Native `git.exe` checkpoint lifecycle including project/home paths with spaces and non-ASCII (`中文`): **PASS**
- Baseline restore + mandatory safety restore: **PASS**
- Project HEAD/branch/index/refs isolation: **PASS**
- Full Go regression with `CGO_ENABLED=0`: **PASS**
- `go build ./cmd/codea`: **PASS**

### macOS — macOS 15

- Native Task 31 mechanical checkpoint acceptance: **PASS**
- Full Go regression: **PASS**
- `go build ./cmd/codea`: **PASS**

## Gate conclusion

All Task 31 technical done criteria are satisfied at production baseline `0057614d0617248efdf40c2420f0303d1417e498`, with Fresh Task 31 Gate `33842705894` passing on Linux, Windows, and macOS. Shadow Git metadata is isolated from the project repository, eligible source state is reversible through mandatory safety checkpoints, restore invalidates Repo Context, checkpoint unavailability is fail-visible without blocking Codea, and native Windows spaces/non-ASCII behavior is covered by real Git execution.

Task 31 technical verification is complete and is ready for human acceptance. Human acceptance remains separate and is intentionally not marked true. Task 32 has not started.
