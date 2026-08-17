$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
$run = Join-Path $root '.run'
$bin = Join-Path $run 'bin'
$vite = [IO.Path]::GetFullPath((Join-Path $root '..\liveshop-gateway\node_modules\vite\bin\vite.js'))
if (-not (Test-Path $run)) { return }
Get-ChildItem $run -Filter *.pid | ForEach-Object {
  $id = [int](Get-Content -Raw $_.FullName)
  $process = Get-CimInstance Win32_Process -Filter "ProcessId=$id" -ErrorAction SilentlyContinue
  $owned = $false
  if ($process -and $process.ExecutablePath) {
    $owned = $process.ExecutablePath.StartsWith($bin, [StringComparison]::OrdinalIgnoreCase) -or
      ($process.Name -eq 'node.exe' -and $process.CommandLine.IndexOf($vite, [StringComparison]::OrdinalIgnoreCase) -ge 0)
  }
  if ($owned) {
    & taskkill.exe /PID $id /T /F 2>$null | Out-Null
  } elseif ($process) {
    Write-Warning "Skipped stale PID $id from $($_.Name); it does not belong to this workspace."
  }
  Remove-Item -LiteralPath $_.FullName -Force
}
Remove-Item -LiteralPath (Join-Path $run 'dev-profile.json') -Force -ErrorAction SilentlyContinue
& docker compose -f (Join-Path $root 'backend/deploy/compose.local.yml') stop registry-db 2>$null | Out-Null
Write-Output 'Development processes stopped.'
