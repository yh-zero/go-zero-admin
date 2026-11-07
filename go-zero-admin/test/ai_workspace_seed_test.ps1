#requires -Version 5.1
$ErrorActionPreference = 'Stop'
$sqlPath = Join-Path $PSScriptRoot '../data/db/init/mysql-all.sql'
$sql = [IO.File]::ReadAllText([IO.Path]::GetFullPath($sqlPath))

function Assert-Equal($actual, $expected, [string]$message) {
    if ($actual -cne $expected) { throw "$message (expected=$expected, actual=$actual)" }
}

# 只读解析完整初始化 SQL；不连接数据库，也不执行种子。
$menus = @([regex]::Matches($sql, "(?m)^\((?<id>\d+),(?<parent>\d+),'(?<title>[^']*)','(?<name>[^']*)',(?<type>\d+),'(?<path>[^']*)','(?<component>[^']*)','(?<permission>[^']*)','(?<icon>[^']*)',(?<sort>\d+),(?<visible>\d+),(?<enabled>\d+),") | ForEach-Object {
    $row = @{}
    foreach ($field in @('id','parent','title','name','type','path','component','permission','icon','sort','visible','enabled')) {
        $row[$field] = $_.Groups[$field].Value
    }
    [pscustomobject]$row
})
$pages = @(
    @{ Id='9001410'; Name='ai_workflow'; Path='/ai/workflows'; Title='常用流程'; Icon='apartment'; Sort='1'; Prefix='sys:ai:workflow:'; Operations=@('list','create','edit','delete','run') },
    @{ Id='9001420'; Name='ai_task'; Path='/ai/tasks'; Title='运行记录'; Icon='audit'; Sort='2'; Prefix='sys:ai:task:'; Operations=@('list','approve','reject','cancel') },
    @{ Id='9001430'; Name='ai_tool'; Path='/ai/tools'; Title='工具与能力'; Icon='api'; Sort='3'; Prefix='sys:ai:tool:'; Operations=@('list') }
)
$newIds = @()
foreach ($page in $pages) {
    $rows = @($menus | Where-Object { $_.id -eq $page.Id })
    Assert-Equal $rows.Count 1 "页面 ID 必须唯一：$($page.Id)"
    $row = $rows[0]
    foreach ($field in @('name','path','title','icon','sort')) { Assert-Equal $row.$field $page[$field] "页面字段不匹配：$($page.Id).$field" }
    Assert-Equal $row.component $page.Name '页面组件必须与注册标识一致'
    Assert-Equal $row.parent '9001400' '页面必须挂载 AI 功能'
    Assert-Equal $row.type '2' '页面菜单类型错误'
    Assert-Equal $row.visible '1' '页面必须可见'
    Assert-Equal $row.enabled '1' '页面必须启用'
    Assert-Equal @($menus | Where-Object { $_.name -eq $page.Name }).Count 1 '页面名称必须唯一'
    $newIds += $page.Id
    $buttons = @($menus | Where-Object { $_.parent -eq $page.Id })
    Assert-Equal $buttons.Count $page.Operations.Count '操作码数量不匹配'
    for ($i = 0; $i -lt $page.Operations.Count; $i++) {
        $permission = $page.Prefix + $page.Operations[$i]
        $matches = @($menus | Where-Object { $_.permission -eq $permission })
        Assert-Equal $matches.Count 1 "操作码必须存在且唯一：$permission"
        $button = $matches[0]
        Assert-Equal $button.id ([string]([int]$page.Id + $i + 1)) '操作 ID 不匹配'
        Assert-Equal $button.parent $page.Id '操作必须属于对应页面'
        Assert-Equal $button.type '3' '操作必须使用按钮菜单类型'
        Assert-Equal $button.visible '0' '按钮不能显示为导航项'
        Assert-Equal $button.enabled '1' '按钮必须启用'
        $newIds += $button.id
    }
}
$associations = @([regex]::Matches($sql, "(?m)^\((?<role>\d+),(?<menu>\d+),'\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}'\)[,;]") | Where-Object { $newIds -contains $_.Groups['menu'].Value })
Assert-Equal $associations.Count $newIds.Count '每个新增菜单必须且仅关联一个角色'
foreach ($id in $newIds) {
    $grants = @($associations | Where-Object { $_.Groups['menu'].Value -eq $id })
    Assert-Equal $grants.Count 1 "菜单关联必须唯一：$id"
    Assert-Equal $grants[0].Groups['role'].Value '1' "新增 AI 菜单只能授予超级管理员：$id"
}
Write-Host 'AI workspace seed regression passed: 3 pages, 10 operations, role 1 only.'
exit 0
