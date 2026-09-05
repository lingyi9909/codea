#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/tui"

GOTOOLCHAIN=local go test ./internal/checkpoint \
  -run 'TestShadowInitializationDoesNotTouchProjectGitState|TestRestoreExactAndSafetyCheckpointCanUndoRestore|TestRestoreDoesNotTouchProjectHeadBranchIndexOrRefs|TestSpacesUnicodeWorkspaceLifecycle|TestGitUnavailableFailsVisibleWithoutMetadata|TestRestoreBlocksSymlinkAncestorEscapeWrite|TestApplyRestoreBlocksSymlinkAncestorEscapeDelete|TestSafetySkippedDirectoryProtectsDescendants|TestInterruptedRestoreReportsPartialFilesChanged' \
  -count=1

GOTOOLCHAIN=local go test ./internal/app ./internal/command ./cmd/codea \
  -run 'Checkpoint|Restore|Baseline|Verified|Composition|Builtin' \
  -count=1

printf '%s\n' \
  'TASK31_AGENT_CHECKPOINT PASS' \
  'SHADOW_GIT_ISOLATION PASS' \
  'BASELINE_RESTORE PASS' \
  'SAFETY_RESTORE PASS' \
  'PROJECT_INDEX_UNCHANGED PASS' \
  'GIT_UNAVAILABLE_FAIL_VISIBLE PASS' \
  'RESTORE_SYMLINK_ESCAPE_BLOCKED PASS' \
  'INTERRUPTED_RESTORE_INVALIDATES_REPO_CONTEXT PASS'
