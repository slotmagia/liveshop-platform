[CmdletBinding()]
param(
    [switch] $SkipContainer
)

$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
$backend = Join-Path $root 'backend'
$contracts = Join-Path $backend 'contracts'
& (Join-Path $PSScriptRoot 'verify-module.ps1')

go -C $backend run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
if ($LASTEXITCODE -ne 0) { throw 'Go vulnerability scan failed.' }
Push-Location $root
try { npm audit --audit-level=high; if ($LASTEXITCODE -ne 0) { throw 'npm audit failed.' } } finally { Pop-Location }

$baseline = Join-Path $contracts 'baseline/platform-v1.binpb'
if (-not (Test-Path -LiteralPath $baseline)) { throw "Buf breaking baseline is missing: $baseline" }
go -C $contracts run github.com/bufbuild/buf/cmd/buf@v1.47.2 breaking --against $baseline
if ($LASTEXITCODE -ne 0) { throw 'Proto breaking change detected.' }

if (-not $SkipContainer) {
    docker version | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Docker is required for release verification.' }
    $kernel = [IO.Path]::GetFullPath((Join-Path $root '..\kernel-go'))
    docker build --build-context "kernel=$kernel" -f (Join-Path $backend 'deploy/Dockerfile') $backend
    if ($LASTEXITCODE -ne 0) { throw 'Platform container build failed.' }
}
Write-Output 'Release verification passed.'
