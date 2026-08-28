param(
  [Parameter(Mandatory=$true)][string]$Archive
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$repo = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
$archivePath = (Resolve-Path -LiteralPath $Archive).Path
$work = Join-Path $env:RUNNER_TEMP ('codea-task27-lifecycle-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $work | Out-Null

function Expand-CodeaPackage([string]$SourceArchive, [string]$Name) {
  $dest = Join-Path $work $Name
  Expand-Archive -LiteralPath $SourceArchive -DestinationPath $dest -Force
  $roots = @(Get-ChildItem -LiteralPath $dest -Directory)
  if ($roots.Count -ne 1) { throw "expected exactly one release root in $SourceArchive, found $($roots.Count)" }
  return $roots[0].FullName
}

function Write-Manifest([string]$PackageDir) {
  $entries = Get-ChildItem -LiteralPath $PackageDir -Recurse -File |
    Where-Object { $_.FullName -ne (Join-Path $PackageDir 'manifest.json') } |
    ForEach-Object {
      $rel = [IO.Path]::GetRelativePath($PackageDir, $_.FullName) -replace '\\','/'
      [pscustomobject]@{
        path = $rel
        size = $_.Length
        sha256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $_.FullName).Hash.ToLowerInvariant()
      }
    } |
    Sort-Object path
  [pscustomobject]@{ schemaVersion = 1; algorithm = 'sha256'; files = @($entries) } |
    ConvertTo-Json -Depth 5 |
    Set-Content -LiteralPath (Join-Path $PackageDir 'manifest.json') -Encoding UTF8
}

function New-VersionedPackage([string]$SourceArchive, [string]$Name, [string]$Version) {
  $root = Expand-CodeaPackage $SourceArchive $Name
  [IO.File]::WriteAllText((Join-Path $root 'VERSION'), $Version, (New-Object Text.UTF8Encoding($false)))
  Write-Manifest $root
  return $root
}

function Assert-NoMotw([string]$Path) {
  $stream = Get-Item -LiteralPath $Path -Stream Zone.Identifier -ErrorAction SilentlyContinue
  if ($stream) { throw "installed runtime still carries Zone.Identifier: $Path" }
}

function Invoke-InstalledRuntimeHealth([string]$CodeaHome, [string]$Scenario) {
  $shim = Join-Path $CodeaHome 'bin\codea.cmd'
  if (-not (Test-Path -LiteralPath $shim -PathType Leaf)) { throw "${Scenario}: installed codea.cmd missing" }

  $shimDir = Split-Path -Parent $shim
  $stdout = Join-Path $work ($Scenario + '-doctor.stdout.txt')
  $stderr = Join-Path $work ($Scenario + '-doctor.stderr.txt')
  $oldHome = $env:CODEA_HOME
  $oldPath = $env:PATH
  try {
    $env:CODEA_HOME = $CodeaHome
    $env:PATH = $shimDir + ';' + $oldPath
    $p = Start-Process -FilePath 'cmd.exe' -ArgumentList @('/d','/s','/c','codea doctor') -NoNewWindow -Wait -PassThru -RedirectStandardOutput $stdout -RedirectStandardError $stderr
  } finally {
    $env:CODEA_HOME = $oldHome
    $env:PATH = $oldPath
  }
  $text = ((Get-Content -LiteralPath $stdout -Raw -ErrorAction SilentlyContinue) + "`n" + (Get-Content -LiteralPath $stderr -Raw -ErrorAction SilentlyContinue))
  Write-Host "===== $Scenario doctor output ====="
  Write-Host $text

  if ($text -match '(?i)access is denied|CreateProcess.+(?:failed|denied)|Runtime 启动失败|start runtime:') {
    throw "${Scenario}: historical Windows startup failure reproduced"
  }
  if ($text -notmatch '(?m)^PASS\s+Runtime 健康\s+-\s+OpenCode 1\.18\.11 healthy\s*$') {
    throw "${Scenario}: OpenCode v1.18.11 Runtime Health PASS evidence missing (doctor exit=$($p.ExitCode))"
  }
  Write-Host "TASK27 $Scenario Runtime Health PASS"
}

function Install-And-Health([string]$PackageDir, [string]$CodeaHome, [string]$Scenario) {
  Remove-Item -LiteralPath $CodeaHome -Recurse -Force -ErrorAction SilentlyContinue
  $env:CODEA_HOME = $CodeaHome
  & (Join-Path $repo 'packaging/platform/windows/install.ps1') -PackageDir $PackageDir
  $version = (Get-Content -LiteralPath (Join-Path $PackageDir 'VERSION') -Raw).Trim()
  $installed = Join-Path $CodeaHome ('versions\' + $version)
  Assert-NoMotw (Join-Path $installed 'bin\opencode.exe')
  Invoke-InstalledRuntimeHealth $CodeaHome $Scenario
  return $installed
}

try {
  # A + D: formal release zip, fresh install, then immediate startup.
  $freshPkg = Expand-CodeaPackage $archivePath 'fresh-package'
  $freshHome = Join-Path $work 'fresh-home'
  [void](Install-And-Health $freshPkg $freshHome 'fresh install immediate start')

  # B: simulate a browser-downloaded/extracted executable carrying MOTW.
  $motwPkg = Expand-CodeaPackage $archivePath 'motw-package'
  foreach ($exe in @('bin\codea.exe','bin\opencode.exe')) {
    Set-Content -LiteralPath (Join-Path $motwPkg $exe) -Stream Zone.Identifier -Value "[ZoneTransfer]`r`nZoneId=3" -NoNewline
  }
  $motwHome = Join-Path $work 'motw-home'
  [void](Install-And-Health $motwPkg $motwHome 'MOTW install immediate start')

  # C: install and start from a CODEA_HOME containing spaces.
  $spacesPkg = Expand-CodeaPackage $archivePath 'spaces-package'
  $spacesHome = Join-Path $work 'Codea Home With Spaces'
  [void](Install-And-Health $spacesPkg $spacesHome 'path with spaces')

  # E + F: install two versions with the same real binaries. The second package
  # differs only by VERSION + regenerated manifest, then the real Windows
  # platform switcher changes current.txt for forward/rollback validation.
  $upgradeHome = Join-Path $work 'upgrade-home'
  $v1Pkg = New-VersionedPackage $archivePath 'upgrade-v1-package' '0.1.9-task27-a'
  $v2Pkg = New-VersionedPackage $archivePath 'upgrade-v2-package' '0.1.9-task27-b'
  $v1 = Install-And-Health $v1Pkg $upgradeHome 'upgrade baseline install'

  $env:CODEA_HOME = $upgradeHome
  & (Join-Path $repo 'packaging/platform/windows/install.ps1') -PackageDir $v2Pkg
  $v2 = Join-Path $upgradeHome 'versions\0.1.9-task27-b'
  Assert-NoMotw (Join-Path $v2 'bin\opencode.exe')
  Invoke-InstalledRuntimeHealth $upgradeHome 'upgrade immediate start'

  Push-Location (Join-Path $repo 'tui')
  try {
    go run ./tests/task27switch -home $upgradeHome -target $v1
    if ($LASTEXITCODE -ne 0) { throw "rollback platform switch failed with exit code $LASTEXITCODE" }
  } finally { Pop-Location }
  Invoke-InstalledRuntimeHealth $upgradeHome 'rollback immediate start'

  Push-Location (Join-Path $repo 'tui')
  try {
    go run ./tests/task27switch -home $upgradeHome -target $v2
    if ($LASTEXITCODE -ne 0) { throw "forward platform switch failed with exit code $LASTEXITCODE" }
  } finally { Pop-Location }
  Invoke-InstalledRuntimeHealth $upgradeHome 'upgrade reselect immediate start'

  Write-Host 'TASK27_WINDOWS_INSTALLED_PACKAGE_REAL_LIFECYCLE PASS'
} finally {
  Remove-Item -LiteralPath $work -Recurse -Force -ErrorAction SilentlyContinue
}
