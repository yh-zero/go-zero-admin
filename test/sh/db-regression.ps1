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
$compatName = '20261003_frontend_routes_compat.sql'
Get-ChildItem -LiteralPath (Join-Path $root 'data/db/migrations') -Filter '*.sql' | Where-Object { $_.Name -ne $compatName } | Copy-Item -Destination (Join-Path $testRoot 'data/db/migrations')
$created = $false
try {
    Docker @('cp', (Join-Path $testRoot 'test/sh/mysql-client.sh'), ($container + ':' + $helper)) | Out-Null
    Docker @('exec', $container, 'sh', $helper, 'query', "CREATE DATABASE $database CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;") | Out-Null
    $created = $true
    Docker @('cp', (Join-Path $root 'data/db/gozero-admin-20240129.sql'), ($container + ':' + $baseline)) | Out-Null
    Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'run', $baseline) | Out-Null
    # Reproduce the old 13-migration bug: the legacy seed misses internal //.
    Query "INSERT INTO sys_base_menus (created_at,updated_at,parent_id,path,name,component,title,hidden) VALUES(NOW(3),NOW(3),0,'/admin//audit','custom-double-audit','views/custom/audit.vue','Custom double audit',0); INSERT INTO sys_authority_menus(sys_base_menu_id,sys_authority_authority_id) SELECT id,1 FROM sys_base_menus WHERE name='custom-double-audit';" | Out-Null
    Migrate
    $count = @(Get-ChildItem -LiteralPath (Join-Path $testRoot 'data/db/migrations') -Filter '*.sql').Count
    Expect 'SELECT COUNT(*) FROM schema_migrations;' ([string]$count)
    Migrate
    Expect 'SELECT COUNT(*) FROM schema_migrations;' ([string]$count)
    Expect "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND non_unique=0 AND index_name IN ('uk_sys_menus_active_name','uk_sys_menus_active_path','uk_sys_apis_active_route','uk_sys_dictionaries_active_type','uk_sys_dictionary_info_active_value') AND seq_in_index=1;" '5'
    Expect "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='sys_users' AND column_name='session_version';" '1'
    Expect "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name IN ('sys_policy_versions','sys_audit_logs','sys_departments','sys_positions','sys_user_departments','sys_user_positions','sys_role_data_scopes','sys_role_scope_departments','sys_file_resources','sys_file_references','sys_device_sessions');" '11'
    Expect "SELECT COUNT(*) FROM sys_apis WHERE deleted_at IS NULL AND (path LIKE '/v1/sys/organization/%' OR path LIKE '/v1/sys/files/%' OR path LIKE '/v1/sys/session/admin/%' OR path='/v1/sys/audit/getAuditLogList');" '20'
    Expect "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0='1' AND (v1 LIKE '/v1/sys/organization/%' OR v1 LIKE '/v1/sys/files/%' OR v1 LIKE '/v1/sys/session/admin/%' OR v1='/v1/sys/audit/getAuditLogList');" '20'
    Expect "SELECT COUNT(*) FROM sys_policy_versions WHERE id=1 AND version>0;" '1'
    Expect "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND ((table_name='sys_file_resources' AND column_name='object_key') OR (table_name='sys_file_references' AND column_name IN ('object_type','object_id'))) AND collation_name='utf8mb4_bin';" '3'
    Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name IN ('audit','files','sessions','organization-departments','organization-positions') AND deleted_at IS NULL;" '5'
    Expect "SELECT COUNT(*) FROM sys_authority_menus granted JOIN sys_base_menus menu ON menu.id=granted.sys_base_menu_id WHERE granted.sys_authority_authority_id=1 AND menu.name IN ('superAdmin','audit','files','sessions','organization-departments','organization-positions') AND menu.deleted_at IS NULL;" '6'
    Expect "SELECT COUNT(*) FROM sys_authority_btns granted JOIN sys_base_menu_btns button ON button.id=granted.sys_base_menu_btn_id JOIN sys_base_menus menu ON menu.id=granted.sys_menu_id WHERE granted.authority_id=1 AND menu.deleted_at IS NULL AND button.deleted_at IS NULL AND (menu.name IN ('files','sessions','organization-departments','organization-positions') OR (menu.name='user' AND button.name='membership') OR (menu.name='authority' AND button.name='dataScope'));" '12'
    $canonicalAudit = "SELECT COUNT(*) FROM sys_base_menus menu LEFT JOIN sys_base_menus parent ON parent.id=menu.parent_id WHERE menu.deleted_at IS NULL AND LOWER(TRIM(TRAILING '/' FROM REGEXP_REPLACE(IF(LEFT(menu.path,1)='/',menu.path,CONCAT('/',parent.path,'/',menu.path)), '/{2,}', '/')))='/admin/audit';"
    Expect $canonicalAudit '2'
    # Also occupy the first fallback through a canonical alias. The repair must
    # detect it and choose its next candidate without changing either custom row.
    Query "INSERT INTO sys_base_menus (created_at,updated_at,parent_id,path,name,component,title,hidden) SELECT NOW(3),NOW(3),0,CONCAT('/admin//audit-module-',id),'custom-fallback-audit','views/custom/audit.vue','Custom fallback audit',0 FROM sys_base_menus WHERE name='audit' AND deleted_at IS NULL;" | Out-Null
    Copy-Item -LiteralPath (Join-Path $root ('data/db/migrations/' + $compatName)) -Destination (Join-Path $testRoot 'data/db/migrations')
    Migrate
    $count++
    Expect 'SELECT COUNT(*) FROM schema_migrations;' ([string]$count)
    Expect $canonicalAudit '1'
    Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='audit' AND path=CONCAT('audit-module-',id,'-1') AND component='views/business/system/audit/index.vue' AND deleted_at IS NULL;" '1'
    Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='custom-double-audit' AND path='/admin//audit' AND component='views/custom/audit.vue' AND title='Custom double audit' AND deleted_at IS NULL;" '1'
    Expect "SELECT COUNT(*) FROM sys_authority_menus granted JOIN sys_base_menus menu ON menu.id=granted.sys_base_menu_id WHERE granted.sys_authority_authority_id=1 AND menu.name IN ('audit','custom-double-audit') AND menu.deleted_at IS NULL;" '2'
    Expect "SELECT COUNT(*) FROM sys_base_menus WHERE deleted_at IS NULL AND ((name='files' AND path='files') OR (name='sessions' AND path='sessions') OR (name='organization-departments' AND path='organization/departments') OR (name='organization-positions' AND path='organization/positions'));" '4'
    $compatMigration = '/tmp/gozero-frontend-route-compat-' + $id + '.sql'
    Docker @('cp', (Join-Path $root ('data/db/migrations/' + $compatName)), ($container + ':' + $compatMigration)) | Out-Null
    try {
        Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'run', $compatMigration) | Out-Host
        Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='audit' AND path=CONCAT('audit-module-',id,'-1') AND deleted_at IS NULL;" '1'
        Expect "SELECT COUNT(*) FROM sys_audit_logs WHERE action='repairFrontendRoute';" '1'
        # A customized seed remains untouched, with an explicit SKIPPED report.
        Query "UPDATE sys_base_menus SET title='Custom sessions' WHERE name='sessions' AND deleted_at IS NULL; INSERT INTO sys_base_menus (created_at,updated_at,parent_id,path,name,component,title,hidden) VALUES(NOW(3),NOW(3),0,'/admin//sessions','custom-double-sessions','views/custom/sessions.vue','Custom sessions route',0);" | Out-Null
        $compatOutput = @(Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'run', $compatMigration))
        $compatOutput | Out-Host
        if (-not @($compatOutput | Where-Object { $_ -match '^sessions\s+SKIPPED:' }).Count) { throw 'Customized conflict was not explicitly reported as SKIPPED.' }
        Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='sessions' AND path='sessions' AND title='Custom sessions' AND deleted_at IS NULL;" '1'
        Expect "SELECT COUNT(*) FROM sys_authority_btns granted JOIN sys_base_menu_btns button ON button.id=granted.sys_base_menu_btn_id JOIN sys_base_menus menu ON menu.id=granted.sys_menu_id WHERE granted.authority_id=1 AND menu.name='sessions' AND button.name='revoke' AND button.deleted_at IS NULL;" '1'
        Expect "SELECT COUNT(*) FROM sys_audit_logs WHERE action='repairFrontendRoute';" '1'
    } finally { Docker @('exec', $container, 'rm', '-f', '--', $compatMigration) | Out-Null }
    $menuMigration = '/tmp/gozero-frontend-modules-' + $id + '.sql'
    Docker @('cp', (Join-Path $root 'data/db/migrations/20261003_frontend_modules.sql'), ($container + ':' + $menuMigration)) | Out-Null
    try {
        # Re-running the actual SQL must not add duplicates or overwrite customization.
        Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'run', $menuMigration) | Out-Host
        Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name IN ('audit','files','sessions','organization-departments','organization-positions') AND deleted_at IS NULL;" '5'
        Expect "SELECT COUNT(*) FROM sys_audit_logs WHERE action='seedFrontendModules' AND object='20261003_frontend_modules.sql';" '1'
        Query "UPDATE sys_base_menus SET component='views/custom/storage.vue',path='custom-storage',title='Custom storage' WHERE name='files' AND deleted_at IS NULL; UPDATE sys_base_menus SET name='custom-audit',path='old-audit' WHERE name='audit' AND deleted_at IS NULL; INSERT INTO sys_base_menus (created_at,updated_at,parent_id,path,name,component,title,hidden) VALUES(NOW(3),NOW(3),0,'/admin/audit','custom-absolute-audit','views/custom/audit.vue','Custom audit',0);" | Out-Null
        Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'run', $menuMigration) | Out-Host
        Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='files' AND component='views/custom/storage.vue' AND path='custom-storage' AND title='Custom storage' AND deleted_at IS NULL;" '1'
        Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='audit' AND deleted_at IS NULL;" '0'
    } finally { Docker @('exec', $container, 'rm', '-f', '--', $menuMigration) | Out-Null }
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
    Write-Host 'PASS: fresh install, repeat migration, schema checks, frontend seed, old canonical conflict reproduced, compatibility repair, fallback collisions, grants preserved, customized conflict skipped, SQL failure stop, retry, checksum guard.'
} finally {
    if ($created) { Docker @('exec', $container, 'sh', $helper, 'query', "DROP DATABASE $database;") | Out-Null }
    Docker @('exec', $container, 'rm', '-f', '--', $helper, $baseline) | Out-Null
    # Keep the test copies and backups under ignored bin/ for failure inspection.
}
