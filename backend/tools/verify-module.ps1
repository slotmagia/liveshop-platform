[CmdletBinding()]
param(
    [int] $MinimumCoverage = 30
)

$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
$backend = Join-Path $root 'backend'
& (Join-Path $PSScriptRoot 'verify-fast.ps1')

go -C $backend test -race ./...
if ($LASTEXITCODE -ne 0) { throw 'race tests failed.' }
$coverage = Join-Path $backend '.coverage.out'
try {
    $testedPackages = @(go -C $backend list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' ./... | Where-Object { $_ })
    if ($LASTEXITCODE -ne 0 -or $testedPackages.Count -eq 0) { throw 'Unable to enumerate packages with tests.' }
    go -C $backend test "-coverprofile=$coverage" $testedPackages
    if ($LASTEXITCODE -ne 0) { throw 'coverage tests failed.' }
    $total = go -C $backend tool cover "-func=$coverage" | Select-Object -Last 1
    if ($total -notmatch '([0-9]+(?:\.[0-9]+)?)%') { throw 'Unable to read total coverage.' }
    if ([double]$Matches[1] -lt $MinimumCoverage) { throw "Coverage $($Matches[1])% is below $MinimumCoverage%." }
} finally {
    if (Test-Path -LiteralPath $coverage) { Remove-Item -LiteralPath $coverage }
}

if ($env:PLATFORM_TEST_DATABASE_URL) {
    go -C $backend test -tags=integration ./internal/platform/registry/...
    if ($LASTEXITCODE -ne 0) { throw 'PostgreSQL integration tests failed.' }
} else {
    Write-Warning 'PLATFORM_TEST_DATABASE_URL is not set; PostgreSQL integration tests were not executed.'
}
Write-Output 'Module verification passed.'
