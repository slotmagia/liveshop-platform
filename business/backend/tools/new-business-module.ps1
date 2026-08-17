[CmdletBinding()]
param(
    [Parameter(Mandatory)] [string] $Destination,
    [Parameter(Mandatory)] [ValidatePattern('^[a-z][a-z0-9-]{1,62}$')] [string] $ModuleId,
    [Parameter(Mandatory)] [string] $ModuleName,
    [Parameter(Mandatory)] [ValidatePattern('^[^\s]+/[^\s]+$')] [string] $GoModule,
    # Comma separated so the value survives `pwsh -File` unchanged.
    [ValidatePattern('^(admin|merch|shop|live)(,(admin|merch|shop|live))*$')] [string] $Surfaces = 'merch'
)

# Thin wrapper. Both the generator and the module template live in liveshop-gui,
# which ships them as one executable; Platform keeps no copy of either so there
# is nothing here that can drift from what the tool actually produces.
$ErrorActionPreference = 'Stop'
$workspace = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..\..\..'))
$generator = Join-Path $workspace 'liveshop-gui'
$arguments = @(
    '-create'
    '-id', $ModuleId
    '-name', $ModuleName
    '-go-module', $GoModule
    '-surfaces', $Surfaces
    '-dest', [IO.Path]::GetFullPath($Destination)
)

$binary = Join-Path $generator 'dist/liveshop-gui-amd64.exe'
$sourceModule = Join-Path $generator 'go.mod'
if (Test-Path -LiteralPath $sourceModule -PathType Leaf) {
    $env:GOWORK = 'off'
    go -C $generator run ./cmd/liveshop-gui @arguments
}
elseif (Test-Path -LiteralPath $binary -PathType Leaf) {
    & $binary @arguments
}
else {
    throw "Module generator is missing: $generator (expected the liveshop-gui repository next to liveshop-platform)"
}
if ($LASTEXITCODE -ne 0) { throw 'Module generation failed.' }

Write-Output "Next: complete business/backend/docs/domain, then follow business/backend/docs/模块接入规范.md."
