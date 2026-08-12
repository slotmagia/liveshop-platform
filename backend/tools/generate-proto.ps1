$ErrorActionPreference = 'Stop'
$contractsRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\contracts'))

go -C $contractsRoot run github.com/bufbuild/buf/cmd/buf@v1.47.2 lint
if ($LASTEXITCODE -ne 0) { throw 'Platform Proto lint failed.' }

go -C $contractsRoot run github.com/bufbuild/buf/cmd/buf@v1.47.2 generate
if ($LASTEXITCODE -ne 0) { throw 'Platform Proto generation failed.' }

Write-Output 'Platform Proto contracts generated.'
