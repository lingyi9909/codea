$ErrorActionPreference = 'Stop'

$Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location (Join-Path $Root 'tui')

$env:GOTOOLCHAIN = 'local'
$env:CGO_ENABLED = '0'

& go test ./internal/checkpoint -run 'TestShadowInitializationDoesNotTouchProjectGitState|TestRestoreExactAndSafetyCheckpointCanUndoRestore|TestRestoreDoesNotTouchProjectHeadBranchIndexOrRefs|TestSpacesUnicodeWorkspaceLifecycle|TestGitUnavailableFailsVisibleWithoutMetadata|TestRestoreBlocksJunctionAncestorEscape|TestInterruptedRestoreReportsPartialFilesChanged' -count=1
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

& go test ./internal/app ./internal/command ./cmd/codea -run 'Checkpoint|Restore|Baseline|Verified|Composition|Builtin' -count=1
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

& go build ./cmd/codea
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Output 'TASK31_AGENT_CHECKPOINT PASS'
Write-Output 'SHADOW_GIT_ISOLATION PASS'
Write-Output 'BASELINE_RESTORE PASS'
Write-Output 'SAFETY_RESTORE PASS'
Write-Output 'PROJECT_INDEX_UNCHANGED PASS'
Write-Output 'WINDOWS_SPACES_UNICODE PASS'
Write-Output 'GIT_UNAVAILABLE_FAIL_VISIBLE PASS'
Write-Output 'RESTORE_JUNCTION_ESCAPE_BLOCKED PASS'
Write-Output 'INTERRUPTED_RESTORE_INVALIDATES_REPO_CONTEXT PASS'
