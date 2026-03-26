$ErrorActionPreference = "Stop"

$sqlDir = Join-Path $PSScriptRoot "..\sql"
$files = Get-ChildItem -Path $sqlDir -File -Filter "*.sql" |
  Where-Object { $_.Name -match '^\d{3}_.+\.sql$' }

$allowedDuplicates = @("016", "017")
$groups = $files | Group-Object { ($_.Name -split '_')[0] } | Where-Object {
  $_.Count -gt 1 -and $allowedDuplicates -notcontains $_.Name
}

if ($groups.Count -eq 0) {
  Write-Host "OK: No new duplicate numeric migration prefixes found."
  exit 0
}

Write-Host "Found duplicate migration prefixes:" -ForegroundColor Yellow
foreach ($g in $groups) {
  Write-Host ("- " + $g.Name + ": " + (($g.Group | Select-Object -ExpandProperty Name) -join ", "))
}
exit 1
