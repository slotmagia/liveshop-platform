param(
  [string]$PlatformUrl = 'http://127.0.0.1:18082',
  [string]$BackendOrigin = '',
  [string]$GRPCEndpoint = 'dns:///platform:19082',
  [string]$ArtifactUrl = 'http://127.0.0.1:15180'
)
$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
$resolvedBackendOrigin = if ($BackendOrigin) { $BackendOrigin } else { $PlatformUrl }
# Windows PowerShell 5.1 defaults to the system ANSI code page. Reading a UTF-8
# module.json that way turns Chinese titles into "????" before they ever reach
# MySQL — force UTF-8 on the way in and on the way out.
$manifestPath = Join-Path $root 'module.json'
$manifest = [IO.File]::ReadAllText($manifestPath, [Text.Encoding]::UTF8)
$manifest = $manifest.Replace('http://127.0.0.1:18082', $resolvedBackendOrigin).Replace('dns:///platform:19082', $GRPCEndpoint).Replace('http://127.0.0.1:15180', $ArtifactUrl)
$artifact = Invoke-WebRequest -Uri $ArtifactUrl -TimeoutSec 10
$bytes = [Text.Encoding]::UTF8.GetBytes([string]$artifact.Content)
# SHA256.HashData and Convert.ToHexString need PowerShell 7; every other script
# here runs on Windows PowerShell 5.1, so this one must not be the exception.
$sha256 = [Security.Cryptography.SHA256]::Create()
try { $hash = $sha256.ComputeHash($bytes) } finally { $sha256.Dispose() }
$digest = 'sha256:' + (($hash | ForEach-Object { $_.ToString('x2') }) -join '')
$manifest = $manifest.Replace('sha256:dev-platform-control', $digest)
$manifestObject = $manifest | ConvertFrom-Json
$env:WORKLOAD_PRIVATE_KEY = 'MEdxJQh5ZzEe9NhL8TQ7G5rCqZ1Cr00n6DVMiCayO_8'
$env:WORKLOAD_KEY_ID = 'ci-workload-dev-1'
$env:WORKLOAD_ISSUER = 'liveshop-workload-identity'
$env:WORKLOAD_SUBJECT = 'module-release-ci'
$env:WORKLOAD_AUDIENCE = 'liveshop-platform-internal'
$kernelRoot = [IO.Path]::GetFullPath((Join-Path $root '..\..\kernel-go'))
$token = & go -C $kernelRoot run ./cmd/workloadtoken
if ($LASTEXITCODE -ne 0 -or -not $token) { throw 'Failed to issue platform module CI identity.' }
$headers = @{Authorization="Bearer $token"}
. (Join-Path $PSScriptRoot '..\..\..\..\register-local-release.ps1')
Publish-LocalModuleRelease -PlatformUrl $PlatformUrl -ModuleId 'platform' -Manifest $manifestObject -Headers $headers
