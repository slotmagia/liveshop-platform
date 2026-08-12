[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
$backend = Join-Path $root 'backend'
$contracts = Join-Path $backend 'contracts'

$goFiles = @(rg --files $backend -g '*.go' -g '!contracts/gen/**')
if ($LASTEXITCODE -ne 0) { throw 'Unable to enumerate Go sources.' }
$unformatted = @($goFiles | ForEach-Object { gofmt -l $_ })
if ($unformatted.Count -gt 0) { throw "Go files require gofmt:`n$($unformatted -join "`n")" }

go -C $backend vet ./...
if ($LASTEXITCODE -ne 0) { throw 'go vet failed.' }
go -C $contracts test ./...
if ($LASTEXITCODE -ne 0) { throw 'contracts tests failed.' }
go -C $backend test ./...
if ($LASTEXITCODE -ne 0) { throw 'backend tests failed.' }

go -C $backend run ./cmd/archcheck -root $root
if ($LASTEXITCODE -ne 0) { throw 'architecture checks failed.' }
go -C $contracts run ./cmd/manifestcompose -mode check -input (Join-Path $root 'module.json') -source (Join-Path $contracts 'manifest/platform')
if ($LASTEXITCODE -ne 0) { throw 'manifest composition drift detected.' }
go -C $contracts run ./cmd/manifestcheck (Join-Path $root 'module.json')
if ($LASTEXITCODE -ne 0) { throw 'module manifest validation failed.' }

go -C $contracts run github.com/bufbuild/buf/cmd/buf@v1.47.2 lint
if ($LASTEXITCODE -ne 0) { throw 'Buf lint failed.' }
$generatedRoot = Join-Path $contracts 'gen/go'
$before = @{}
Get-ChildItem -LiteralPath $generatedRoot -Recurse -File | ForEach-Object { $before[$_.FullName] = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash }
& (Join-Path $PSScriptRoot 'generate-proto.ps1')
$after = @{}
Get-ChildItem -LiteralPath $generatedRoot -Recurse -File | ForEach-Object { $after[$_.FullName] = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash }
if (($before.Keys.Count -ne $after.Keys.Count) -or ($before.Keys | Where-Object { -not $after.ContainsKey($_) -or $after[$_] -ne $before[$_] })) {
    throw 'Generated Proto files drifted; commit the output of backend/tools/generate-proto.ps1.'
}

Push-Location $root
try { npm run build; if ($LASTEXITCODE -ne 0) { throw 'frontend build failed.' } } finally { Pop-Location }
Write-Output 'Fast verification passed.'
