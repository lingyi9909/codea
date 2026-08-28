param(
  [Parameter(Mandatory=$true)][string[]]$File,
  [Parameter(Mandatory=$true)][string]$PfxPath,
  [Parameter(Mandatory=$true)][string]$PfxPassword,
  [string]$TimestampUrl = ''
)

$ErrorActionPreference = 'Stop'
$codeSigningEku = '1.3.6.1.5.5.7.3.3'

function Resolve-SignTool {
  $cmd = Get-Command signtool.exe -ErrorAction SilentlyContinue
  if ($cmd) { return $cmd.Source }
  $kits = Join-Path ${env:ProgramFiles(x86)} 'Windows Kits\10\bin'
  if (Test-Path $kits) {
    $candidate = Get-ChildItem -LiteralPath $kits -Recurse -Filter signtool.exe -File -ErrorAction SilentlyContinue |
      Where-Object { $_.FullName -match '\\x64\\signtool\.exe$' } |
      Sort-Object FullName -Descending |
      Select-Object -First 1
    if ($candidate) { return $candidate.FullName }
  }
  throw 'signtool.exe not found; install the Windows SDK signing tools'
}

if (-not (Test-Path -LiteralPath $PfxPath -PathType Leaf)) { throw "PFX not found: $PfxPath" }
$resolvedPfx = (Resolve-Path -LiteralPath $PfxPath).Path
$signTool = Resolve-SignTool
$cert = $null
try {
  $flags = [Security.Cryptography.X509Certificates.X509KeyStorageFlags]::EphemeralKeySet
  $cert = [Security.Cryptography.X509Certificates.X509Certificate2]::new($resolvedPfx, $PfxPassword, $flags)
  if (-not $cert.HasPrivateKey) { throw 'signing certificate does not contain a private key' }
  if ($cert.NotAfter -le [DateTime]::UtcNow) { throw 'signing certificate is expired' }
  $eku = $cert.Extensions | Where-Object { $_.Oid.Value -eq '2.5.29.37' } | Select-Object -First 1
  if (-not $eku -or -not ($eku.Format($false) -match 'Code Signing|1\.3\.6\.1\.5\.5\.7\.3\.3')) {
    throw "signing certificate missing Code Signing EKU $codeSigningEku"
  }

  foreach ($item in $File) {
    if (-not (Test-Path -LiteralPath $item -PathType Leaf)) { throw "file to sign not found: $item" }
    $resolved = (Resolve-Path -LiteralPath $item).Path
    $args = @('sign', '/fd', 'SHA256', '/f', $resolvedPfx, '/p', $PfxPassword)
    if (-not [string]::IsNullOrWhiteSpace($TimestampUrl)) {
      $args += @('/tr', $TimestampUrl, '/td', 'SHA256')
    }
    $args += $resolved
    & $signTool @args
    if ($LASTEXITCODE -ne 0) { throw "signtool signing failed for $resolved with exit code $LASTEXITCODE" }
  }
} finally {
  if ($cert) { $cert.Dispose() }
  $PfxPassword = $null
}
