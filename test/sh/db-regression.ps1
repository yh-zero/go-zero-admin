#requires -Version 5.1
# Requires the development MySQL container. Only touches a new isolated database.
$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
$id = [guid]::NewGuid().ToString('N')
$database = 'gozero_test_' + $id
$testRoot = Join-Path $root ('bin/db-regression-' + $id)
$helper = '/tmp/gozero-db-test-' + $id + '.sh'
$baseline = '/tmp/gozero-db-test-' + $id + '.sql'
$utf8 = New-Object Text.UTF8Encoding($false)
function Docker([string[]]$Arguments) {
    $output = & docker.exe @Arguments
    if ($LASTEXITCODE -ne 0) { throw ('Docker failed: ' + ($Arguments -join ' ')) }
    return $output
}
function Query([string]$Sql) { Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'query', $Sql) }
function Migrate([bool]$ShouldSucceed = $true) {
    $ErrorActionPreference = 'Continue'
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $testRoot 'test/sh/db.ps1') -Action Migrate -Database $database 2>&1 | Out-Host
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = 'Stop'
    if (($exitCode -eq 0) -ne $ShouldSucceed) { throw 'Unexpected migration exit status.' }
}
function Expect([string]$Sql, [string]$Expected) {
    $actual = (Query $Sql | Out-String).Trim()
    if ($actual -cne $Expected) { throw "Expected $Expected, got $actual" }
}
$container = (Docker @('compose', '-f', (Join-Path $root 'docker-compose.yml'), 'ps', '-q', 'mysql') | Out-String).Trim()
if (-not $container) { throw 'Start development MySQL first.' }
New-Item -ItemType Directory -Path (Join-Path $testRoot 'test/sh'), (Join-Path $testRoot 'data/db/migrations') -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $root 'docker-compose.yml') -Destination $testRoot
Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'db.ps1') -Destination (Join-Path $testRoot 'test/sh')
[IO.File]::WriteAllText((Join-Path $testRoot 'test/sh/mysql-client.sh'), [IO.File]::ReadAllText((Join-Path $PSScriptRoot 'mysql-client.sh')).Replace("`r`n", "`n"), $utf8)
Get-ChildItem -LiteralPath (Join-Path $root 'data/db/migrations') -Filter '*.sql' | Copy-Item -Destination (Join-Path $testRoot 'data/db/migrations')
$created = $false
try {
    Docker @('cp', (Join-Path $testRoot 'test/sh/mysql-client.sh'), ($container + ':' + $helper)) | Out-Null
    Docker @('exec', $container, 'sh', $helper, 'query', "CREATE DATABASE $database CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;") | Out-Null
    $created = $true
    Docker @('cp', (Join-Path $root 'data/db/gozero-admin-20240129.sql'), ($container + ':' + $baseline)) | Out-Null
    Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'run', $baseline) | Out-Null
    Migrate
    $count = @(Get-ChildItem -LiteralPath (Join-Path $testRoot 'data/db/migrations') -Filter '*.sql').Count
    Expect 'SELECT COUNT(*) FROM schema_migrations;' ([string]$count)
    Migrate
    Expect 'SELECT COUNT(*) FROM schema_migrations;' ([string]$count)
    Expect "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND non_unique=0 AND index_name IN ('uk_sys_menus_active_name','uk_sys_menus_active_path','uk_sys_apis_active_route','uk_sys_dictionaries_active_type','uk_sys_dictionary_info_active_value') AND seq_in_index=1;" '5'
    Expect "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='sys_users' AND column_name='session_version';" '1'
    # The failing batch must stop before later statements and must not be recorded.
    $probe = Join-Path $testRoot 'data/db/migrations/99991230_probe.sql'
    [IO.File]::WriteAllText($probe, "SELECT * FROM intentionally_missing_regression_table;`nINSERT INTO schema_migrations(filename,checksum) VALUES('must_not_run','bad');`n", $utf8)
    Migrate $false
    Expect "SELECT COUNT(*) FROM schema_migrations WHERE filename IN ('99991230_probe.sql','must_not_run');" '0'
    [IO.File]::WriteAllText($probe, "SELECT 1;`n", $utf8)
    Migrate
    Expect "SELECT COUNT(*) FROM schema_migrations WHERE filename='99991230_probe.sql';" '1'
    [IO.File]::WriteAllText($probe, "SELECT 2;`n", $utf8)
    Migrate $false
    Write-Host 'PASS: fresh install, repeat migration, schema checks, SQL failure stop, retry, checksum guard.'
} finally {
    if ($created) { Docker @('exec', $container, 'sh', $helper, 'query', "DROP DATABASE $database;") | Out-Null }
    Docker @('exec', $container, 'rm', '-f', '--', $helper, $baseline) | Out-Null
    # Keep the test copies and backups under ignored bin/ for failure inspection.
}
