param([switch]$Volumes)

$ErrorActionPreference = 'Stop'
$compose = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\deploy\compose.local.yml'))
$args = @('-f', $compose, 'down', '--remove-orphans')
if ($Volumes) { $args += '-v' }
$previous = $ErrorActionPreference
$ErrorActionPreference = 'Continue'
try { docker compose @args } finally { $ErrorActionPreference = $previous }
if ($LASTEXITCODE -ne 0) { throw 'Failed to stop the local Platform containers.' }
if ($Volumes) {
  Write-Output 'Platform containers and Platform MySQL volumes were removed. Shared liveshop-grpc-certs is owned by Registry.'
} else {
  Write-Output 'Platform containers stopped. Named volumes were preserved.'
}
