param(
  [int]$PlatformPort = 18082,
  [int]$PlatformGRPCPort = 19082,
  [int]$GatewayPort = 18081,
  [int]$FrontendPortOffset = 0
)
$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
$gatewayRepositoryRoot = [IO.Path]::GetFullPath((Join-Path $root '..\..\liveshop-gateway\business'))
$gatewayBackendRoot = Join-Path $gatewayRepositoryRoot 'backend'
if (-not (Test-Path -LiteralPath (Join-Path $gatewayBackendRoot 'go.mod') -PathType Leaf)) {
  throw "Gateway repository is missing: $gatewayRepositoryRoot"
}
$run = Join-Path $root '.run'
$bin = Join-Path $run 'bin'
$grpcCerts = Join-Path $run 'grpc-certs'
$platformConfigFile = Join-Path $run 'platform.yaml'
$gatewayConfigFile = Join-Path $run 'gateway.yaml'
New-Item -ItemType Directory -Force -Path $run | Out-Null
New-Item -ItemType Directory -Force -Path $bin | Out-Null
$node = (Get-Command node.exe).Source
$vite = Join-Path $gatewayRepositoryRoot 'node_modules/vite/bin/vite.js'
if (-not (Test-Path -LiteralPath $vite -PathType Leaf)) {
  throw "Gateway frontend dependencies are missing; run npm install in $gatewayRepositoryRoot"
}
$registryUrl = "http://127.0.0.1:$PlatformPort"
$gatewayUrl = "http://127.0.0.1:$GatewayPort"
$corsOrigins = @(
  "http://127.0.0.1:$(15173+$FrontendPortOffset)",
  "http://127.0.0.1:$(15175+$FrontendPortOffset)", "http://127.0.0.1:$(15176+$FrontendPortOffset)",
  "http://127.0.0.1:$(15180+$FrontendPortOffset)",
  "http://127.0.0.1:15191", "http://127.0.0.1:15290", "http://127.0.0.1:15291"
) -join ','
$requestedPorts = @(
  $PlatformPort, $PlatformGRPCPort, $GatewayPort,
  (15173+$FrontendPortOffset),
  (15175+$FrontendPortOffset), (15176+$FrontendPortOffset), (15180+$FrontendPortOffset)
)
$busyPorts = [Net.NetworkInformation.IPGlobalProperties]::GetIPGlobalProperties().GetActiveTcpListeners().Port
foreach ($port in $requestedPorts) {
  if ($busyPorts -contains $port) { throw "TCP port $port is already in use." }
  $listener = $null
  try {
    $listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, $port)
    $listener.Start()
  } catch { throw "TCP port $port cannot be bound: $($_.Exception.Message)" }
  finally { if ($listener) { $listener.Stop() } }
}

& go -C (Join-Path $root 'backend') run ./cmd/grpccerts -out $grpcCerts -force
if ($LASTEXITCODE -ne 0) { throw 'Failed to generate local Platform gRPC certificates.' }

$grpcCertificateFile = (Join-Path $grpcCerts 'server.pem').Replace('\', '/')
$grpcPrivateKeyFile = (Join-Path $grpcCerts 'server-key.pem').Replace('\', '/')
$grpcClientCAFile = (Join-Path $grpcCerts 'ca.pem').Replace('\', '/')
@"
# start-dev.ps1 生成的本地开发配置，禁止用于生产。
service: platform
log:
  level: info
  format: text
server:
  http: ":$PlatformPort"
  grpc: ":$PlatformGRPCPort"
database:
  url: liveshop:liveshop-local@tcp(127.0.0.1:33069)/liveshop_registry?parseTime=true&loc=UTC&charset=utf8mb4&collation=utf8mb4_0900_ai_ci
  max_open_connections: 40
  max_idle_connections: 10
  connection_max_lifetime: 30m
  connection_max_idle_time: 5m
module_capability:
  key_id: module-capability-dev-1
  public_key: 11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo
  issuer: liveshop-identity
workload_identity:
  issuer: liveshop-workload-identity
  http:
    gateway:
      key_id: gateway-workload-dev-1
      public_key: ky88xYQS66lbhNA-cUpijVuxRWcWAdFRgMIHFKF7PkA
      subject: liveshop-gateway
      permissions: [registry.routes.read]
    release:
      key_id: ci-workload-dev-1
      public_key: fkfxuRj0sxDYBT3U_qghrTrtjfv4y3djZObZ-EL_Zho
      subject: module-release-ci
      permissions: [registry.release.write, registry.activation.write]
    identity:
      key_id: identity-workload-dev-1
      public_key: 11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo
      subject: identity
      permissions: [platform.notify-event.dispatch]
  grpc:
    gateway:
      spiffe_id: spiffe://liveshop.local/gateway
      subject: liveshop-gateway
      permissions: [platform.registry.routes.read]
    identity:
      spiffe_id: spiffe://liveshop.local/identity
      subject: liveshop-identity
      permissions: [platform.registry.active-capabilities.read, platform.notify-event.dispatch]
http:
  allowed_origins: [http://gateway.internal]
  cookie_secure: false
grpc:
  tls:
    certificate_file: '$grpcCertificateFile'
    private_key_file: '$grpcPrivateKeyFile'
    client_ca_file: '$grpcClientCAFile'
"@ | Set-Content -LiteralPath $platformConfigFile -Encoding utf8

# Gateway reads exactly one YAML too, so local development exercises the same
# configuration path as Compose and production.
$gatewayOrigins = ($corsOrigins -split ',') | ForEach-Object { "  - $_" }
@"
# start-dev.ps1 生成的本地开发配置，禁止用于生产。
service: gateway
log:
  level: info
  format: text
server:
  http: ":$GatewayPort"
platform:
  registry_url: "$registryUrl"
module_capability:
  key_id: module-capability-dev-1
  public_key: 11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo
  issuer: liveshop-identity
workload_identity:
  key_id: gateway-workload-dev-1
  private_key: k51SIJI3oT-PJGXf2uWjL6jyTiDJ1Nwmk6l1ehqDrqA
  issuer: liveshop-workload-identity
http:
  allowed_origins:
$($gatewayOrigins -join "`n")
route_refresh:
  interval: 5s
  timeout: 3s
"@ | Set-Content -LiteralPath $gatewayConfigFile -Encoding utf8

$builds = @(
  @{Name='platform'; Root=(Join-Path $root 'backend'); Package='./internal/cmd'},
  @{Name='gateway'; Root=$gatewayBackendRoot; Package='./internal/gateway/cmd'}
)
foreach ($build in $builds) {
  $output = Join-Path $bin "$($build.Name).exe"
  & go -C $build.Root build -o $output $build.Package
  if ($LASTEXITCODE -ne 0) { throw "Failed to build $($build.Name)." }
}

$processes = @(
  @{Name='platform'; File=(Join-Path $bin 'platform.exe'); Args=@('-config',$platformConfigFile); Root=(Join-Path $root 'backend')},
  @{Name='gateway'; File=(Join-Path $bin 'gateway.exe'); Args=@('-config',$gatewayConfigFile); Root=$gatewayRepositoryRoot},
  @{Name='app-admin'; File=$node; Args=@($vite,'--host','127.0.0.1','--port',(15173+$FrontendPortOffset),'--strictPort'); Root=(Join-Path $gatewayRepositoryRoot 'frontend-admin'); Env=@{VITE_GATEWAY_URL=$gatewayUrl}},
  @{Name='app-shop'; File=$node; Args=@($vite,'--host','127.0.0.1','--port',(15175+$FrontendPortOffset),'--strictPort'); Root=(Join-Path $gatewayRepositoryRoot 'frontend-shop'); Env=@{VITE_GATEWAY_URL=$gatewayUrl}},
  @{Name='app-live'; File=$node; Args=@($vite,'--host','127.0.0.1','--port',(15176+$FrontendPortOffset),'--strictPort'); Root=(Join-Path $gatewayRepositoryRoot 'frontend-live'); Env=@{VITE_GATEWAY_URL=$gatewayUrl}},
  @{Name='platform-control'; File=$node; Args=@($vite,'--host','127.0.0.1','--port',(15180+$FrontendPortOffset),'--strictPort'); Root=(Join-Path $root 'frontend-admin'); Env=@{}}
)
$started = @()
try {
  foreach ($item in $processes) {
    $pidFile = Join-Path $run "$($item.Name).pid"
    if (Test-Path $pidFile) { throw "$($item.Name) already has a PID file; run backend/tools/stop-dev.ps1 first." }
    $stdout = Join-Path $run "$($item.Name).log"
    $stderr = Join-Path $run "$($item.Name).error.log"
    $startParameters = @{
      FilePath = $item.File
      ArgumentList = $item.Args
      WorkingDirectory = $item.Root
      RedirectStandardOutput = $stdout
      RedirectStandardError = $stderr
      WindowStyle = 'Hidden'
      PassThru = $true
    }
    if ($item.ContainsKey('Env') -and $item.Env.Count -gt 0) {
      $startParameters.Environment = $item.Env
    }
    $proc = Start-Process @startParameters
    Set-Content -LiteralPath $pidFile -Value $proc.Id
    $started += @{Name=$item.Name; Process=$proc; PidFile=$pidFile}
  }
  Start-Sleep -Milliseconds 500
  foreach ($item in $started) {
    $item.Process.Refresh()
    if ($item.Process.HasExited) { throw "$($item.Name) exited during startup; inspect .run/$($item.Name).error.log." }
  }
} catch {
  foreach ($item in $started) {
    if (-not $item.Process.HasExited) { & taskkill.exe /PID $item.Process.Id /T /F 2>$null | Out-Null }
    Remove-Item -LiteralPath $item.PidFile -Force -ErrorAction SilentlyContinue
  }
  throw
}
Write-Output "Platform development processes started: platform-http=$PlatformPort platform-grpc=$PlatformGRPCPort gateway=$GatewayPort."
