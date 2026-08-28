param(
  [Parameter(Mandatory=$true)][string[]]$File,
  [string]$ExpectedThumbprint = '',
  [string]$ExpectedSubject = ''
)

$ErrorActionPreference = 'Stop'

foreach ($item in $File) {
  if (-not (Test-Path -LiteralPath $item -PathType Leaf)) { throw "signed file not found: $item" }
  $resolved = (Resolve-Path -LiteralPath $item).Path
  $signature = Get-AuthenticodeSignature -LiteralPath $resolved
  if ($signature.Status -ne [Management.Automation.SignatureStatus]::Valid) {
    throw "Authenticode signature for $resolved is $($signature.Status): $($signature.StatusMessage)"
  }
  if (-not $signature.SignerCertificate) { throw "SignerCertificate missing for $resolved" }

  $thumbprint = $signature.SignerCertificate.Thumbprint
  if (-not [string]::IsNullOrWhiteSpace($ExpectedThumbprint)) {
    $want = ($ExpectedThumbprint -replace '\s','').ToUpperInvariant()
    if ($thumbprint.ToUpperInvariant() -ne $want) {
      throw "Thumbprint mismatch for ${resolved}: $thumbprint != $want"
    }
  }
  if (-not [string]::IsNullOrWhiteSpace($ExpectedSubject) -and $signature.SignerCertificate.Subject -notlike "*$ExpectedSubject*") {
    throw "Signer subject mismatch for ${resolved}: $($signature.SignerCertificate.Subject)"
  }

  [pscustomobject]@{
    File = $resolved
    Status = [string]$signature.Status
    Subject = $signature.SignerCertificate.Subject
    Thumbprint = $thumbprint
  }
}
