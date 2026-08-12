param(
  [int]$PlatformPort = 8082,
  [int]$PlatformGRPCPort = 9082,
  [int]$GatewayPort = 8081,
  [int]$FrontendPortOffset = 0
)
$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
$gatewayRepositoryRoot = [IO.Path]::GetFullPath((Join-Path $root '..\liveshop-gateway'))
$gatewayBackendRoot = Join-Path $gatewayRepositoryRoot 'backend'
if (-not (Test-Path -LiteralPath (Join-Path $gatewayBackendRoot 'go.mod') -PathType Leaf)) {
  throw "Gateway repository is missing: $gatewayRepositoryRoot"
}
$run = Join-Path $root '.run'
$bin = Join-Path $run 'bin'
$grpcCerts = Join-Path $run 'grpc-certs'
$platformConfigFile = Join-Path $run 'platform.yaml'
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
  "http://127.0.0.1:$(5173+$FrontendPortOffset)", "http://127.0.0.1:$(5174+$FrontendPortOffset)",
  "http://127.0.0.1:$(5175+$FrontendPortOffset)", "http://127.0.0.1:$(5176+$FrontendPortOffset)"
  "http://127.0.0.1:$(5180+$FrontendPortOffset)"
) -join ','
$requestedPorts = @(
  $PlatformPort, $PlatformGRPCPort, $GatewayPort,
  (5173+$FrontendPortOffset), (5174+$FrontendPortOffset),
  (5175+$FrontendPortOffset), (5176+$FrontendPortOffset), (5180+$FrontendPortOffset)
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
  url: postgres://liveshop:liveshop-local@127.0.0.1:54329/liveshop_registry?sslmode=disable
  max_open_connections: 40
  max_idle_connections: 10
  connection_max_lifetime: 30m
  connection_max_idle_time: 5m
module_session:
  key_id: module-session-dev-1
  private_key: nWGxne_9WmC6hEr0kuwsxERJxWl7MmkZcDusAxyuf2A
  public_key: 11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo
  issuer: liveshop-platform
access_identity:
  key_id: access-identity-dev-1
  private_key: TM0Imyj_ltqdtsNG7BFOD1uKMZ81q6Yk2oz27U-4pvs
  public_key: PUAXw-hDiVqStwqnTRt-vJyYLM8uxJaMwM1V8Sr0Zgw
  issuer: liveshop-identity
workload_identity:
  issuer: liveshop-workload-identity
  gateway:
    key_id: gateway-workload-dev-1
    public_key: ky88xYQS66lbhNA-cUpijVuxRWcWAdFRgMIHFKF7PkA
    spiffe_id: spiffe://liveshop.local/gateway
    subject: liveshop-gateway
    permissions: [registry.routes.read, registry.capabilities.read, platform.registry.routes.read, platform.registry.capabilities.read]
  release:
    key_id: ci-workload-dev-1
    public_key: fkfxuRj0sxDYBT3U_qghrTrtjfv4y3djZObZ-EL_Zho
    spiffe_id: spiffe://liveshop.local/module-release-ci
    subject: module-release-ci
    permissions: [registry.release.write, registry.activation.write]
http:
  allowed_origins: [http://gateway.internal]
  cookie_secure: false
grpc:
  tls:
    certificate_file: '$grpcCertificateFile'
    private_key_file: '$grpcPrivateKeyFile'
    client_ca_file: '$grpcClientCAFile'
"@ | Set-Content -LiteralPath $platformConfigFile -Encoding utf8

$builds = @(
  @{Name='platform'; Root=(Join-Path $root 'backend'); Package='./internal/platform/cmd'},
  @{Name='gateway'; Root=$gatewayBackendRoot; Package='./internal/gateway/cmd'}
)
foreach ($build in $builds) {
  $output = Join-Path $bin "$($build.Name).exe"
  & go -C $build.Root build -o $output $build.Package
  if ($LASTEXITCODE -ne 0) { throw "Failed to build $($build.Name)." }
}

$processes = @(
  @{Name='platform'; File=(Join-Path $bin 'platform.exe'); Args=@('-config',$platformConfigFile); Root=(Join-Path $root 'backend')},
  @{Name='gateway'; File=(Join-Path $bin 'gateway.exe'); Args=@(); Root=$gatewayRepositoryRoot; Env=@{HTTP_ADDR=":$GatewayPort";PLATFORM_REGISTRY_URL=$registryUrl;WORKLOAD_IDENTITY_ISSUER='liveshop-workload-identity';WORKLOAD_KEY_ID='gateway-workload-dev-1';WORKLOAD_PRIVATE_KEY='k51SIJI3oT-PJGXf2uWjL6jyTiDJ1Nwmk6l1ehqDrqA';MODULE_SESSION_KEY_ID='module-session-dev-1';MODULE_SESSION_PUBLIC_KEY='11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo';MODULE_SESSION_ISSUER='liveshop-platform';CORS_ALLOWED_ORIGINS=$corsOrigins}},
  @{Name='app-admin'; File=$node; Args=@($vite,'--host','127.0.0.1','--port',(5173+$FrontendPortOffset),'--strictPort'); Root=(Join-Path $gatewayRepositoryRoot 'frontend-admin'); Env=@{VITE_GATEWAY_URL=$gatewayUrl}},
  @{Name='app-merch'; File=$node; Args=@($vite,'--host','127.0.0.1','--port',(5174+$FrontendPortOffset),'--strictPort'); Root=(Join-Path $gatewayRepositoryRoot 'frontend-merch'); Env=@{VITE_GATEWAY_URL=$gatewayUrl}},
  @{Name='app-shop'; File=$node; Args=@($vite,'--host','127.0.0.1','--port',(5175+$FrontendPortOffset),'--strictPort'); Root=(Join-Path $gatewayRepositoryRoot 'frontend-shop'); Env=@{VITE_GATEWAY_URL=$gatewayUrl}},
  @{Name='app-live'; File=$node; Args=@($vite,'--host','127.0.0.1','--port',(5176+$FrontendPortOffset),'--strictPort'); Root=(Join-Path $gatewayRepositoryRoot 'frontend-live'); Env=@{VITE_GATEWAY_URL=$gatewayUrl}}
  @{Name='platform-iam'; File=$node; Args=@($vite,'--host','127.0.0.1','--port',(5180+$FrontendPortOffset),'--strictPort'); Root=(Join-Path $root 'frontend-admin'); Env=@{}}
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
