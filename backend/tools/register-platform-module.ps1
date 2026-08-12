param(
  [string]$PlatformUrl = 'http://127.0.0.1:8082',
  [string]$BackendOrigin = '',
  [string]$GRPCEndpoint = 'dns:///platform:9082',
  [string]$ArtifactUrl = 'http://127.0.0.1:5180'
)
$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
$resolvedBackendOrigin = if ($BackendOrigin) { $BackendOrigin } else { $PlatformUrl }
$manifest = Get-Content -Raw -LiteralPath (Join-Path $root 'module.json')
$manifest = $manifest.Replace('http://127.0.0.1:8082', $resolvedBackendOrigin).Replace('dns:///platform:9082', $GRPCEndpoint).Replace('http://127.0.0.1:5180', $ArtifactUrl)
$artifact = Invoke-WebRequest -Uri $ArtifactUrl -TimeoutSec 10
$bytes = [Text.Encoding]::UTF8.GetBytes([string]$artifact.Content)
$digest = 'sha256:' + [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($bytes)).ToLowerInvariant()
$manifest = $manifest.Replace('sha256:dev-platform-iam', $digest)
$version = ($manifest | ConvertFrom-Json).metadata.version
$env:WORKLOAD_PRIVATE_KEY = 'MEdxJQh5ZzEe9NhL8TQ7G5rCqZ1Cr00n6DVMiCayO_8'
$env:WORKLOAD_KEY_ID = 'ci-workload-dev-1'
$env:WORKLOAD_ISSUER = 'liveshop-workload-identity'
$env:WORKLOAD_SUBJECT = 'module-release-ci'
$env:WORKLOAD_AUDIENCE = 'liveshop-platform-internal'
$kernelRoot = [IO.Path]::GetFullPath((Join-Path $root '..\kernel-go'))
$token = & go -C $kernelRoot run ./cmd/workloadtoken
if ($LASTEXITCODE -ne 0 -or -not $token) { throw 'Failed to issue platform module CI identity.' }
$headers = @{Authorization="Bearer $token"}
$release = Invoke-RestMethod -Method Post -Uri "$PlatformUrl/internal/v1/module-registry/releases" -Headers $headers -ContentType 'application/json' -Body $manifest
$body = @{moduleId='platform';version=$version} | ConvertTo-Json -Compress
$activation = Invoke-RestMethod -Method Post -Uri "$PlatformUrl/internal/v1/module-registry/activate" -Headers $headers -ContentType 'application/json' -Body $body
if ($release.code -ne 0 -or $activation.code -ne 0) { throw 'Platform IAM module registration failed.' }
Write-Output "Platform IAM $version registered and activated. digest=$($release.data.digest)"
