$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
$run = Join-Path $root '.run'
$profileFile = Join-Path $run 'dev-profile.json'

if (Get-ChildItem $run -Filter *.pid -ErrorAction SilentlyContinue) {
  throw 'Development PID files already exist. Run backend/tools/stop-dev.ps1 before launching again.'
}

$profiles = @(
  [pscustomobject]@{Name='standard'; Platform=18082; PlatformGRPC=19082; Gateway=18081; FrontendOffset=0},
  [pscustomobject]@{Name='isolated-38'; Platform=38182; PlatformGRPC=39182; Gateway=38181; FrontendOffset=30000},
  [pscustomobject]@{Name='isolated-48'; Platform=48182; PlatformGRPC=49182; Gateway=48181; FrontendOffset=40000},
  [pscustomobject]@{Name='isolated-58'; Platform=58182; PlatformGRPC=59182; Gateway=58181; FrontendOffset=50000}
)
function Test-PortsAvailable([int[]]$Ports) {
  $listeners = @()
  try {
    foreach ($port in $Ports) {
      $listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, $port)
      $listener.Start()
      $listeners += $listener
    }
    return $true
  } catch { return $false }
  finally { foreach ($listener in $listeners) { $listener.Stop() } }
}
$busyPorts = [Net.NetworkInformation.IPGlobalProperties]::GetIPGlobalProperties().GetActiveTcpListeners().Port
$profile = $null
foreach ($candidate in $profiles) {
  $ports = @(
    $candidate.Platform, $candidate.PlatformGRPC, $candidate.Gateway,
    (15173+$candidate.FrontendOffset),
    (15175+$candidate.FrontendOffset), (15176+$candidate.FrontendOffset), (15180+$candidate.FrontendOffset)
  )
  if (-not ($ports | Where-Object { $busyPorts -contains $_ }) -and (Test-PortsAvailable $ports)) { $profile = $candidate; break }
}
if (-not $profile) { throw 'No complete local platform port profile is available.' }

$platformUrl = "http://127.0.0.1:$($profile.Platform)"
$gatewayUrl = "http://127.0.0.1:$($profile.Gateway)"
$adminUrl = "http://127.0.0.1:$(15173+$profile.FrontendOffset)"
$shopUrl = "http://127.0.0.1:$(15175+$profile.FrontendOffset)"
$liveUrl = "http://127.0.0.1:$(15176+$profile.FrontendOffset)"
$controlArtifactUrl = "http://127.0.0.1:$(15180+$profile.FrontendOffset)"

function Wait-LocalUrl([string]$Url) {
  $deadline = [DateTime]::UtcNow.AddSeconds(30)
  while ([DateTime]::UtcNow -lt $deadline) {
    try {
      $response = Invoke-WebRequest -Uri $Url -TimeoutSec 2
      if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 500) { return }
    } catch {}
    Start-Sleep -Milliseconds 250
  }
  throw "Timed out waiting for $Url"
}

try {
  & docker compose -f (Join-Path $root 'backend/deploy/compose.local.yml') up -d --wait registry-db
  if ($LASTEXITCODE -ne 0) { throw 'Failed to start the MySQL module registry.' }
  & docker compose -f (Join-Path $root 'backend/deploy/compose.local.yml') run --rm registry-migrate
  if ($LASTEXITCODE -ne 0) { throw 'Failed to apply platform control-plane migrations.' }
  & (Join-Path $PSScriptRoot 'start-dev.ps1') -PlatformPort $profile.Platform -PlatformGRPCPort $profile.PlatformGRPC -GatewayPort $profile.Gateway -FrontendPortOffset $profile.FrontendOffset
  Wait-LocalUrl $controlArtifactUrl
  & (Join-Path $PSScriptRoot 'register-platform-module.ps1') -PlatformUrl 'http://127.0.0.1:18070' -GRPCEndpoint "dns:///127.0.0.1:$($profile.PlatformGRPC)" -ArtifactUrl $controlArtifactUrl
  if ($LASTEXITCODE -ne 0) { throw 'Failed to register the Platform Control Plane module.' }
  Wait-LocalUrl "$gatewayUrl/health"
  Wait-LocalUrl $adminUrl
  Wait-LocalUrl $shopUrl
  Wait-LocalUrl $liveUrl

  [ordered]@{
    name=$profile.Name; platformUrl=$platformUrl; platformGrpcAddress="127.0.0.1:$($profile.PlatformGRPC)"; gatewayUrl=$gatewayUrl;
    adminUrl=$adminUrl; shopUrl=$shopUrl; liveUrl=$liveUrl; controlArtifactUrl=$controlArtifactUrl
  } | ConvertTo-Json | Set-Content -LiteralPath $profileFile -Encoding utf8

  Write-Host ''
  Write-Host "LiveShop platform is running with profile '$($profile.Name)':"
  Write-Host "  Admin:    $adminUrl"
  Write-Host "  Shop:     $shopUrl"
  Write-Host "  Live:     $liveUrl"
  Write-Host "  Platform Control Plane artifact: $controlArtifactUrl"
  Write-Host "  Platform: $platformUrl"
  Write-Host "  Platform gRPC: 127.0.0.1:$($profile.PlatformGRPC)"
  Write-Host "  Gateway:  $gatewayUrl"
  Write-Host 'Identity must run separately to issue browser Module Capabilities.'
  Write-Host 'Registry must already be listening on http://127.0.0.1:18070 before this script registers Platform.'
  Write-Host 'Start an external module repository to add business capabilities.'
} catch {
  & (Join-Path $PSScriptRoot 'stop-dev.ps1')
  throw
}
