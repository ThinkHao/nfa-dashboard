$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$guardScript = Join-Path $PSScriptRoot "sql_migration_guard.py"

if (-not (Test-Path $guardScript)) {
  throw "Missing guard script: $guardScript"
}

python $guardScript --repo-root $repoRoot check
if ($LASTEXITCODE -ne 0) {
  exit $LASTEXITCODE
}
