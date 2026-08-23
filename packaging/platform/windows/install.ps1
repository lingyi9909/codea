param(
  [Parameter(Mandatory=$true)][string]$PackageDir
)
$ErrorActionPreference = 'Stop'

$PackageDir = (Resolve-Path $PackageDir).Path
$versionFile = Join-Path $PackageDir 'VERSION'
if (-not (Test-Path $versionFile -PathType Leaf)) { throw 'VERSION missing' }
$version = (Get-Content $versionFile -Raw).Trim()
if ([string]::IsNullOrWhiteSpace($version)) { throw 'VERSION is empty' }

$manifestPath = Join-Path $PackageDir 'manifest.json'
if (-not (Test-Path $manifestPath -PathType Leaf)) { throw 'manifest.json missing' }
$manifest = Get-Content $manifestPath -Raw | ConvertFrom-Json
if ($manifest.schemaVersion -ne 1 -or $manifest.algorithm -ne 'sha256') { throw 'unsupported manifest schema' }

$manifestFiles = @{}
foreach ($entry in $manifest.files) {
  $rel = [string]$entry.path
  $parts = $rel -split '/'
  if ($rel.StartsWith('/') -or $rel.StartsWith('\\') -or ($parts -contains '..')) { throw "unsafe manifest path: $rel" }
  $path = Join-Path $PackageDir ($rel -replace '/', [IO.Path]::DirectorySeparatorChar)
  if (-not (Test-Path $path -PathType Leaf)) { throw "manifest file missing: $rel" }
  $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $path).Hash.ToLowerInvariant()
  if ($hash -ne ([string]$entry.sha256).ToLowerInvariant()) { throw "checksum mismatch: $rel" }
  if ((Get-Item -LiteralPath $path).Length -ne [int64]$entry.size) { throw "size mismatch: $rel" }
  $manifestFiles[$rel] = $true
}

$actualFiles = Get-ChildItem -LiteralPath $PackageDir -Recurse -File | ForEach-Object {
  $full = $_.FullName
  $rel = $full.Substring($PackageDir.Length).TrimStart('\\','/') -replace '\\','/'
  if ($rel -ne 'manifest.json') { $rel }
}
foreach ($rel in $actualFiles) {
  if (-not $manifestFiles.ContainsKey($rel)) { throw "unmanifested file: $rel" }
}

$pluginDir = Join-Path $PackageDir 'plugins'
if (-not (Test-Path $pluginDir -PathType Container)) { throw 'plugins directory missing' }
foreach ($name in @('package.json','bun.lock','bun.lockb')) {
  if (Test-Path (Join-Path $pluginDir $name)) { throw "runtime plugin dependency metadata found: $name" }
}
$pluginFiles = Get-ChildItem -LiteralPath $pluginDir -Filter '*.js' -File
if (-not $pluginFiles) { throw 'no plugin bundle found' }
$importPatterns = @(
  '(?:from\s*|import\s*\(|require\s*\()\s*["'']([^"'']+)["'']',
  '(?:^|[;\r\n])\s*import\s*["'']([^"'']+)["'']'
)
foreach ($plugin in $pluginFiles) {
  $text = Get-Content -LiteralPath $plugin.FullName -Raw
  foreach ($pattern in $importPatterns) {
    foreach ($match in [regex]::Matches($text, $pattern, [Text.RegularExpressions.RegexOptions]::Multiline)) {
      $spec = $match.Groups[1].Value
      if (-not ($spec.StartsWith('.') -or $spec.StartsWith('/') -or $spec.StartsWith('node:') -or $spec.StartsWith('bun:') -or $spec.StartsWith('data:'))) {
        throw "external import '$spec' in $($plugin.Name)"
      }
    }
  }
}

$home = if ($env:CODEA_HOME) { $env:CODEA_HOME } else { Join-Path $env:USERPROFILE '.codea' }
$versions = Join-Path $home 'versions'
$target = Join-Path $versions $version
if (Test-Path $target) { throw "version already installed: $version" }
$binDir = Join-Path $home 'bin'
New-Item -ItemType Directory -Force -Path $versions,$binDir | Out-Null
$tmp = Join-Path $versions ('.install-' + $version + '-' + $PID)
try {
  Copy-Item -Recurse -Force -LiteralPath $PackageDir -Destination $tmp
  Move-Item -LiteralPath $tmp -Destination $target
} finally {
  if (Test-Path $tmp) { Remove-Item -Recurse -Force -LiteralPath $tmp }
}

$currentTxt = Join-Path $home 'current.txt'
[IO.File]::WriteAllText($currentTxt, $target + [Environment]::NewLine, (New-Object Text.UTF8Encoding($false)))
$currentDir = Join-Path $home 'current'
if (Test-Path $currentDir) { Remove-Item -Force -Recurse $currentDir }
New-Item -ItemType Junction -Path $currentDir -Target $target | Out-Null

$shim = Join-Path $binDir 'codea.cmd'
$shimBody = @"
@echo off
setlocal
for %%I in ("%~dp0..") do set "CODEA_HOME=%%~fI"
set "CODEA_RUNTIME_CONFIG_DIR=%CODEA_HOME%\runtime-config"
if exist "%CODEA_HOME%\update.in-progress" (
  echo Error: Codea upgrade/rollback transaction is in progress; try again after recovery completes. 1>&2
  exit /b 75
)
set "CODEA_POINTER=%CODEA_HOME%\current.txt"
if not exist "%CODEA_POINTER%" (
  echo Error: Codea current pointer missing: %CODEA_POINTER% 1>&2
  exit /b 1
)
set /p "CODEA_CURRENT="<"%CODEA_POINTER%"
if not defined CODEA_CURRENT (
  echo Error: Codea current pointer is empty 1>&2
  exit /b 1
)
set "OPENCODE_BIN=%CODEA_CURRENT%\bin\opencode.exe"
set "CODEA_AGENTS_DIR=%CODEA_CURRENT%\agents"
set "CODEA_SKILLS_DIR=%CODEA_CURRENT%\skills"
set "CODEA_PLUGIN_BUNDLE=%CODEA_CURRENT%\plugins\index.js"
"%CODEA_CURRENT%\bin\codea.exe" %*
"@
[IO.File]::WriteAllText($shim, ($shimBody -replace "`n", "`r`n"), [Text.Encoding]::ASCII)

Write-Host "Installed Codea $version"
Write-Host "Add $binDir to PATH if needed."
