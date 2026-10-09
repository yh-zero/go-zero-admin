#requires -Version 5.1
# Import only into a newly created, isolated database in the development MySQL.
[CmdletBinding()]
param([string]$SqlFile = '')
$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
if (-not $SqlFile) { $SqlFile = Join-Path $root 'data/db/gozero-admin.sql' }
if (-not (Test-Path -LiteralPath $SqlFile -PathType Leaf)) { throw 'The current full installation SQL is missing.' }
$sqlText = [IO.File]::ReadAllText($SqlFile)
if ($sqlText -match '(?im)^\s*(CREATE\s+DATABASE|USE\s+)') { throw 'The installation SQL must use the explicitly selected database.' }
$id = [guid]::NewGuid().ToString('N')
$database = 'gozero_import_test_' + $id
$helper = '/tmp/gozero-import-test-' + $id + '.sh'
$remoteSql = '/tmp/gozero-import-test-' + $id + '.sql'
$helperLocal = Join-Path ([IO.Path]::GetTempPath()) ('gozero-import-test-' + $id + '.sh')
$created = $false
function Docker([string[]]$Arguments) {
    $output = & docker.exe @Arguments
    if ($LASTEXITCODE -ne 0) { throw 'Docker command failed during the isolated import test.' }
    return $output
}
function Query([string]$Sql) { Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'query', $Sql) }
function Expect([string]$Sql, [string]$Expected) {
    $actual = (Query $Sql | Out-String).Trim()
    if ($actual -cne $Expected) { throw ('Expected ' + $Expected + ', got ' + $actual) }
}
$container = (Docker @('compose', '-f', (Join-Path $root 'docker-compose.yml'), 'ps', '-q', 'mysql') | Out-String).Trim()
if (-not $container) { throw 'Start development MySQL before running the import regression.' }
try {
    [IO.File]::WriteAllText($helperLocal, [IO.File]::ReadAllText((Join-Path $PSScriptRoot 'mysql-client.sh')).Replace("`r`n", "`n"), (New-Object Text.UTF8Encoding($false)))
    Docker @('cp', $helperLocal, ($container + ':' + $helper)) | Out-Null
    Docker @('exec', $container, 'sh', $helper, 'query', ('CREATE DATABASE ' + $database + ' CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;')) | Out-Null
    $created = $true
    Docker @('cp', $SqlFile, ($container + ':' + $remoteSql)) | Out-Null
    # Exercise the documented PowerShell -> Docker -> shell import path.
    Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', '-c', 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --user=root --database="$MYSQL_DATABASE" < "$1"', 'sh', $remoteSql) | Out-Null
    Expect 'SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE();' '30'
    Expect "SELECT COUNT(*) FROM sys_users WHERE id=1 AND username='admin' AND authority_id=1 AND enable=1 AND deleted_at IS NULL;" '1'
    Expect 'SELECT COUNT(*) FROM sys_users;' '1'
    Expect 'SELECT COUNT(*) FROM sys_user_authority WHERE sys_user_id=1 AND sys_authority_authority_id=1;' '1'
    Expect 'SELECT (SELECT COUNT(*) FROM sys_device_sessions)+(SELECT COUNT(*) FROM sys_audit_logs)+(SELECT COUNT(*) FROM sys_ai_runs)+(SELECT COUNT(*) FROM sys_ai_messages)+(SELECT COUNT(*) FROM sys_ai_conversations)+(SELECT COUNT(*) FROM sys_file_resources)+(SELECT COUNT(*) FROM sys_file_references);' '0'
    Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name IN ('audit','files','sessions','organization-departments','organization-positions','ai-agent') AND deleted_at IS NULL;" '6'
    Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name IN ('test1','test2','test3','test4','test5');" '0'
    Expect 'SELECT COUNT(*) FROM sys_authority_menus grant_row LEFT JOIN sys_base_menus menu ON menu.id=grant_row.sys_base_menu_id WHERE menu.id IS NULL;' '0'
    Expect 'SELECT COUNT(*) FROM sys_authorities;' '1'
    Expect 'SELECT COUNT(*) FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL;' '1'
    Expect 'SELECT COUNT(*) FROM schema_migrations;' '16'
    foreach ($migration in @(Get-ChildItem -LiteralPath (Join-Path $root 'data/db/migrations') -Filter '*.sql')) {
        $normalized = [IO.File]::ReadAllText($migration.FullName).Replace("`r`n", "`n")
        $algorithm = [Security.Cryptography.SHA256]::Create()
        try { $hash = ([BitConverter]::ToString($algorithm.ComputeHash([Text.Encoding]::UTF8.GetBytes($normalized)))).Replace('-', '').ToLowerInvariant() } finally { $algorithm.Dispose() }
        Expect ("SELECT COUNT(*) FROM schema_migrations WHERE filename='" + $migration.Name + "' AND checksum='" + $hash + "';") '1'
    }
    $status = @(& powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot 'db.ps1') -Action Status -Database $database 2>&1)
    if ($LASTEXITCODE -ne 0 -or @($status | Where-Object { [string]$_ -match '^PENDING ' }).Count -gt 0) { throw 'A fresh SQL import must be ready without additional migrations.' }
    Query "INSERT INTO sys_audit_logs(event_type,module,action,result,status_code,params) VALUES('operation','import-test','preserveExistingRows','success',200,'{}');" | Out-Null
    $ErrorActionPreference = 'Continue'
    & docker.exe exec -e ('MYSQL_DATABASE=' + $database) $container sh $helper run $remoteSql 2>&1 | Out-Null
    $repeatExit = $LASTEXITCODE
    $ErrorActionPreference = 'Stop'
    if ($repeatExit -eq 0) { throw 'Importing over an existing database must fail instead of resetting it.' }
    Expect "SELECT COUNT(*) FROM sys_audit_logs WHERE action='preserveExistingRows';" '1'
    Expect 'SELECT COUNT(*) FROM sys_users;' '1'
    Expect 'SELECT COUNT(*) FROM schema_migrations;' '16'
    Write-Host 'PASS: current SQL imports into a custom empty database, is ready for startup, and preserves existing data on repeat import.'
} finally {
    if ($created) { Query ('DROP DATABASE ' + $database + ';') | Out-Null }
    if ($container) { & docker.exe exec $container rm -f -- $helper $remoteSql | Out-Null }
    if (Test-Path -LiteralPath $helperLocal) { Remove-Item -LiteralPath $helperLocal -Force }
}
