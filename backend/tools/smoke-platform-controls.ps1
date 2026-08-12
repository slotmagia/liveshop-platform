param(
  [string]$PlatformUrl = 'http://127.0.0.1:8082',
  [string]$GatewayUrl = 'http://127.0.0.1:8081'
)
$ErrorActionPreference = 'Stop'

function Assert-HttpStatus([scriptblock]$Action, [int]$Expected, [string]$Description) {
  try {
    & $Action | Out-Null
  } catch {
    $actual = [int]$_.Exception.Response.StatusCode
    if ($actual -eq $Expected) { return }
    throw "$Description returned HTTP $actual instead of $Expected. $($_.Exception.Message)"
  }
  throw "$Description unexpectedly succeeded; expected HTTP $Expected."
}

function Login([string]$Realm, [string]$Username) {
  $session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
  $password = if ($Realm -eq 'PLATFORM') {'admin'} else {'123456'}
  $login = @{realm=$Realm;username=$Username;password=$password}
  if ($Realm -eq 'MERCHANT') { $login.appId = 1001; $login.merchantId = 2001 }
  $body = $login | ConvertTo-Json -Compress
  $surface = if ($Realm -eq 'PLATFORM') {'admin'} else {'merch'}
  $response = Invoke-RestMethod -Method Post -Uri "$GatewayUrl/auth/login" -WebSession $session -Headers @{'X-Liveshop-Surface'=$surface} -ContentType 'application/json' -Body $body
  if ($response.code -ne 0 -or -not $response.data.accessToken) { throw "$Realm login failed." }
  return @{Session=$session;Token=$response.data.accessToken}
}

function Contribution([string]$Token, [string]$ContributionId) {
  $response = Invoke-RestMethod -Uri "$GatewayUrl/runtime/v1/contributions?surface=admin" -Headers @{Authorization="Bearer $Token";'X-Liveshop-Surface'='admin'}
  $item = @($response.data.items) | Where-Object {$_.contribution.id -eq $ContributionId} | Select-Object -First 1
  if (-not $item) { throw "Contribution '$ContributionId' is missing." }
  return $item
}

function Module-Session([string]$Token, [string]$ContributionId) {
  $item = Contribution $Token $ContributionId
  $body = @{moduleId=$item.moduleId;moduleVersion=$item.moduleVersion;contributionId=$item.contribution.id;surface='admin'} | ConvertTo-Json -Compress
  $response = Invoke-RestMethod -Method Post -Uri "$GatewayUrl/runtime/v1/module-sessions" -Headers @{Authorization="Bearer $Token";'X-Liveshop-Surface'='admin'} -ContentType 'application/json' -Body $body
  if (-not $response.data.token) { throw "Module session for '$ContributionId' was not issued." }
  return $response.data.token
}

# Realm is an authorization boundary, not a UI hint.
$platform = Login 'PLATFORM' 'admin'
$merchant = Login 'MERCHANT' 'merch@sufeipay.com'
Assert-HttpStatus { Invoke-RestMethod -Uri "$GatewayUrl/runtime/v1/contributions?surface=merch" -Headers @{Authorization="Bearer $($platform.Token)";'X-Liveshop-Surface'='merch'} } 403 'PLATFORM identity on Merchant surface'
Assert-HttpStatus { Invoke-RestMethod -Uri "$GatewayUrl/runtime/v1/contributions?surface=admin" -Headers @{Authorization="Bearer $($merchant.Token)";'X-Liveshop-Surface'='admin'} } 403 'MERCHANT identity on Platform surface'
Assert-HttpStatus {
  $body = @{realm='PLATFORM';username='admin';password='wrong-password'} | ConvertTo-Json -Compress
  Invoke-RestMethod -Method Post -Uri "$GatewayUrl/auth/login" -Headers @{'X-Liveshop-Surface'='admin'} -ContentType 'application/json' -Body $body
} 401 'Invalid password login'

# Refresh tokens rotate once. Reusing an old token revokes the whole session family.
$reuse = Login 'PLATFORM' 'admin'
$cookieUri = [Uri]"$GatewayUrl/auth/refresh"
$oldRefresh = $reuse.Session.Cookies.GetCookies($cookieUri)['liveshop_refresh'].Value
if (-not $oldRefresh) { throw 'HttpOnly refresh cookie was not issued.' }
$rotated = Invoke-RestMethod -Method Post -Uri "$GatewayUrl/auth/refresh" -WebSession $reuse.Session -Headers @{'X-Liveshop-Surface'='admin'}
if (-not $rotated.data.accessToken) { throw 'Refresh rotation failed.' }
Assert-HttpStatus { Invoke-RestMethod -Method Post -Uri "$GatewayUrl/auth/refresh" -Headers @{Cookie="liveshop_refresh=$oldRefresh";'X-Liveshop-Surface'='admin'} } 401 'Reused refresh token'
Assert-HttpStatus { Invoke-RestMethod -Method Post -Uri "$GatewayUrl/auth/refresh" -WebSession $reuse.Session -Headers @{'X-Liveshop-Surface'='admin'} } 401 'Revoked refresh-token family'

$registryToken = Module-Session $platform.Token 'platform.admin.registry'
$registryHeaders = @{Authorization="Bearer $registryToken";'X-Liveshop-Surface'='admin'}
$modules = Invoke-RestMethod -Uri "$GatewayUrl/admin/platform/registry/modules" -Headers $registryHeaders
if (@($modules.data).Count -lt 1) { throw 'Registry module list is empty.' }
Assert-HttpStatus { Invoke-RestMethod -Method Delete -Uri "$GatewayUrl/admin/platform/registry/modules/platform/activation" -Headers $registryHeaders } 403 'Platform self-deactivation'

$accountToken = Module-Session $platform.Token 'platform.admin.accounts'
$accountHeaders = @{Authorization="Bearer $accountToken";'X-Liveshop-Surface'='admin'}
$accounts = Invoke-RestMethod -Uri "$GatewayUrl/admin/platform/identity/accounts" -Headers $accountHeaders
$selfAccount = @($accounts.data) | Where-Object {$_.realm -eq 'PLATFORM' -and $_.subject -eq 'platform-admin'} | Select-Object -First 1
if (-not $selfAccount) { throw 'Seeded platform account is missing.' }
$selfDisableBody = @{expectedVersion=$selfAccount.version;username=$selfAccount.username;status='DISABLED';password=''} | ConvertTo-Json -Compress
Assert-HttpStatus { Invoke-RestMethod -Method Put -Uri "$GatewayUrl/admin/platform/identity/accounts/PLATFORM/platform-admin" -Headers $accountHeaders -ContentType 'application/json' -Body $selfDisableBody } 403 'Self account disable'

$smokeAccount = @($accounts.data) | Where-Object {$_.realm -eq 'PLATFORM' -and $_.subject -eq 'smoke-operator'} | Select-Object -First 1
$accountExpected = if ($smokeAccount) {[int64]$smokeAccount.version} else {0}
$accountBody = @{expectedVersion=$accountExpected;username='smoke-operator';status='ACTIVE';password='SmokeLocal!123'} | ConvertTo-Json -Compress
$accountUpdated = Invoke-RestMethod -Method Put -Uri "$GatewayUrl/admin/platform/identity/accounts/PLATFORM/smoke-operator" -Headers $accountHeaders -ContentType 'application/json' -Body $accountBody
$staleAccountBody = @{expectedVersion=$accountExpected;username='stale-name';status='ACTIVE';password=''} | ConvertTo-Json -Compress
Assert-HttpStatus { Invoke-RestMethod -Method Put -Uri "$GatewayUrl/admin/platform/identity/accounts/PLATFORM/smoke-operator" -Headers $accountHeaders -ContentType 'application/json' -Body $staleAccountBody } 409 'Stale account write'

$wrongSmokeLogin = @{realm='PLATFORM';username='smoke-operator';password='wrong-password'} | ConvertTo-Json -Compress
for ($attempt = 1; $attempt -le 4; $attempt++) {
  Assert-HttpStatus { Invoke-RestMethod -Method Post -Uri "$GatewayUrl/auth/login" -Headers @{'X-Liveshop-Surface'='admin'} -ContentType 'application/json' -Body $wrongSmokeLogin } 401 "Failed login $attempt"
}
Assert-HttpStatus { Invoke-RestMethod -Method Post -Uri "$GatewayUrl/auth/login" -Headers @{'X-Liveshop-Surface'='admin'} -ContentType 'application/json' -Body $wrongSmokeLogin } 429 'Fifth failed login lockout'
$correctSmokeLogin = @{realm='PLATFORM';username='smoke-operator';password='SmokeLocal!123'} | ConvertTo-Json -Compress
Assert-HttpStatus { Invoke-RestMethod -Method Post -Uri "$GatewayUrl/auth/login" -Headers @{'X-Liveshop-Surface'='admin'} -ContentType 'application/json' -Body $correctSmokeLogin } 429 'Correct password during lockout'

$unlockBody = @{expectedVersion=$accountUpdated.data.version;username='smoke-operator';status='ACTIVE';password='SmokeLocal!123'} | ConvertTo-Json -Compress
$unlocked = Invoke-RestMethod -Method Put -Uri "$GatewayUrl/admin/platform/identity/accounts/PLATFORM/smoke-operator" -Headers $accountHeaders -ContentType 'application/json' -Body $unlockBody
$smokeSession = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
$smokeLogin = Invoke-RestMethod -Method Post -Uri "$GatewayUrl/auth/login" -WebSession $smokeSession -Headers @{'X-Liveshop-Surface'='admin'} -ContentType 'application/json' -Body $correctSmokeLogin
if (-not $smokeLogin.data.accessToken) { throw 'Password reset did not unlock the smoke account.' }
$disableBody = @{expectedVersion=$unlocked.data.version;username='smoke-operator';status='DISABLED';password=''} | ConvertTo-Json -Compress
Invoke-RestMethod -Method Put -Uri "$GatewayUrl/admin/platform/identity/accounts/PLATFORM/smoke-operator" -Headers $accountHeaders -ContentType 'application/json' -Body $disableBody | Out-Null
Assert-HttpStatus { Invoke-RestMethod -Method Post -Uri "$GatewayUrl/auth/refresh" -WebSession $smokeSession -Headers @{'X-Liveshop-Surface'='admin'} } 401 'Disabled account refresh session'

$settingsToken = Module-Session $platform.Token 'platform.admin.settings'
$settingsHeaders = @{Authorization="Bearer $settingsToken";'X-Liveshop-Surface'='admin'}
$settings = Invoke-RestMethod -Uri "$GatewayUrl/admin/platform/settings" -Headers $settingsHeaders
$existing = @($settings.data) | Where-Object {$_.namespace -eq 'smoke-verification'} | Select-Object -First 1
$expectedVersion = if ($existing) {[int64]$existing.version} else {0}
$value = @{verifiedAt=[DateTime]::UtcNow.ToString('o');purpose='platform-control-smoke'}
$putBody = @{expectedVersion=$expectedVersion;value=$value} | ConvertTo-Json -Depth 5 -Compress
$updated = Invoke-RestMethod -Method Put -Uri "$GatewayUrl/admin/platform/settings/smoke-verification" -Headers $settingsHeaders -ContentType 'application/json' -Body $putBody
if ([int64]$updated.data.version -ne ($expectedVersion + 1)) { throw 'Settings optimistic version did not advance exactly once.' }
$staleBody = @{expectedVersion=$expectedVersion;value=@{purpose='stale-write-must-fail'}} | ConvertTo-Json -Depth 5 -Compress
Assert-HttpStatus { Invoke-RestMethod -Method Put -Uri "$GatewayUrl/admin/platform/settings/smoke-verification" -Headers $settingsHeaders -ContentType 'application/json' -Body $staleBody } 409 'Stale settings write'
$secretBody = @{expectedVersion=0;value=@{apiKey='must-not-be-stored'}} | ConvertTo-Json -Depth 5 -Compress
Assert-HttpStatus { Invoke-RestMethod -Method Put -Uri "$GatewayUrl/admin/platform/settings/smoke-secret-check" -Headers $settingsHeaders -ContentType 'application/json' -Body $secretBody } 400 'Secret-like platform setting'

$auditToken = Module-Session $platform.Token 'platform.admin.audit'
$audit = Invoke-RestMethod -Uri "$GatewayUrl/admin/platform/audit/events?limit=100" -Headers @{Authorization="Bearer $auditToken";'X-Liveshop-Surface'='admin'}
if (-not (@($audit.data) | Where-Object {$_.action -eq 'settings.update' -and $_.resourceId -eq 'smoke-verification'} | Select-Object -First 1)) {
  throw 'Atomic settings audit event is missing.'
}
if (-not (@($audit.data) | Where-Object {$_.action -eq 'identity.account.put' -and $_.resourceId -eq 'PLATFORM:smoke-operator'} | Select-Object -First 1)) {
  throw 'Atomic identity-account audit event is missing.'
}

Invoke-WebRequest -Method Post -Uri "$GatewayUrl/auth/logout" -WebSession $platform.Session -Headers @{'X-Liveshop-Surface'='admin'} | Out-Null
Write-Output "Platform control smoke passed: account lifecycle, login lockout, realm isolation, refresh revocation, registry protection, optimistic settings, secret rejection and audit."
