[CmdletBinding()]
param(
    [Parameter(Mandatory)] [string] $Destination,
    [Parameter(Mandatory)] [ValidatePattern('^[a-z][a-z0-9-]{1,62}$')] [string] $ModuleId,
    [Parameter(Mandatory)] [string] $ModuleName,
    [Parameter(Mandatory)] [ValidatePattern('^[^\s]+/[^\s]+$')] [string] $GoModule
)

$ErrorActionPreference = 'Stop'
$templateRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot 'templates/business-module'))
$destinationRoot = [IO.Path]::GetFullPath($Destination)
$packageName = $ModuleId.Replace('-', '')
$typeName = (($ModuleId -split '-') | ForEach-Object {
    [Globalization.CultureInfo]::InvariantCulture.TextInfo.ToTitleCase($_)
}) -join ''

if (-not (Test-Path -LiteralPath $templateRoot -PathType Container)) {
    throw "Business module template is missing: $templateRoot"
}
if (Test-Path -LiteralPath $destinationRoot) {
    if (Get-ChildItem -LiteralPath $destinationRoot -Force | Select-Object -First 1) {
        throw "Destination must not exist or must be empty: $destinationRoot"
    }
} else {
    New-Item -ItemType Directory -Path $destinationRoot | Out-Null
}

$replacements = [ordered]@{
    '{{MODULE_ID}}' = $ModuleId
    '{{MODULE_NAME}}' = $ModuleName
    '{{MODULE_PACKAGE}}' = $packageName
    '{{MODULE_TYPE}}' = $typeName
    '{{GO_MODULE}}' = $GoModule
}

Get-ChildItem -LiteralPath $templateRoot -Recurse -File | ForEach-Object {
    $relative = [IO.Path]::GetRelativePath($templateRoot, $_.FullName)
    foreach ($entry in $replacements.GetEnumerator()) {
        $relative = $relative.Replace($entry.Key, $entry.Value)
    }
    if ($relative.EndsWith('.tmpl', [StringComparison]::Ordinal)) {
        $relative = $relative.Substring(0, $relative.Length - 5)
    }
    $target = Join-Path $destinationRoot $relative
    $targetDirectory = Split-Path -Parent $target
    New-Item -ItemType Directory -Path $targetDirectory -Force | Out-Null
    $content = [IO.File]::ReadAllText($_.FullName)
    foreach ($entry in $replacements.GetEnumerator()) {
        $content = $content.Replace($entry.Key, $entry.Value)
    }
    [IO.File]::WriteAllText($target, $content, [Text.UTF8Encoding]::new($false))
}

Write-Output "Created business module '$ModuleId' at $destinationRoot"
Write-Output "Next: complete backend/docs/domain, define contracts, then run backend/tools/verify-fast.ps1."
