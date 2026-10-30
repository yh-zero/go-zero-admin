#requires -Version 5.1
# 本机前端联调数据导入。须先启动独立 MySQL；不启动应用，不修改既有业务记录。
# 用法：powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\seed-test-data.ps1
[CmdletBinding()]
param(
    [ValidateSet('gozero-new-mysql')][string]$Container = 'gozero-new-mysql',
    [ValidateSet('gozero_admin')][string]$Database = 'gozero_admin'
)
$ErrorActionPreference = 'Stop'
$fixtureSql = Join-Path $PSScriptRoot 'seed-test-data.sql'
if (-not (Test-Path -LiteralPath $fixtureSql -PathType Leaf)) { throw '找不到联调 SQL 文件。' }
$state = & docker inspect --format '{{.State.Running}}' $Container
if ($LASTEXITCODE -ne 0 -or $state -ne 'true') { throw '独立联调 MySQL 容器尚未运行。' }
$containerSql = '/tmp/gozero-frontend-fixtures-' + [guid]::NewGuid().ToString('N') + '.sql'
$copied = $false
try {
    # 复制 UTF-8 文件，避免 PowerShell 5.1 管道将中文转为问号。
    & docker cp $fixtureSql ($Container + ':' + $containerSql)
    if ($LASTEXITCODE -ne 0) { throw '复制联调 SQL 失败。' }
    $copied = $true

    # 密码仅在容器内由环境变量传给 mysql，不进入参数、输出、SQL 或宿主机文件。
    $mysqlCommand = 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --user=root --default-character-set=utf8mb4 --batch --raw "$1" < "$2"'
    & docker exec $Container sh -c $mysqlCommand sh $Database $containerSql
    if ($LASTEXITCODE -ne 0) { throw '联调数据导入失败：基础数据缺失或固定 ID 与自然身份冲突时已自动回滚。' }
    Write-Host '联调数据导入完成，已核对固定 fixture ID 与自然身份；既有记录未覆盖。'
} finally {
    if ($copied) {
        & docker exec $Container rm -f -- $containerSql
        if ($LASTEXITCODE -ne 0) { Write-Warning ('请手动清理容器临时 SQL：' + $containerSql) }
    }
}
