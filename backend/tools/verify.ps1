[CmdletBinding()]
param(
    [switch] $SkipContainer
)

$ErrorActionPreference = 'Stop'
& (Join-Path $PSScriptRoot 'verify-release.ps1') -SkipContainer:$SkipContainer
