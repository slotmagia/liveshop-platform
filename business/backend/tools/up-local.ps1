param(
  [switch]$Fresh,
  [switch]$Register
)

$ErrorActionPreference = 'Stop'
if ($PSVersionTable.PSVersion.Major -lt 7) {
  throw "This deployment requires PowerShell 7. Run: pwsh -File $PSCommandPath"
}

$tools = $PSScriptRoot
$compose = [IO.Path]::GetFullPath((Join-Path $tools '..\deploy\compose.local.yml'))

function Invoke-Native {
  param([Parameter(Mandatory)][scriptblock]$Command, [string]$FailureMessage)
  $previous = $ErrorActionPreference
  $ErrorActionPreference = 'Continue'
  try { & $Command } finally { $ErrorActionPreference = $previous }
  if ($LASTEXITCODE -ne 0 -and $FailureMessage) { throw $FailureMessage }
}

function Ensure-LocalNetwork {
  $network = Invoke-Native { docker network ls --filter name='^liveshop-local$' --format '{{.Name}}' }
  if ($network -ne 'liveshop-local') {
    Invoke-Native { docker network create liveshop-local | Out-Null } 'Failed to create the shared Docker network liveshop-local.'
  }
}

function Wait-Http([string]$Url, [int]$TimeoutMinutes = 5) {
  $deadline = [DateTime]::UtcNow.AddMinutes($TimeoutMinutes)
  while ([DateTime]::UtcNow -lt $deadline) {
    try {
      $response = Invoke-WebRequest -Uri $Url -TimeoutSec 3 -UseBasicParsing -SkipHttpErrorCheck
      if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 500) { return }
    } catch {}
    Start-Sleep -Milliseconds 500
  }
  throw "Timed out waiting for $Url"
}

function Wait-Ready([string]$Url, [int]$TimeoutMinutes = 5) {
  $deadline = [DateTime]::UtcNow.AddMinutes($TimeoutMinutes)
  while ([DateTime]::UtcNow -lt $deadline) {
    try {
      $response = Invoke-WebRequest -Uri $Url -TimeoutSec 3 -UseBasicParsing -SkipHttpErrorCheck
      if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 300) { return }
    } catch {}
    Start-Sleep -Milliseconds 500
  }
  throw "Timed out waiting for ready service $Url"
}

Ensure-LocalNetwork
$certs = Invoke-Native { docker volume ls -q --filter 'name=^liveshop-grpc-certs$' }
if ("$certs".Trim() -ne 'liveshop-grpc-certs') {
  throw 'Missing liveshop-grpc-certs. Start Registry first: liveshop-registry/business/backend/tools/up-local.ps1'
}
try {
  Wait-Ready 'http://127.0.0.1:18070/readyz'
} catch {
  throw 'Registry is not ready on http://127.0.0.1:18070/readyz. Start Registry first: liveshop-registry/business/backend/tools/up-local.ps1'
}

if ($Fresh) {
  Invoke-Native { docker compose -f $compose down -v --remove-orphans } 'Failed to reset the local Platform stack.'
}

Invoke-Native { docker compose -f $compose up -d --build --remove-orphans } 'Local Platform container deployment failed.'
Wait-Ready 'http://127.0.0.1:18082/readyz'
Wait-Http 'http://127.0.0.1:15180'

if ($Register) {
  & (Join-Path $tools 'register-platform-module.ps1') `
    -PlatformUrl 'http://127.0.0.1:18070' `
    -BackendOrigin 'http://platform:18082' `
    -GRPCEndpoint 'dns:///platform:19082' `
    -ArtifactUrl 'http://127.0.0.1:15180'
}

Invoke-Native { docker compose -f $compose ps }
Write-Host 'Platform local containers are running: http://127.0.0.1:18082  artifact http://127.0.0.1:15180'
Write-Host 'Gateway and Hosts are owned by liveshop-gateway. Start them after Identity.'
