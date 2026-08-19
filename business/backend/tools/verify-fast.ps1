[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
$backend = Join-Path $root 'backend'
# Wire contracts live in the sibling protocol module, which is published on its
# own so consumers never depend on this module's implementation.
$protocol = [IO.Path]::GetFullPath((Join-Path $root '..\..\liveshop-protocol\platform'))

$goFiles = @(rg --files $backend $protocol -g '*.go' -g '!gen/**')
if ($LASTEXITCODE -ne 0) { throw 'Unable to enumerate Go sources.' }
$unformatted = @($goFiles | ForEach-Object { gofmt -l $_ })
if ($unformatted.Count -gt 0) { throw "Go files require gofmt:`n$($unformatted -join "`n")" }

go -C $backend vet ./...
if ($LASTEXITCODE -ne 0) { throw 'go vet failed.' }
go -C $protocol test ./...
if ($LASTEXITCODE -ne 0) { throw 'protocol tests failed.' }
go -C $backend test ./...
if ($LASTEXITCODE -ne 0) { throw 'backend tests failed.' }

go -C $backend run ./cmd/archcheck -root $root -protocol $protocol
if ($LASTEXITCODE -ne 0) { throw 'architecture checks failed.' }
go -C $protocol run ./cmd/manifestcompose -mode check -input (Join-Path $root 'module.json') -source (Join-Path $protocol 'manifest/platform')
if ($LASTEXITCODE -ne 0) { throw 'manifest composition drift detected.' }
go -C $protocol run ./cmd/manifestcheck (Join-Path $root 'module.json')
if ($LASTEXITCODE -ne 0) { throw 'module manifest validation failed.' }

go -C $protocol run github.com/bufbuild/buf/cmd/buf@v1.47.2 lint
if ($LASTEXITCODE -ne 0) { throw 'Buf lint failed.' }
$generatedRoot = Join-Path $protocol 'gen/go'
$before = @{}
Get-ChildItem -LiteralPath $generatedRoot -Recurse -File | ForEach-Object { $before[$_.FullName] = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash }
& (Join-Path $protocol 'tools/generate-proto.ps1')
$after = @{}
Get-ChildItem -LiteralPath $generatedRoot -Recurse -File | ForEach-Object { $after[$_.FullName] = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash }
if (($before.Keys.Count -ne $after.Keys.Count) -or ($before.Keys | Where-Object { -not $after.ContainsKey($_) -or $after[$_] -ne $before[$_] })) {
    throw 'Generated Proto files drifted; commit the output of protocol/tools/generate-proto.ps1.'
}

Push-Location $root
try { npm run build; if ($LASTEXITCODE -ne 0) { throw 'frontend build failed.' } } finally { Pop-Location }
Write-Output 'Fast verification passed.'
