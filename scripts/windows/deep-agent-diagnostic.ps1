param(
  [string]$CodeaHome = "",
  [int]$AgentTimeoutSeconds = 45,
  [switch]$NoRepair
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Net.Http

if ([string]::IsNullOrWhiteSpace($CodeaHome)) {
  if ($env:CODEA_HOME) { $CodeaHome = $env:CODEA_HOME }
  else { $CodeaHome = Join-Path $env:USERPROFILE '.codea' }
}
$CodeaHome = [IO.Path]::GetFullPath($CodeaHome)
$currentTxt = Join-Path $CodeaHome 'current.txt'
if (-not (Test-Path -LiteralPath $currentTxt -PathType Leaf)) { throw "Codea current pointer missing: $currentTxt" }
$current = (Get-Content -LiteralPath $currentTxt -Raw).Trim()
if (-not (Test-Path -LiteralPath $current -PathType Container)) { throw "Codea current version missing: $current" }

$openCodeBin = Join-Path $current 'bin\opencode.exe'
$pluginBundle = Join-Path $current 'plugins\index.js'
$persistentConfig = Join-Path $CodeaHome 'runtime-config'
$materializedAgents = Join-Path $persistentConfig 'agents'
$doctorWorkspace = Join-Path $CodeaHome 'doctor-workspace'
foreach ($required in @($openCodeBin, $pluginBundle, $persistentConfig)) {
  if (-not (Test-Path -LiteralPath $required)) { throw "Required Codea path missing: $required" }
}
New-Item -ItemType Directory -Force -Path $doctorWorkspace | Out-Null

$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$diagRoot = Join-Path $env:TEMP ("codea-agent-diag-" + $stamp)
New-Item -ItemType Directory -Force -Path $diagRoot | Out-Null
$utf8NoBom = New-Object Text.UTF8Encoding($false)

function Get-FreePort {
  $listener = New-Object Net.Sockets.TcpListener([Net.IPAddress]::Loopback, 0)
  $listener.Start()
  try { return ([Net.IPEndPoint]$listener.LocalEndpoint).Port }
  finally { $listener.Stop() }
}

function Get-PluginUri([string]$Path) {
  $resolved = (Resolve-Path -LiteralPath $Path).Path
  return (New-Object System.Uri($resolved)).AbsoluteUri
}

function New-IsolatedLayout([string]$Name, [bool]$IncludePlugin, [bool]$IncludeAgents) {
  $root = Join-Path $diagRoot $Name
  $cfg = Join-Path $root 'config'
  $xdg = Join-Path $root 'xdg'
  foreach ($p in @($cfg, (Join-Path $xdg 'config'), (Join-Path $xdg 'data'), (Join-Path $xdg 'cache'), (Join-Path $xdg 'state'))) {
    New-Item -ItemType Directory -Force -Path $p | Out-Null
  }
  if ($IncludePlugin) {
    $json = @{ plugin = @((Get-PluginUri $pluginBundle)) } | ConvertTo-Json -Depth 5
    [IO.File]::WriteAllText((Join-Path $cfg 'opencode.json'), $json + [Environment]::NewLine, $utf8NoBom)
  }
  if ($IncludeAgents) {
    if (-not (Test-Path -LiteralPath $materializedAgents -PathType Container)) {
      throw "Materialized agents missing: $materializedAgents. Run codea doctor once before this diagnostic."
    }
    $target = Join-Path $cfg 'agents'
    New-Item -ItemType Directory -Force -Path $target | Out-Null
    Copy-Item -LiteralPath (Join-Path $materializedAgents '*') -Destination $target -Recurse -Force
  }
  return @{ Root=$root; Config=$cfg; Xdg=$xdg }
}

function Get-InternalLogTail([string]$XdgData, [int]$Lines = 60) {
  $logRoot = Join-Path $XdgData 'opencode\log'
  if (-not (Test-Path -LiteralPath $logRoot -PathType Container)) { return @() }
  $latest = Get-ChildItem -LiteralPath $logRoot -File -ErrorAction SilentlyContinue | Sort-Object LastWriteTime -Descending | Select-Object -First 1
  if (-not $latest) { return @() }
  return @(Get-Content -LiteralPath $latest.FullName -Tail $Lines -ErrorAction SilentlyContinue)
}

function Invoke-JsonGet([string]$Url, [string]$User, [string]$Password, [string]$Directory, [int]$TimeoutSeconds) {
  $handler = New-Object System.Net.Http.HttpClientHandler
  $handler.UseProxy = $false
  $client = New-Object System.Net.Http.HttpClient($handler)
  $client.Timeout = [TimeSpan]::FromSeconds($TimeoutSeconds)
  try {
    $request = New-Object System.Net.Http.HttpRequestMessage([System.Net.Http.HttpMethod]::Get, $Url)
    $token = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes($User + ':' + $Password))
    $request.Headers.Authorization = New-Object System.Net.Http.Headers.AuthenticationHeaderValue('Basic', $token)
    if (-not [string]::IsNullOrWhiteSpace($Directory)) { $request.Headers.Add('x-opencode-directory', $Directory) }
    $response = $client.SendAsync($request).GetAwaiter().GetResult()
    $body = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    return @{ Ok=$response.IsSuccessStatusCode; Status=[int]$response.StatusCode; Body=$body; Error='' }
  } catch {
    return @{ Ok=$false; Status=0; Body=''; Error=$_.Exception.GetBaseException().Message }
  } finally {
    $client.Dispose()
    $handler.Dispose()
  }
}

function Wait-Health([string]$BaseUrl, [string]$User, [string]$Password, [int]$Seconds) {
  $deadline = (Get-Date).AddSeconds($Seconds)
  $last = $null
  while ((Get-Date) -lt $deadline) {
    $last = Invoke-JsonGet ($BaseUrl + '/global/health') $User $Password '' 2
    if ($last.Ok) { return $last }
    Start-Sleep -Milliseconds 250
  }
  return $last
}

function Stop-DiagnosticProcess($Process) {
  if (-not $Process) { return }
  try {
    if (-not $Process.HasExited) {
      & taskkill.exe /PID $Process.Id /T /F 2>$null | Out-Null
      $Process.WaitForExit(5000) | Out-Null
    }
  } catch {
    try { Stop-Process -Id $Process.Id -Force -ErrorAction SilentlyContinue } catch {}
  }
}

function Invoke-Scenario(
  [string]$Name,
  [string]$ConfigDir,
  [string]$XdgRoot,
  [string]$Description
) {
  Write-Host ""
  Write-Host ("=== {0}: {1} ===" -f $Name, $Description)
  $port = Get-FreePort
  $user = 'opencode'
  $password = 'codea-deep-diagnostic'
  $baseUrl = "http://127.0.0.1:$port"
  $stdoutFile = Join-Path $diagRoot ($Name + '-stdout.log')
  $stderrFile = Join-Path $diagRoot ($Name + '-stderr.log')

  $psi = New-Object Diagnostics.ProcessStartInfo
  $psi.FileName = $openCodeBin
  $psi.Arguments = "serve --hostname 127.0.0.1 --port $port"
  $psi.WorkingDirectory = $doctorWorkspace
  $psi.UseShellExecute = $false
  $psi.CreateNoWindow = $true
  $psi.RedirectStandardOutput = $true
  $psi.RedirectStandardError = $true

  $psi.EnvironmentVariables['OPENCODE_CONFIG_DIR'] = $ConfigDir
  $psi.EnvironmentVariables['OPENCODE_SERVER_USERNAME'] = $user
  $psi.EnvironmentVariables['OPENCODE_SERVER_PASSWORD'] = $password
  $psi.EnvironmentVariables['OPENCODE_DISABLE_CLAUDE_CODE'] = '1'
  $psi.EnvironmentVariables['OPENCODE_DISABLE_MODELS_FETCH'] = '1'
  $psi.EnvironmentVariables['OPENCODE_DISABLE_AUTOUPDATE'] = '1'
  $psi.EnvironmentVariables['OPENCODE_DISABLE_EMBEDDED_WEB_UI'] = '1'
  $psi.EnvironmentVariables['OPENCODE_DISABLE_LSP_DOWNLOAD'] = '1'
  $psi.EnvironmentVariables['OPENCODE_DISABLE_DEFAULT_PLUGINS'] = '1'
  $psi.EnvironmentVariables['OPENCODE_DISABLE_EXTERNAL_SKILLS'] = '1'
  $psi.EnvironmentVariables['XDG_CONFIG_HOME'] = Join-Path $XdgRoot 'config'
  $psi.EnvironmentVariables['XDG_DATA_HOME'] = Join-Path $XdgRoot 'data'
  $psi.EnvironmentVariables['XDG_CACHE_HOME'] = Join-Path $XdgRoot 'cache'
  $psi.EnvironmentVariables['XDG_STATE_HOME'] = Join-Path $XdgRoot 'state'

  foreach ($p in @(
    $psi.EnvironmentVariables['XDG_CONFIG_HOME'],
    $psi.EnvironmentVariables['XDG_DATA_HOME'],
    $psi.EnvironmentVariables['XDG_CACHE_HOME'],
    $psi.EnvironmentVariables['XDG_STATE_HOME']
  )) { New-Item -ItemType Directory -Force -Path $p | Out-Null }

  $proc = New-Object Diagnostics.Process
  $proc.StartInfo = $psi
  $started = Get-Date
  $stdoutTask = $null
  $stderrTask = $null
  try {
    if (-not $proc.Start()) { throw 'failed to start opencode.exe' }
    $stdoutTask = $proc.StandardOutput.ReadToEndAsync()
    $stderrTask = $proc.StandardError.ReadToEndAsync()

    $health = Wait-Health $baseUrl $user $password 20
    if (-not $health -or -not $health.Ok) {
      $elapsed = [Math]::Round(((Get-Date) - $started).TotalSeconds, 2)
      return [pscustomobject]@{ Name=$Name; Health='FAIL'; Agent='SKIP'; Seconds=$elapsed; Error=('health: ' + $(if($health){$health.Error}else{'no response'})); ConfigDir=$ConfigDir; XdgRoot=$XdgRoot }
    }

    $agentStarted = Get-Date
    $agent = Invoke-JsonGet ($baseUrl + '/agent') $user $password $doctorWorkspace $AgentTimeoutSeconds
    $elapsed = [Math]::Round(((Get-Date) - $started).TotalSeconds, 2)
    $agentElapsed = [Math]::Round(((Get-Date) - $agentStarted).TotalSeconds, 2)
    if ($agent.Ok) {
      $count = 0
      try { $count = @($agent.Body | ConvertFrom-Json).Count } catch {}
      Write-Host ("PASS /agent in {0}s, count={1}" -f $agentElapsed, $count)
      return [pscustomobject]@{ Name=$Name; Health='PASS'; Agent='PASS'; Seconds=$elapsed; Error=''; ConfigDir=$ConfigDir; XdgRoot=$XdgRoot }
    }
    Write-Host ("FAIL /agent after {0}s: {1}" -f $agentElapsed, $agent.Error)
    return [pscustomobject]@{ Name=$Name; Health='PASS'; Agent='FAIL'; Seconds=$elapsed; Error=$agent.Error; ConfigDir=$ConfigDir; XdgRoot=$XdgRoot }
  } catch {
    $elapsed = [Math]::Round(((Get-Date) - $started).TotalSeconds, 2)
    return [pscustomobject]@{ Name=$Name; Health='FAIL'; Agent='SKIP'; Seconds=$elapsed; Error=$_.Exception.GetBaseException().Message; ConfigDir=$ConfigDir; XdgRoot=$XdgRoot }
  } finally {
    Stop-DiagnosticProcess $proc
    try {
      if ($stdoutTask) { [IO.File]::WriteAllText($stdoutFile, $stdoutTask.GetAwaiter().GetResult(), $utf8NoBom) }
      if ($stderrTask) { [IO.File]::WriteAllText($stderrFile, $stderrTask.GetAwaiter().GetResult(), $utf8NoBom) }
    } catch {}
  }
}

Write-Host "Codea deep /agent diagnostic"
Write-Host ("CodeaHome: {0}" -f $CodeaHome)
Write-Host ("Current:   {0}" -f $current)
Write-Host ("OpenCode:  {0}" -f $openCodeBin)
Write-Host ("Version:   {0}" -f (& $openCodeBin --version 2>$null))
Write-Host ("DiagRoot:  {0}" -f $diagRoot)
Write-Host "NOTE: no model/provider/API key is required for these /agent probes."

$bare = New-IsolatedLayout 'bare' $false $false
$agents = New-IsolatedLayout 'agents-only' $false $true
$plugin = New-IsolatedLayout 'plugin-only' $true $false
$full = New-IsolatedLayout 'fresh-full' $true $true

$results = @()
$results += Invoke-Scenario 'bare' $bare.Config $bare.Xdg 'bare OpenCode, no Codea plugin/agents'
$results += Invoke-Scenario 'agents-only' $agents.Config $agents.Xdg 'fresh state + Codea agents, no plugin'
$results += Invoke-Scenario 'plugin-only' $plugin.Config $plugin.Xdg 'fresh state + Codea plugin, no agents'
$results += Invoke-Scenario 'fresh-full' $full.Config $full.Xdg 'fresh state + Codea plugin + agents'

$currentXdg = Join-Path $persistentConfig 'xdg'
foreach ($p in @('config','data','cache','state')) { New-Item -ItemType Directory -Force -Path (Join-Path $currentXdg $p) | Out-Null }
$results += Invoke-Scenario 'current' $persistentConfig $currentXdg 'actual persistent Codea config + XDG state'

$freshCurrentXdg = Join-Path $diagRoot 'current-config-fresh-xdg'
foreach ($p in @('config','data','cache','state')) { New-Item -ItemType Directory -Force -Path (Join-Path $freshCurrentXdg $p) | Out-Null }
$results += Invoke-Scenario 'current-config-fresh-xdg' $persistentConfig $freshCurrentXdg 'actual opencode.json/agents + fresh XDG state'

Write-Host ""
Write-Host '=== SUMMARY ==='
$results | Format-Table Name,Health,Agent,Seconds,Error -AutoSize

function Find-Result([string]$Name) { return $results | Where-Object { $_.Name -eq $Name } | Select-Object -First 1 }
$rBare = Find-Result 'bare'
$rAgents = Find-Result 'agents-only'
$rPlugin = Find-Result 'plugin-only'
$rFull = Find-Result 'fresh-full'
$rCurrent = Find-Result 'current'
$rCurrentFreshXdg = Find-Result 'current-config-fresh-xdg'

$classification = 'UNKNOWN'
if ($rBare.Agent -ne 'PASS') { $classification = 'OPENCode_OR_WINDOWS_ENVIRONMENT' }
elseif ($rAgents.Agent -ne 'PASS') { $classification = 'CODEA_AGENT_FILES' }
elseif ($rPlugin.Agent -ne 'PASS') { $classification = 'CODEA_PLUGIN' }
elseif ($rFull.Agent -ne 'PASS') { $classification = 'PLUGIN_AGENT_INTERACTION' }
elseif ($rCurrent.Agent -ne 'PASS' -and $rCurrentFreshXdg.Agent -eq 'PASS') { $classification = 'PERSISTED_XDG_STATE' }
elseif ($rCurrent.Agent -ne 'PASS') { $classification = 'PERSISTED_CONFIG' }
elseif ($rCurrent.Agent -eq 'PASS') { $classification = 'CODEA_PARENT_OR_HTTP_CLIENT' }
Write-Host ("CLASSIFICATION={0}" -f $classification)

$repairBackup = ''
if ($classification -eq 'PERSISTED_XDG_STATE' -and -not $NoRepair) {
  Write-Host "Persistent XDG state is the isolated cause. Backing it up and resetting Codea-owned XDG state..."
  $xdgPath = Join-Path $persistentConfig 'xdg'
  $repairBackup = Join-Path $persistentConfig ("xdg.backup." + $stamp)
  if (Test-Path -LiteralPath $xdgPath) { Move-Item -LiteralPath $xdgPath -Destination $repairBackup }
  foreach ($p in @('config','data','cache','state')) { New-Item -ItemType Directory -Force -Path (Join-Path $xdgPath $p) | Out-Null }
  $repair = Invoke-Scenario 'current-after-repair' $persistentConfig $xdgPath 'actual config after safe XDG reset'
  $results += $repair
  if ($repair.Agent -eq 'PASS') {
    Write-Host 'AUTO_REPAIR=SUCCESS'
    Write-Host ("BACKUP={0}" -f $repairBackup)
    $classification = 'PERSISTED_XDG_STATE_REPAIRED'
  } else {
    Write-Host 'AUTO_REPAIR=FAILED; restoring previous XDG state'
    Remove-Item -LiteralPath $xdgPath -Recurse -Force -ErrorAction SilentlyContinue
    Move-Item -LiteralPath $repairBackup -Destination $xdgPath
    $repairBackup = ''
  }
}

$failed = $results | Where-Object { $_.Agent -eq 'FAIL' -or $_.Health -eq 'FAIL' } | Select-Object -First 1
if ($failed) {
  Write-Host ""
  Write-Host ("=== LOG TAIL FOR FIRST FAILURE: {0} ===" -f $failed.Name)
  $stderrPath = Join-Path $diagRoot ($failed.Name + '-stderr.log')
  $stdoutPath = Join-Path $diagRoot ($failed.Name + '-stdout.log')
  if (Test-Path -LiteralPath $stderrPath) { Get-Content -LiteralPath $stderrPath -Tail 80 -ErrorAction SilentlyContinue }
  if (Test-Path -LiteralPath $stdoutPath) { Get-Content -LiteralPath $stdoutPath -Tail 80 -ErrorAction SilentlyContinue }
  $xdgData = Join-Path $failed.XdgRoot 'data'
  Get-InternalLogTail $xdgData 80
}

$configPath = Join-Path $persistentConfig 'opencode.json'
if (Test-Path -LiteralPath $configPath) {
  try {
    $cfg = Get-Content -LiteralPath $configPath -Raw | ConvertFrom-Json
    $keys = @($cfg.PSObject.Properties.Name | Sort-Object)
    Write-Host ("CURRENT_CONFIG_KEYS={0}" -f ($keys -join ','))
  } catch {
    Write-Host ("CURRENT_CONFIG_PARSE_ERROR={0}" -f $_.Exception.Message)
  }
}

Write-Host ("FINAL_CLASSIFICATION={0}" -f $classification)
Write-Host ("DIAGNOSTIC_DIR={0}" -f $diagRoot)
if ($repairBackup) { Write-Host ("XDG_BACKUP={0}" -f $repairBackup) }
Write-Host 'Diagnostic complete.'
