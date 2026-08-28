param(
  [Parameter(Mandatory=$true)][string]$Archive,
  [Parameter(Mandatory=$true)][string]$GitCommit,
  [string]$ReleaseTag = '',
  [Parameter(Mandatory=$true)][string]$Output
)

$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '../../..')).Path
$archivePath = (Resolve-Path -LiteralPath $Archive).Path
$runtimeMeta = Get-Content -LiteralPath (Join-Path $repo 'runtime/version.json') -Raw | ConvertFrom-Json
$version = (Get-Content -LiteralPath (Join-Path $repo 'VERSION') -Raw).Trim()
$archiveItem = Get-Item -LiteralPath $archivePath
$archiveHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash.ToLowerInvariant()

$extract = Join-Path $env:TEMP ('codea-evidence-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $extract | Out-Null
try {
  Expand-Archive -LiteralPath $archivePath -DestinationPath $extract -Force
  $codeaExe = Get-ChildItem -LiteralPath $extract -Recurse -Filter codea.exe -File | Select-Object -First 1
  if (-not $codeaExe) { throw 'codea.exe missing from release archive' }
  $sig = Get-AuthenticodeSignature -LiteralPath $codeaExe.FullName
  $signer = $sig.SignerCertificate

  $evidence = [ordered]@{
    schemaVersion = 1
    codeaVersion = $version
    releaseFile = $archiveItem.Name
    releaseSize = [int64]$archiveItem.Length
    releaseSha256 = $archiveHash
    gitCommit = $GitCommit
    releaseTag = $ReleaseTag
    openCodeVersion = [string]$runtimeMeta.openCodeVersion
    openCodeChecksum = [string]$runtimeMeta.platforms.'windows-x64'.checksum
    signatureStatus = [string]$sig.Status
    signerSubject = if ($signer) { [string]$signer.Subject } else { '' }
    signerThumbprint = if ($signer) { [string]$signer.Thumbprint } else { '' }
  }
  if ($evidence.openCodeVersion -ne '1.18.11') { throw "unexpected OpenCode version: $($evidence.openCodeVersion)" }
  if (-not $evidence.openCodeChecksum.StartsWith('sha256:')) { throw 'OpenCode checksum is not locked' }

  $outDir = Split-Path -Parent $Output
  if ($outDir) { New-Item -ItemType Directory -Force -Path $outDir | Out-Null }
  $evidence | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $Output -Encoding UTF8
  Write-Host "Windows security evidence written: $Output"
} finally {
  Remove-Item -LiteralPath $extract -Recurse -Force -ErrorAction SilentlyContinue
}
