param(
  [Parameter(Mandatory=$true)][string]$Archive,
  [Parameter(Mandatory=$true)][string]$Evidence,
  [Parameter(Mandatory=$true)][string]$ExpectedThumbprint
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$repo = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
$archivePath = (Resolve-Path -LiteralPath $Archive).Path
$evidencePath = (Resolve-Path -LiteralPath $Evidence).Path
$expected = ($ExpectedThumbprint -replace '\s','').ToUpperInvariant()
$checksumPath = $archivePath + '.sha256'
if (-not (Test-Path -LiteralPath $checksumPath -PathType Leaf)) { throw 'finalized release SHA256 sidecar missing' }

$actualArchiveHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash.ToLowerInvariant()
$checksumText = (Get-Content -LiteralPath $checksumPath -Raw).Trim()
if ($checksumText -notmatch '^([0-9a-fA-F]{64})\s+') { throw 'invalid finalized release SHA256 sidecar format' }
$declaredArchiveHash = $Matches[1].ToLowerInvariant()
if ($declaredArchiveHash -ne $actualArchiveHash) {
  throw "finalized release SHA256 mismatch: $declaredArchiveHash != $actualArchiveHash"
}

$work = Join-Path $env:RUNNER_TEMP ('codea-task27-stable-finalized-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $work | Out-Null
try {
  Expand-Archive -LiteralPath $archivePath -DestinationPath $work -Force
  $roots = @(Get-ChildItem -LiteralPath $work -Directory)
  if ($roots.Count -ne 1) { throw "expected exactly one finalized release root, found $($roots.Count)" }
  $root = $roots[0].FullName
  $codeaExe = Join-Path $root 'bin\codea.exe'
  $manifestPath = Join-Path $root 'manifest.json'
  if (-not (Test-Path -LiteralPath $codeaExe -PathType Leaf)) { throw 'signed codea.exe missing from finalized release' }
  if (-not (Test-Path -LiteralPath $manifestPath -PathType Leaf)) { throw 'manifest.json missing from finalized release' }

  $sig = Get-AuthenticodeSignature -LiteralPath $codeaExe
  if ([string]$sig.Status -ne 'Valid') { throw "finalized codea.exe Authenticode status is $($sig.Status): $($sig.StatusMessage)" }
  if (-not $sig.SignerCertificate) { throw 'finalized codea.exe signer certificate missing' }
  $actualThumbprint = ($sig.SignerCertificate.Thumbprint -replace '\s','').ToUpperInvariant()
  if ($actualThumbprint -ne $expected) { throw "finalized codea.exe signer thumbprint mismatch: $actualThumbprint != $expected" }

  $manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
  $manifestEntries = @($manifest.files | Where-Object { [string]$_.path -eq 'bin/codea.exe' })
  if ($manifestEntries.Count -ne 1) { throw "expected one manifest entry for bin/codea.exe, found $($manifestEntries.Count)" }
  $manifestEntry = $manifestEntries[0]
  $actualExeHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $codeaExe).Hash.ToLowerInvariant()
  $actualExeSize = (Get-Item -LiteralPath $codeaExe).Length
  if (([string]$manifestEntry.sha256).ToLowerInvariant() -ne $actualExeHash) {
    throw 'manifest bin/codea.exe SHA256 does not match finalized signed executable'
  }
  if ([int64]$manifestEntry.size -ne [int64]$actualExeSize) {
    throw 'manifest bin/codea.exe size does not match finalized signed executable'
  }

  $evidenceJson = Get-Content -LiteralPath $evidencePath -Raw | ConvertFrom-Json
  if ([string]$evidenceJson.signatureStatus -ne 'Valid') { throw "security evidence signatureStatus is $($evidenceJson.signatureStatus), expected Valid" }
  if ([string]::IsNullOrWhiteSpace([string]$evidenceJson.signerSubject)) { throw 'security evidence signerSubject is empty' }
  if ([string]::IsNullOrWhiteSpace([string]$evidenceJson.signerThumbprint)) { throw 'security evidence signerThumbprint is empty' }
  $evidenceThumbprint = (([string]$evidenceJson.signerThumbprint) -replace '\s','').ToUpperInvariant()
  if ($evidenceThumbprint -ne $expected) { throw "security evidence signerThumbprint mismatch: $evidenceThumbprint != $expected" }
  if (([string]$evidenceJson.releaseSha256).ToLowerInvariant() -ne $actualArchiveHash) {
    throw 'security evidence releaseSha256 does not match finalized ZIP SHA256'
  }

  Write-Host "TASK27 stable finalized codea.exe Authenticode PASS thumbprint=$actualThumbprint"
  Write-Host "TASK27 stable finalized manifest rebind PASS sha256=$actualExeHash size=$actualExeSize"
  Write-Host "TASK27 stable finalized ZIP SHA256 PASS sha256=$actualArchiveHash"
  Write-Host "TASK27 stable finalized security evidence PASS subject=$($evidenceJson.signerSubject)"
} finally {
  Remove-Item -LiteralPath $work -Recurse -Force -ErrorAction SilentlyContinue
}

# The finalized signed ZIP itself must still install and start the real bundled
# OpenCode runtime. Reuse the full installed-package lifecycle Gate so stable
# finalization cannot regress fresh/MOTW/spaces/upgrade/rollback behavior.
& (Join-Path $repo 'tests/release/task27-windows-installed-lifecycle.ps1') -Archive $archivePath
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host 'TASK27_STABLE_FINALIZED_RELEASE_E2E PASS'
