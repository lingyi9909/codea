param(
  [Parameter(Mandatory=$true)][string]$PackageDir
)
$ErrorActionPreference = 'Stop'

$PackageDir = (Resolve-Path $PackageDir).Path
$versionFile = Join-Path $PackageDir 'VERSION'
if (-not (Test-Path $versionFile)) { throw 'VERSION missing' }
$version = (Get-Content $versionFile -Raw).Trim()
if ([string]::IsNullOrWhiteSpace($version)) { throw 'VERSION is empty' }

$manifestPath = Join-Path $PackageDir 'manifest.json'
if (-not (Test-Path $manifestPath)) { throw 'manifest.json missing' }
$manifest = Get-Content $manifestPath -Raw | ConvertFrom-Json
if ($manifest.schemaVersion -ne 1 -or $manifest.algorithm -ne 'sha256') { throw 'unsupported manifest schema' }
foreach ($entry in $manifest.files) {
  if ($entry.path.StartsWith('/') -or $entry.path.Contains('..')) { throw "unsafe manifest path: $($entry.path)" }
  $path = Join-Path $PackageDir ($entry.path -replace '/', [IO.Path]::DirectorySeparatorChar)
  if (-not (Test-Path $path -PathType Leaf)) { throw "manifest file missing: $($entry.path)" }
  $hash = (Get-FileHash -Algorithm SHA256 $path).Hash.ToLowerInvariant()
  if ($hash -ne $entry.sha256.ToLowerInvariant()) { throw "checksum mismatch: $($entry.path)" }
  if ((Get-Item $path).Length -ne [int64]$entry.size) { throw "size mismatch: $($entry.path)" }
}

$home = if ($env:CODEA_HOME) { $env:CODEA_HOME } else { Join-Path $env:USERPROFILE '.codea' }
$versions = Join-Path $home 'versions'
$target = Join-Path $versions $version
if (Test-Path $target) { throw "version already installed: $version" }
New-Item -ItemType Directory -Force -Path $versions,(Join-Path $home 'bin') | Out-Null
$tmp = Join-Path $versions ('.install-' + $version + '-' + $PID)
try {
  Copy-Item -Recurse -Force $PackageDir $tmp
  Move-Item $tmp $target
} finally {
  if (Test-Path $tmp) { Remove-Item -Recurse -Force $tmp }
}

# Windows V1 uses an explicit pointer file rather than requiring symlink privileges.
Set-Content -Path (Join-Path $home 'current.txt') -Value $target -Encoding UTF8
$shim = Join-Path $home 'bin\codea.cmd'
@"
@echo off
set /p CODEA_CURRENT=<"$home\current.txt"
"%CODEA_CURRENT%\bin\codea.exe" %*
"@ | Set-Content -Path $shim -Encoding ASCII

Write-Host "Installed Codea $version"
Write-Host "Add $home\bin to PATH if needed."
