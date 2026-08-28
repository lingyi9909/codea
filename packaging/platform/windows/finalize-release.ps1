param(
  [Parameter(Mandatory=$true)][string]$Archive,
  [Parameter(Mandatory=$true)][ValidateSet('preview','stable')][string]$Channel,
  [Parameter(Mandatory=$true)][string]$GitCommit,
  [Parameter(Mandatory=$true)][string]$ReleaseTag,
  [Parameter(Mandatory=$true)][string]$EvidenceOutput
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$repo = (Resolve-Path (Join-Path $PSScriptRoot '../../..')).Path
$archivePath = (Resolve-Path -LiteralPath $Archive).Path

$pfxBase64 = [string]$env:CODEA_WINDOWS_SIGNING_PFX_BASE64
$pfxPassword = [string]$env:CODEA_WINDOWS_SIGNING_PFX_PASSWORD
$timestampUrl = [string]$env:CODEA_WINDOWS_TIMESTAMP_URL
$hasPfx = -not [string]::IsNullOrWhiteSpace($pfxBase64)
$hasPassword = -not [string]::IsNullOrWhiteSpace($pfxPassword)

if ($Channel -eq 'stable' -and (-not $hasPfx -or -not $hasPassword)) {
  throw 'stable Windows release requires CODEA_WINDOWS_SIGNING_PFX_BASE64 and CODEA_WINDOWS_SIGNING_PFX_PASSWORD'
}
if ($hasPfx -xor $hasPassword) { throw 'incomplete Windows signing credentials' }

$extract = Join-Path $env:TEMP ('codea-release-finalize-' + [guid]::NewGuid().ToString('N'))
$pfx = Join-Path $env:TEMP ('codea-signing-' + [guid]::NewGuid().ToString('N') + '.pfx')
New-Item -ItemType Directory -Force -Path $extract | Out-Null
try {
  Expand-Archive -LiteralPath $archivePath -DestinationPath $extract -Force
  $roots = @(Get-ChildItem -LiteralPath $extract -Directory)
  if ($roots.Count -ne 1) { throw "expected one release root, found $($roots.Count)" }
  $root = $roots[0].FullName
  $codeaExe = Join-Path $root 'bin\codea.exe'
  if (-not (Test-Path -LiteralPath $codeaExe -PathType Leaf)) { throw 'codea.exe missing from release' }

  $signed = $false
  if ($hasPfx) {
    [IO.File]::WriteAllBytes($pfx, [Convert]::FromBase64String($pfxBase64))
    $cert = [Security.Cryptography.X509Certificates.X509Certificate2]::new($pfx, $pfxPassword)
    try { $thumbprint = $cert.Thumbprint } finally { $cert.Dispose() }
    & (Join-Path $PSScriptRoot 'sign-release.ps1') -File $codeaExe -PfxPath $pfx -PfxPassword $pfxPassword -TimestampUrl $timestampUrl
    & (Join-Path $PSScriptRoot 'verify-signature.ps1') -File $codeaExe -ExpectedThumbprint $thumbprint
    $signed = $true
  } elseif ($Channel -eq 'preview') {
    Write-Warning 'Finalizing explicitly unsigned preview. Stable releases fail closed without production Authenticode credentials.'
  }

  # Signing changes codea.exe bytes. Re-bind the installer manifest to the
  # exact finalized executable before recreating the archive.
  $entries = Get-ChildItem -LiteralPath $root -Recurse -File |
    Where-Object { $_.FullName -ne (Join-Path $root 'manifest.json') } |
    ForEach-Object {
      $rel = [IO.Path]::GetRelativePath($root, $_.FullName) -replace '\\','/'
      [pscustomobject]@{
        path = $rel
        size = $_.Length
        sha256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $_.FullName).Hash.ToLowerInvariant()
      }
    } | Sort-Object path
  [pscustomobject]@{ schemaVersion=1; algorithm='sha256'; files=@($entries) } |
    ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $root 'manifest.json') -Encoding UTF8

  Remove-Item -LiteralPath $archivePath -Force
  Compress-Archive -Path $root -DestinationPath $archivePath -CompressionLevel Optimal
  $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash.ToLowerInvariant()
  [IO.File]::WriteAllText(($archivePath + '.sha256'), "$hash  $([IO.Path]::GetFileName($archivePath))`n", (New-Object Text.UTF8Encoding($false)))

  & (Join-Path $PSScriptRoot 'build-security-evidence.ps1') -Archive $archivePath -GitCommit $GitCommit -ReleaseTag $ReleaseTag -Output $EvidenceOutput

  [pscustomobject]@{
    archive = $archivePath
    sha256 = $hash
    signed = $signed
    evidence = (Resolve-Path -LiteralPath $EvidenceOutput).Path
  }
} finally {
  Remove-Item -LiteralPath $extract -Recurse -Force -ErrorAction SilentlyContinue
  Remove-Item -LiteralPath $pfx -Force -ErrorAction SilentlyContinue
  $pfxPassword = $null
  $pfxBase64 = $null
}
