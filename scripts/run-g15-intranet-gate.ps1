param(
  [Parameter(Mandatory=$true)][string]$SourceCommit,
  [Parameter(Mandatory=$true)][string]$OpenCodeBin,
  [Parameter(Mandatory=$true)][string]$PluginBundle,
  [Parameter(Mandatory=$true)][string]$MavenMirrorUrl,
  [Parameter(Mandatory=$true)][string]$NpmRegistryUrl,
  [Parameter(Mandatory=$true)][string]$PypiIndexUrl,
  [Parameter(Mandatory=$true)][string]$GoProxyUrl,
  [string]$EvidenceFile = ""
)
$ErrorActionPreference = 'Stop'
$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
if ([string]::IsNullOrWhiteSpace($EvidenceFile)) {
  $EvidenceFile = Join-Path $RepoRoot 'tests\evidence\g15-intranet-evidence.json'
}
$OpenCodeBin = (Resolve-Path $OpenCodeBin).Path
$PluginBundle = (Resolve-Path $PluginBundle).Path
$GateFile = Join-Path (Split-Path -Parent $EvidenceFile) 'G15.json'

$goVersion = (& go version) -join ''
if ($goVersion -notmatch 'go1\.26\.5') { throw "G15 requires Go 1.26.5, got: $goVersion" }
$runtimeVersion = (& $OpenCodeBin --version 2>&1) -join ''
if ($runtimeVersion -notmatch '1\.18\.11') { throw "G15 requires OpenCode v1.18.11, got: $runtimeVersion" }

$old = @{}
$keys = @(
  'CODEA_G15_INTRANET','CODEA_SOURCE_COMMIT','OPENCODE_BIN','CODEA_PLUGIN_BUNDLE',
  'CODEA_G15_MAVEN_MIRROR_URL','CODEA_G15_NPM_REGISTRY_URL','CODEA_G15_PYPI_INDEX_URL','CODEA_G15_GOPROXY_URL','CODEA_G15_EVIDENCE','GOTOOLCHAIN'
)
foreach ($key in $keys) { $old[$key] = [Environment]::GetEnvironmentVariable($key, 'Process') }
try {
  $env:CODEA_G15_INTRANET = '1'
  $env:CODEA_SOURCE_COMMIT = $SourceCommit
  $env:OPENCODE_BIN = $OpenCodeBin
  $env:CODEA_PLUGIN_BUNDLE = $PluginBundle
  $env:CODEA_G15_MAVEN_MIRROR_URL = $MavenMirrorUrl
  $env:CODEA_G15_NPM_REGISTRY_URL = $NpmRegistryUrl
  $env:CODEA_G15_PYPI_INDEX_URL = $PypiIndexUrl
  $env:CODEA_G15_GOPROXY_URL = $GoProxyUrl
  $env:CODEA_G15_EVIDENCE = $EvidenceFile
  $env:GOTOOLCHAIN = 'local'

  Push-Location (Join-Path $RepoRoot 'tui')
  try {
    & go test ./tests/parity -run '^TestG15IntranetMirrorsThroughGeneralAgent$' -count=1 -v
    if ($LASTEXITCODE -ne 0) { throw "G15 General Agent mirror test failed with exit $LASTEXITCODE" }
  } finally { Pop-Location }

  $evidence = Get-Content -LiteralPath $EvidenceFile -Raw | ConvertFrom-Json
  if ($evidence.passed -ne $true -or $evidence.sourceCommit -ne $SourceCommit -or $evidence.passedChecks -ne 4 -or $evidence.totalChecks -ne 4) {
    throw "G15 evidence invalid: $($evidence | ConvertTo-Json -Compress)"
  }
  $gate = [ordered]@{
    id = 'G15'
    status = 'pass'
    evidence = 'real Codea general Agent on approved intranet host resolved Maven/npm/PyPI/Go dependencies through configured internal mirrors; 4/4 checks'
    sourceCommit = $SourceCommit
  }
  New-Item -ItemType Directory -Force -Path (Split-Path -Parent $GateFile) | Out-Null
  [IO.File]::WriteAllText($GateFile, (($gate | ConvertTo-Json -Depth 4) + [Environment]::NewLine), (New-Object Text.UTF8Encoding($false)))
  Write-Host "[PASS] G15 intranet mirror gate: $GateFile"
} finally {
  foreach ($key in $keys) {
    $value = $old[$key]
    if ($null -eq $value) { Remove-Item "Env:$key" -ErrorAction SilentlyContinue }
    else { [Environment]::SetEnvironmentVariable($key, $value, 'Process') }
  }
}
