#requires -Version 5.1
# 本机前端联调数据导入。须先启动独立 MySQL；不启动应用，不修改既有业务记录。
# 用法：powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\seed-test-data.ps1
[CmdletBinding()]
param(
    [ValidateSet('gozero-new-mysql')][string]$Container = 'gozero-new-mysql',
    [ValidateSet('gozero_admin')][string]$Database = 'gozero_admin'
)
$ErrorActionPreference = 'Stop'
$fixtureFiles = @('seed-test-data.sql', 'seed-full-test-data.sql')
foreach ($fixtureFile in $fixtureFiles) {
    if (-not (Test-Path -LiteralPath (Join-Path $PSScriptRoot $fixtureFile) -PathType Leaf)) {
        throw ('找不到联调 SQL 文件：' + $fixtureFile)
    }
}
$state = & docker inspect --format '{{.State.Running}}' $Container
if ($LASTEXITCODE -ne 0 -or $state -ne 'true') { throw '独立联调 MySQL 容器尚未运行。' }
foreach ($fixtureFile in $fixtureFiles) {
    $fixtureSql = Join-Path $PSScriptRoot $fixtureFile
    $containerSql = '/tmp/gozero-frontend-fixtures-' + [guid]::NewGuid().ToString('N') + '.sql'
    $copied = $false
    try {
        # 复制 UTF-8 文件，避免 PowerShell 5.1 管道将中文转为问号。
        & docker cp $fixtureSql ($Container + ':' + $containerSql)
        if ($LASTEXITCODE -ne 0) { throw '复制联调 SQL 失败。' }
        $copied = $true

        # 每份 SQL 自成事务；失败关闭连接回滚该份，已成功的基础数据可安全重跑。
        # 密码仅在容器内经环境变量传入；输出只保留聚合统计，不展示账号资料。
        $mysqlCommand = 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --user=root --default-character-set=utf8mb4 --batch --raw "$1" < "$2"'
        $importOutput = & docker exec $Container sh -c $mysqlCommand sh $Database $containerSql
        if ($LASTEXITCODE -ne 0) {
            throw ('联调数据导入失败并回滚当前文件：' + $fixtureFile + '。请检查基础数据、表结构及 fixture 身份冲突。')
        }
        $importOutput | Where-Object {
            $_ -match '^(table_name\t|sys_[a-z_]+\t|casbin_rule\t|module_check\t|[a-z_]+_verified\t|result\t|full_fixture_committed\t)'
        } | Write-Output
        Write-Host ($fixtureFile + ' 导入完成；既有记录未覆盖。')
    } finally {
        if ($copied) {
            & docker exec $Container rm -f -- $containerSql
            if ($LASTEXITCODE -ne 0) { Write-Warning ('请手动清理容器临时 SQL：' + $containerSql) }
        }
    }
}
