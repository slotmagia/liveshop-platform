param(
  [string]$PlatformUrl = 'http://127.0.0.1:8082',
  [string]$GatewayUrl = 'http://127.0.0.1:8081'
)
$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
$profileFile = Join-Path $root '.run/dev-profile.json'
if (Test-Path $profileFile) {
  $profile = Get-Content -Raw -LiteralPath $profileFile | ConvertFrom-Json
  if (-not $PSBoundParameters.ContainsKey('PlatformUrl')) { $PlatformUrl = $profile.platformUrl }
  if (-not $PSBoundParameters.ContainsKey('GatewayUrl')) { $GatewayUrl = $profile.gatewayUrl }
}
$env:ACCESS_IDENTITY_PRIVATE_KEY = 'TM0Imyj_ltqdtsNG7BFOD1uKMZ81q6Yk2oz27U-4pvs'
$env:ACCESS_IDENTITY_KEY_ID = 'access-identity-dev-1'
$env:ACCESS_IDENTITY_ISSUER = 'liveshop-identity'
$kernelRoot = [IO.Path]::GetFullPath((Join-Path $root '..\kernel-go'))
$platformToken = & go -C $kernelRoot run ./cmd/identitytoken -realm PLATFORM
$merchantToken = & go -C $kernelRoot run ./cmd/identitytoken -realm MERCHANT

$gatewayHealth = Invoke-RestMethod "$GatewayUrl/health"
if (-not $gatewayHealth) { throw 'gateway health response is empty' }
foreach ($surface in @('admin','merch','shop','live')) {
  $surfaceToken = if ($surface -eq 'admin') { $platformToken } else { $merchantToken }
  $response = Invoke-RestMethod "$GatewayUrl/runtime/v1/contributions?surface=$surface" -Headers @{Authorization="Bearer $surfaceToken";'X-Liveshop-Surface'=$surface}
  if ($response.code -ne 0 -or $null -eq $response.data.items) {
    throw "platform contribution endpoint failed for surface '$surface'"
  }
}
$admin = Invoke-RestMethod "$GatewayUrl/runtime/v1/contributions?surface=admin" -Headers @{Authorization="Bearer $platformToken";'X-Liveshop-Surface'='admin'}
$iam = @($admin.data.items) | Where-Object {$_.moduleId -eq 'platform' -and $_.contribution.id -eq 'platform.admin.iam'} | Select-Object -First 1
if (-not $iam) { throw 'Platform IAM Admin contribution is missing.' }
$sessionBody = @{moduleId=$iam.moduleId;moduleVersion=$iam.moduleVersion;contributionId=$iam.contribution.id;surface='admin'} | ConvertTo-Json -Compress
$session = Invoke-RestMethod -Method Post -Uri "$GatewayUrl/runtime/v1/module-sessions" -Headers @{Authorization="Bearer $platformToken";'X-Liveshop-Surface'='admin'} -ContentType 'application/json' -Body $sessionBody
$roles = Invoke-RestMethod -Uri "$GatewayUrl/admin/platform/iam/roles" -Headers @{Authorization="Bearer $($session.data.token)";'X-Liveshop-Surface'='admin'}
if ($roles.code -ne 0 -or @($roles.data).Count -lt 1) { throw 'Platform IAM role query failed.' }
Write-Output "Platform smoke passed: four surfaces, IAM contribution, module session and Gateway ($(@($roles.data).Count) roles)."
