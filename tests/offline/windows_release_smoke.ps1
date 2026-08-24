param(
  [Parameter(Mandatory=$true)][string]$PackageDir,
  [string]$EvidenceFile = ""
)
$ErrorActionPreference = 'Stop'
if ($env:OS -ne 'Windows_NT') { throw 'BLOCKED: Windows x64 host required' }
if ($env:PROCESSOR_ARCHITECTURE -notmatch 'AMD64') { throw "BLOCKED: Windows x64 required, got $env:PROCESSOR_ARCHITECTURE" }
if ($env:WSL_DISTRO_NAME) { throw 'FAIL: this gate must run on native Windows, not WSL' }

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
if ([string]::IsNullOrWhiteSpace($EvidenceFile)) {
  $EvidenceFile = Join-Path $RepoRoot 'tests\offline\evidence\task17-windows-x64-evidence.json'
}
$PackageDir = (Resolve-Path $PackageDir).Path
$install = Join-Path $PackageDir 'install\install.ps1'
if (-not (Test-Path $install -PathType Leaf)) { throw 'install.ps1 missing' }

# The acceptance gate must be executed in a genuinely isolated environment.
$publicReachable = $false
try {
  Invoke-WebRequest -UseBasicParsing -Uri 'https://github.com/' -TimeoutSec 3 | Out-Null
  $publicReachable = $true
} catch { }
if ($publicReachable) { throw 'FAIL: public HTTPS is reachable' }

$work = Join-Path ([IO.Path]::GetTempPath()) ('codea-win-release-' + [Guid]::NewGuid().ToString('N'))
$codeaHome = Join-Path $work 'home'
New-Item -ItemType Directory -Force -Path $work | Out-Null
$runtimeProcess = $null
$codeaProcess = $null
$originalPath = $env:PATH
try {
  $env:CODEA_HOME = $codeaHome
  & $install -PackageDir $PackageDir

  $current = Join-Path $codeaHome 'current'
  $currentItem = Get-Item -LiteralPath $current
  if ($currentItem.LinkType -ne 'Junction' -and -not ($currentItem.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
    throw 'FAIL: current is not a Junction/ReparsePoint'
  }
  $codeaExe = Join-Path $current 'bin\codea.exe'
  $opencodeExe = Join-Path $current 'bin\opencode.exe'
  $plugin = Join-Path $current 'plugins\index.js'
  $agents = Join-Path $current 'agents'
  $skills = Join-Path $current 'skills'
  $shim = Join-Path $codeaHome 'bin\codea.cmd'
  foreach ($path in @($codeaExe,$opencodeExe,$plugin,$shim)) {
    if (-not (Test-Path $path -PathType Leaf)) { throw "FAIL: installed file missing: $path" }
  }
  foreach ($path in @($agents,$skills)) {
    if (-not (Test-Path $path -PathType Container)) { throw "FAIL: installed directory missing: $path" }
  }
  $shimText = Get-Content -LiteralPath $shim -Raw
  foreach ($key in @('OPENCODE_BIN','CODEA_AGENTS_DIR','CODEA_SKILLS_DIR','CODEA_PLUGIN_BUNDLE')) {
    if (-not $shimText.Contains($key)) { throw "FAIL: codea.cmd missing $key" }
  }

  # G2: trap external package-manager invocation. A release startup must not
  # depend on npm/bun/pip/mvn installing anything at runtime.
  $sentinelDir = Join-Path $work 'package-manager-sentinels'
  $packageManagerMarker = Join-Path $work 'package-manager-invocations'
  New-Item -ItemType Directory -Force -Path $sentinelDir | Out-Null
  foreach ($commandName in @('npm','bun','pip','pip3','mvn')) {
    $wrapper = Join-Path $sentinelDir ($commandName + '.cmd')
    $wrapperText = "@echo off`r`n>>`"$packageManagerMarker`" echo $commandName`r`nexit /b 97`r`n"
    [IO.File]::WriteAllText($wrapper, $wrapperText, (New-Object Text.UTF8Encoding($false)))
  }
  $env:PATH = "$sentinelDir;$originalPath"

  $port = if ($env:SMOKE_PORT) { [int]$env:SMOKE_PORT } else { 49332 }
  $username = 'codea-smoke'
  $password = 'codea-smoke-pass'
  $runtimeConfig = Join-Path $work 'runtime-config'
  New-Item -ItemType Directory -Force -Path $runtimeConfig | Out-Null
  $pluginUri = ([Uri](Resolve-Path $plugin).Path).AbsoluteUri
  $runtimeConfigPayload = [ordered]@{
    plugin = @($pluginUri)
    permission = @{ bash = 'ask' }
  }
  [IO.File]::WriteAllText((Join-Path $runtimeConfig 'opencode.json'), (($runtimeConfigPayload | ConvertTo-Json -Depth 5) + [Environment]::NewLine), (New-Object Text.UTF8Encoding($false)))
  $env:OPENCODE_CONFIG_DIR = $runtimeConfig
  $env:OPENCODE_SERVER_USERNAME = $username
  $env:OPENCODE_SERVER_PASSWORD = $password
  $env:OPENCODE_DISABLE_MODELS_FETCH = '1'
  $env:OPENCODE_DISABLE_AUTOUPDATE = '1'
  $env:OPENCODE_DISABLE_EMBEDDED_WEB_UI = '1'
  $env:OPENCODE_DISABLE_LSP_DOWNLOAD = '1'
  $env:OPENCODE_DISABLE_DEFAULT_PLUGINS = '1'
  $runtimeLog = Join-Path $work 'opencode.log'
  $runtimeErr = Join-Path $work 'opencode.err.log'
  $runtimeProcess = Start-Process -FilePath $opencodeExe -ArgumentList @('serve','--hostname','127.0.0.1','--port',[string]$port) -PassThru -NoNewWindow -RedirectStandardOutput $runtimeLog -RedirectStandardError $runtimeErr

  $token = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("${username}:${password}"))
  $headers = @{ Authorization = "Basic $token" }
  $health = $null
  for ($i=0; $i -lt 100; $i++) {
    if ($runtimeProcess.HasExited) {
      throw "FAIL: packaged OpenCode exited early: $(Get-Content $runtimeErr -Raw -ErrorAction SilentlyContinue)"
    }
    try {
      $health = Invoke-RestMethod -Uri "http://127.0.0.1:$port/global/health" -Headers $headers -TimeoutSec 1
      if ($health.healthy -eq $true) { break }
    } catch { }
    Start-Sleep -Milliseconds 200
  }
  if ($null -eq $health -or $health.healthy -ne $true -or [string]$health.version -ne '1.18.11') {
    throw "FAIL: packaged OpenCode health invalid: $($health | ConvertTo-Json -Compress)"
  }

  # G2.1: live locked Runtime must register all 8 bundled enterprise tools while
  # public HTTPS is unavailable and without package-manager invocation.
  $encodedDirectory = [Uri]::EscapeDataString($work)
  $toolIds = Invoke-RestMethod -Uri "http://127.0.0.1:$port/experimental/tool/ids?directory=$encodedDirectory" -Headers $headers -TimeoutSec 30
  $expectedTools = @(
    'collect_review_context','analyze_test_project','write_test_file','run_project_test',
    'extract_api_spec','validate_api_example','write_document','dify-query'
  )
  $missingTools = @($expectedTools | Where-Object { $_ -notin $toolIds })
  if ($missingTools.Count -gt 0) { throw "FAIL: missing enterprise plugin tools: $($missingTools -join ', ')" }
  if (Test-Path $packageManagerMarker -PathType Leaf) {
    $invocations = (Get-Content $packageManagerMarker -ErrorAction SilentlyContinue) -join ' '
    if (-not [string]::IsNullOrWhiteSpace($invocations)) { throw "FAIL: Runtime invoked external package manager(s): $invocations" }
  }

  # Launch through codea.cmd, not the raw executable, so this proves the installed
  # launcher injects bundled runtime/agent/skill/plugin paths. Attach to the
  # already-healthy packaged runtime to isolate launcher/TUI startup from process supervision.
  $env:OPENCODE_URL = "http://127.0.0.1:$port"
  $env:OPENCODE_USERNAME = $username
  $env:OPENCODE_PASSWORD = $password
  $env:CODEA_RUNTIME_CONFIG_DIR = Join-Path $work 'codea-config'
  $cmdArg = '"' + $shim + '"'
  $codeaProcess = Start-Process -FilePath 'cmd.exe' -ArgumentList @('/d','/s','/c',$cmdArg) -PassThru
  Start-Sleep -Seconds 3
  if ($codeaProcess.HasExited) { throw "FAIL: installed Codea launcher exited during startup (exit=$($codeaProcess.ExitCode))" }
  Stop-Process -Id $codeaProcess.Id -Force -ErrorAction SilentlyContinue
  $codeaProcess = $null
  if (Test-Path $packageManagerMarker -PathType Leaf) {
    $invocations = (Get-Content $packageManagerMarker -ErrorAction SilentlyContinue) -join ' '
    if (-not [string]::IsNullOrWhiteSpace($invocations)) { throw "FAIL: Codea startup invoked external package manager(s): $invocations" }
  }

  $evidenceDir = Split-Path -Parent $EvidenceFile
  New-Item -ItemType Directory -Force -Path $evidenceDir | Out-Null
  $evidence = [ordered]@{
    timestamp = [DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')
    platform = 'windows-x64'
    nativeWindows = $true
    wslUsed = $false
    publicHttpsBlocked = $true
    installerPassed = $true
    currentJunctionValid = $true
    bundledResourcesPresent = $true
    opencodeServeHealthy = $true
    openCodeVersion = '1.18.11'
    codeaLauncherStarted = $true
    externalPackageManagerInvocations = 0
    enterprisePluginToolsRegistered = 8
    passedChecks = 10
    totalChecks = 10
  }
  [IO.File]::WriteAllText($EvidenceFile, (($evidence | ConvertTo-Json -Depth 4) + [Environment]::NewLine), (New-Object Text.UTF8Encoding($false)))
  Write-Host '[PASS] windows-x64 installed release: offline + install + no startup package manager + 8/8 plugin tools + OpenCode serve + Codea launcher; no WSL'
} finally {
  $env:PATH = $originalPath
  if ($codeaProcess -and -not $codeaProcess.HasExited) { Stop-Process -Id $codeaProcess.Id -Force -ErrorAction SilentlyContinue }
  if ($runtimeProcess -and -not $runtimeProcess.HasExited) { Stop-Process -Id $runtimeProcess.Id -Force -ErrorAction SilentlyContinue }
  Remove-Item Env:OPENCODE_URL -ErrorAction SilentlyContinue
  Remove-Item -Recurse -Force -LiteralPath $work -ErrorAction SilentlyContinue
}
