#requires -Version 5.1
<#
本地真实 HTTP 回归：角色/菜单/字典部分更新，以及仅使用 text_stats 的两审批点工作流。
凭据和已正常取得的一次性验证码仅从参数或 MUTATION_E2E_* 环境变量传入；
BASE_URL 不包含 /v1。验证码开启时，每次运行前须重新正常取得 CAPTCHA_ID/CAPTCHA。
CLEANUP_MANIFEST 必须是本仓库内被 git 忽略且尚不存在的文件路径。
脚本不输出凭据、验证码、token、个人字段或响应正文，不读取数据库/Redis。
只撤销自己本次登录的会话；finally 按精确 ID 通过 API 软删自己创建的对象。
任务无 HTTP 删除入口，任务 ID 留在清理清单，供调用方按本轮 ID 最终清理。
#>
[CmdletBinding()]
param(
    [string]$BaseUrl = $env:MUTATION_E2E_BASE_URL,
    [string]$Username = $env:MUTATION_E2E_USERNAME,
    [string]$Password = $env:MUTATION_E2E_PASSWORD,
    [string]$CaptchaId = $env:MUTATION_E2E_CAPTCHA_ID,
    [string]$Captcha = $env:MUTATION_E2E_CAPTCHA,
    [string]$CleanupManifestPath = $env:MUTATION_E2E_CLEANUP_MANIFEST
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$VerbosePreference = 'SilentlyContinue'
$DebugPreference = 'SilentlyContinue'
$script:CheckCount = 0
$script:CurrentStep = '参数检查'
$script:Failure = '发生未预期错误'
$script:ActiveAccess = ''
$script:ManifestFile = ''
$script:Prefix = 'normfix_' + [DateTime]::UtcNow.ToString('yyyyMMddHHmmssfff') + '_' + [Guid]::NewGuid().ToString('N').Substring(0, 8)
$script:Created = @{}
$script:Deleted = @{}
$script:Attempted = @{}
foreach ($kind in @('roles', 'menus', 'dicts', 'workflows', 'tasks')) {
    $script:Created[$kind] = New-Object System.Collections.ArrayList
    $script:Deleted[$kind] = New-Object System.Collections.ArrayList
    $script:Attempted[$kind] = $false
}
$script:Names = @{
    role_code = $script:Prefix + '_role'
    menu_name = $script:Prefix + '_menu'
    dict_code = $script:Prefix + '_dict'
    workflow_name = $script:Prefix + '_workflow'
}
$testExitCode = 0
$login = $loginBody = $options = $null

function Stop-Test([string]$Reason) {
    $script:Failure = $Reason
    throw 'Mutation HTTP regression assertion failed.'
}

function Assert-Test([bool]$Condition, [string]$Reason) {
    if (-not $Condition) { Stop-Test $Reason }
    $script:CheckCount++
}

function Read-Field($Object, [string]$Name) {
    if ($null -eq $Object -or $Object.PSObject.Properties.Name -notcontains $Name) { return $null }
    return $Object.$Name
}

function Save-Manifest {
    if (-not $script:ManifestFile) { return }
    $manifest = [ordered]@{
        prefix = $script:Prefix
        updated_at_utc = [DateTime]::UtcNow.ToString('o')
        names = $script:Names
        attempted = $script:Attempted
        ids = [ordered]@{}
        api_soft_deleted_ids = [ordered]@{}
    }
    foreach ($kind in @('roles', 'menus', 'dicts', 'workflows', 'tasks')) {
        $manifest.ids[$kind] = @($script:Created[$kind].ToArray())
        $manifest.api_soft_deleted_ids[$kind] = @($script:Deleted[$kind].ToArray())
    }
    # 文件内只有本轮测试名称与 ID；先写临时文件，再替换自身清单，防止中断留下半份 JSON。
    $temporaryFile = $script:ManifestFile + '.tmp'
    [IO.File]::WriteAllText($temporaryFile, ($manifest | ConvertTo-Json -Depth 8), [Text.UTF8Encoding]::new($false))
    Move-Item -LiteralPath $temporaryFile -Destination $script:ManifestFile -Force
}

function Register-Created([string]$Kind, [long]$Id) {
    Assert-Test ($Id -gt 0) '创建对象未取得有效 ID'
    if (-not $script:Created[$Kind].Contains($Id)) { $null = $script:Created[$Kind].Add($Id) }
    Save-Manifest
}

function Register-Deleted([string]$Kind, [long]$Id) {
    if (-not $script:Deleted[$Kind].Contains($Id)) { $null = $script:Deleted[$Kind].Add($Id) }
    Save-Manifest
}

function Invoke-TestApi {
    param(
        [string]$Step,
        [string]$Method,
        [string]$Path,
        [hashtable]$Body = $null,
        [int]$ExpectedCode = 200,
        [switch]$Anonymous
    )
    $script:CurrentStep = $Step
    $script:Failure = '请求或响应处理失败'
    $requestArgs = @{
        Uri = $script:ApiRoot + $Path
        Method = $Method
        UseBasicParsing = $true
        TimeoutSec = 30
        MaximumRedirection = 0
        ErrorAction = 'Stop'
        Headers = @{ Accept = 'application/json' }
    }
    if (-not $Anonymous -and $script:ActiveAccess) { $requestArgs.Headers.Authorization = 'Bearer ' + $script:ActiveAccess }
    if ($null -ne $Body) {
        $requestArgs.ContentType = 'application/json; charset=utf-8'
        $requestArgs.Body = [Text.Encoding]::UTF8.GetBytes(($Body | ConvertTo-Json -Depth 16 -Compress))
    }
    try { $response = Invoke-WebRequest @requestArgs }
    catch { Stop-Test '未收到正常 HTTP 响应；网络或 HTTP 错误不能当作预期业务拒绝' }
    Assert-Test ([int]$response.StatusCode -eq 200) '业务接口未返回 HTTP 200'
    try { $envelope = $response.Content | ConvertFrom-Json }
    catch { Stop-Test '响应不是有效 JSON' }
    $code = Read-Field $envelope 'code'
    $message = Read-Field $envelope 'msg'
    Assert-Test (($code -is [int] -or $code -is [long]) -and $message -is [string]) '统一信封 code/msg 类型不正确'
    Assert-Test ($code -eq $ExpectedCode) ('业务码与预期不符；预期为 ' + $ExpectedCode + '，实际为 ' + $code)
    if ($ExpectedCode -eq 200) {
        Assert-Test ($null -ne (Read-Field $envelope 'result')) '成功信封缺少非 null 的 result'
    }
    return $envelope
}

function Find-CreatedIds([string]$Kind) {
    # Create 无 ID 的三个旧接口只能按本轮唯一名称精确定位；不取列表首行。
    switch ($Kind) {
        'roles' {
            $code = [Uri]::EscapeDataString($script:Names.role_code)
            $response = Invoke-TestApi '按唯一编码定位本轮角色' 'GET' ('/v1/sys/role/list?page=1&page_size=100&code=' + $code)
            $matches = @($response.result.list | Where-Object { $_.code -ceq $script:Names.role_code })
        }
        'menus' {
            $response = Invoke-TestApi '按唯一名称定位本轮菜单' 'GET' '/v1/sys/menu/tree'
            $matches = @($response.result.list | Where-Object { $_.name -ceq $script:Names.menu_name })
        }
        'dicts' {
            $code = [Uri]::EscapeDataString($script:Names.dict_code)
            $response = Invoke-TestApi '按唯一编码定位本轮字典' 'GET' ('/v1/sys/dict/list?page=1&page_size=100&code=' + $code)
            $matches = @($response.result.list | Where-Object { $_.code -ceq $script:Names.dict_code })
        }
        'workflows' {
            # 创建响应若丢失，分页读取本人才可恢复本轮唯一名称对应的 ID。
            $matches = @()
            for ($page = 1; $page -le 100; $page++) {
                $response = Invoke-TestApi '恢复本轮工作流 ID' 'GET' ('/v1/sys/ai/workflows?page=' + $page + '&page_size=100')
                $rows = @($response.result.list)
                $matches += @($rows | Where-Object { $_.name -ceq $script:Names.workflow_name })
                if ($rows.Count -eq 0 -or $page * 100 -ge [long]$response.result.total) { break }
                if ($page -eq 100) { Stop-Test '工作流恢复分页超过上限，需根据清单人工核对' }
            }
        }
        'tasks' {
            $matches = @()
            foreach ($workflowId in $script:Created.workflows.ToArray()) {
                $response = Invoke-TestApi '恢复本轮工作流任务 ID' 'GET' ('/v1/sys/ai/tasks?page=1&page_size=100&workflow_id=' + $workflowId)
                $rows = @($response.result.list)
                Assert-Test ([long]$response.result.total -le 100) '本轮工作流任务数异常，不能遗漏清理 ID'
                $matches += @($rows | Where-Object { [long]$_.workflow_id -eq $workflowId })
            }
        }
        default { Stop-Test '未知测试对象类型' }
    }
    foreach ($row in $matches) { Register-Created $Kind ([long]$row.id) }
    return $matches
}

function Get-TestRole([long]$Id) {
    $response = Invoke-TestApi '读取本轮角色详情' 'GET' ('/v1/sys/role/' + $Id)
    return $response.result.role
}

function Get-TestMenu([long]$Id) {
    $response = Invoke-TestApi '读取本轮菜单详情' 'GET' ('/v1/sys/menu/' + $Id)
    return $response.result.menu
}

function Get-TestDict {
    $response = Invoke-TestApi '读取本轮字典详情' 'GET' ('/v1/sys/dict/code/' + [Uri]::EscapeDataString($script:Names.dict_code))
    return $response.result.dict
}

function Get-TestTask([long]$Id) {
    $response = Invoke-TestApi '读取本轮任务断点' 'GET' ('/v1/sys/ai/tasks/' + $Id)
    return $response.result.task
}

try {
    foreach ($required in @($BaseUrl, $Username, $Password, $CleanupManifestPath)) {
        if ([string]::IsNullOrWhiteSpace($required)) { Stop-Test '缺少服务地址、测试账号或清理清单参数' }
    }
    $targetUri = $null
    Assert-Test ([Uri]::TryCreate($BaseUrl, [UriKind]::Absolute, [ref]$targetUri)) '服务根地址不是有效 URI'
    Assert-Test ($targetUri.Scheme -in @('http', 'https') -and $targetUri.IsLoopback -and -not $targetUri.UserInfo -and -not $targetUri.Query -and -not $targetUri.Fragment -and $targetUri.AbsolutePath -eq '/') '仅接受不含路径、凭据或查询的本地 HTTP(S) 服务根地址'
    $script:ApiRoot = $BaseUrl.TrimEnd('/')
    $repositoryRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..')).TrimEnd('\', '/')
    $manifestFile = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($CleanupManifestPath)
    Assert-Test ($manifestFile.StartsWith($repositoryRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) '清理清单必须位于当前仓库内'
    Assert-Test (-not (Test-Path -LiteralPath $manifestFile)) '清理清单已存在；请提供新文件，避免覆盖上次未清理 ID'
    $manifestDirectory = Split-Path -Parent $manifestFile
    Assert-Test (Test-Path -LiteralPath $manifestDirectory -PathType Container) '清理清单目录须已存在'
    $relativeManifest = $manifestFile.Substring($repositoryRoot.Length + 1).Replace('\', '/')
    & git -C $repositoryRoot check-ignore --quiet -- $relativeManifest
    Assert-Test ($LASTEXITCODE -eq 0) '清理清单必须被仓库 git 忽略'
    $script:ManifestFile = $manifestFile
    Save-Manifest

    $ready = Invoke-TestApi '服务就绪' 'GET' '/ready' -Anonymous
    Assert-Test ($ready.result.ready -eq $true) '服务未就绪'
    $options = Invoke-TestApi '读取正常验证码配置' 'GET' '/v1/sys/auth/captcha' -Anonymous
    Assert-Test ((Read-Field $options.result 'enabled') -is [bool]) '验证码配置缺少 enabled'
    $loginBody = @{ username = $Username; password = $Password }
    if ($options.result.enabled) {
        if ([string]::IsNullOrWhiteSpace($CaptchaId) -or [string]::IsNullOrWhiteSpace($Captcha)) { Stop-Test '验证码已启用；须提供本次正常取得的验证码' }
        $loginBody.captcha_id = $CaptchaId
        $loginBody.captcha = $Captcha
    }
    $login = Invoke-TestApi '正常账号登录' 'POST' '/v1/sys/auth/login' $loginBody -Anonymous
    $script:ActiveAccess = [string](Read-Field $login.result 'access_token')
    Assert-Test (-not [string]::IsNullOrWhiteSpace($script:ActiveAccess)) '登录未返回有效访问令牌'
    $Password = $Captcha = $loginBody = $login = $options = $null

    $script:Attempted.menus = $true
    Save-Manifest
    $null = Invoke-TestApi '创建默认启用菜单' 'POST' '/v1/sys/menu/create' @{
        title = $script:Names.menu_name; name = $script:Names.menu_name; parent_id = 0; type = 2
        path = '/normfix/' + $script:Prefix; component = 'normfix'; icon = 'setting'; sort = 9; visible = $true
    }
    $matches = @(Find-CreatedIds 'menus')
    Assert-Test ($matches.Count -eq 1) '本轮唯一菜单未精确定位为一行'
    $menuId = [long]$matches[0].id
    $menu = Get-TestMenu $menuId
    Assert-Test ($menu.status -eq 1 -and $menu.visible -eq $true -and $menu.sort -eq 9) '菜单省略 status 未默认启用或初始字段错误'

    $script:Attempted.roles = $true
    Save-Manifest
    $null = Invoke-TestApi '创建默认范围与状态角色' 'POST' '/v1/sys/role/create' @{
        name = $script:Names.role_code; code = $script:Names.role_code; description = 'initial'; sort = 7; menu_ids = @($menuId)
    }
    $matches = @(Find-CreatedIds 'roles')
    Assert-Test ($matches.Count -eq 1) '本轮唯一角色未精确定位为一行'
    $roleId = [long]$matches[0].id
    $role = Get-TestRole $roleId
    Assert-Test ($role.status -eq 1 -and $role.data_scope -ceq 'SELF') '角色省略 status/data_scope 默认值不正确'
    Assert-Test (@($role.menu_ids).Count -eq 1 -and $role.menu_ids[0] -eq $menuId) '角色初始菜单授权不正确'
    $editedName = $script:Names.role_code + '_edited'
    $null = Invoke-TestApi '角色省略字段保持' 'PUT' ('/v1/sys/role/' + $roleId) @{ name = $editedName }
    $role = Get-TestRole $roleId
    Assert-Test ($role.name -ceq $editedName -and $role.code -ceq $script:Names.role_code -and $role.description -ceq 'initial' -and $role.sort -eq 7 -and $role.status -eq 1 -and $role.data_scope -ceq 'SELF') '角色省略字段被覆盖'
    Assert-Test (@($role.menu_ids).Count -eq 1 -and $role.menu_ids[0] -eq $menuId) '普通角色编辑覆盖了原菜单授权'
    $null = Invoke-TestApi '角色 null 字段保持' 'PUT' ('/v1/sys/role/' + $roleId) @{
        name = $null; code = $null; description = $null; sort = $null; status = $null; data_scope = $null; menu_ids = $null
    }
    $role = Get-TestRole $roleId
    Assert-Test ($role.name -ceq $editedName -and $role.code -ceq $script:Names.role_code -and $role.description -ceq 'initial' -and $role.sort -eq 7 -and $role.status -eq 1 -and $role.data_scope -ceq 'SELF' -and @($role.menu_ids).Count -eq 1) '角色 null 字段未保持'
    $null = Invoke-TestApi '角色显式清空说明与零排序' 'PUT' ('/v1/sys/role/' + $roleId) @{ description = ''; sort = 0 }
    $role = Get-TestRole $roleId
    Assert-Test ($role.description -ceq '' -and $role.sort -eq 0 -and $role.name -ceq $editedName -and @($role.menu_ids).Count -eq 1) '角色空说明/零排序语义不正确'
    $null = Invoke-TestApi '拒绝修改角色编码' 'PUT' ('/v1/sys/role/' + $roleId) @{ code = $script:Names.role_code + '_different' } -ExpectedCode 400
    $role = Get-TestRole $roleId
    Assert-Test ($role.code -ceq $script:Names.role_code -and @($role.menu_ids).Count -eq 1) '角色编码拒绝后仍改变数据'
    $null = Invoke-TestApi '角色空数组清空菜单授权' 'PUT' ('/v1/sys/role/' + $roleId) @{ menu_ids = @() }
    $role = Get-TestRole $roleId
    # protobuf repeated 空集合经 RPC 可变为 JSON null；null 与 [] 均表示没有授权。
    Assert-Test ($null -eq $role.menu_ids -or @($role.menu_ids).Count -eq 0) '角色 menu_ids 空数组未清空'

    $null = Invoke-TestApi '菜单纯资料 patch 的 false/0/空值' 'PUT' ('/v1/sys/menu/' + $menuId) @{
        visible = $false; sort = 0; path = ''; component = ''; icon = ''; perm_code = ''
    }
    $menu = Get-TestMenu $menuId
    Assert-Test ($menu.visible -is [bool] -and -not $menu.visible -and $menu.sort -eq 0 -and $menu.path -ceq '' -and $menu.component -ceq '' -and $menu.icon -ceq '' -and $menu.perm_code -ceq '') '菜单 false/0/空字符串未生效'
    Assert-Test ($menu.title -ceq $script:Names.menu_name -and $menu.name -ceq $script:Names.menu_name -and $menu.type -eq 2 -and $menu.status -eq 1 -and $menu.parent_id -eq 0) '菜单纯资料 patch 覆盖了省略字段'
    $null = Invoke-TestApi '菜单 null 保持当前资料' 'PUT' ('/v1/sys/menu/' + $menuId) @{ title = $null; visible = $null; sort = $null; path = $null }
    $menu = Get-TestMenu $menuId
    Assert-Test ($menu.title -ceq $script:Names.menu_name -and $menu.visible -eq $false -and $menu.sort -eq 0 -and $menu.path -ceq '') '菜单 null 覆盖了已有资料'
    $null = Invoke-TestApi '拒绝菜单挂载自身' 'PUT' ('/v1/sys/menu/' + $menuId) @{ parent_id = $menuId } -ExpectedCode 400
    $menu = Get-TestMenu $menuId
    Assert-Test ($menu.parent_id -eq 0) '菜单自父请求被拒绝后仍改变树'
    $null = Invoke-TestApi '删除本轮菜单' 'DELETE' ('/v1/sys/menu/' + $menuId)
    Register-Deleted 'menus' $menuId
    $null = Invoke-TestApi '删除菜单不可再次修改' 'PUT' ('/v1/sys/menu/' + $menuId) @{ title = 'must_not_restore' } -ExpectedCode 20003
    $null = Invoke-TestApi '删除菜单保持不可读取' 'GET' ('/v1/sys/menu/' + $menuId) -ExpectedCode 20003

    $script:Attempted.dicts = $true
    Save-Manifest
    $null = Invoke-TestApi '创建默认启用字典' 'POST' '/v1/sys/dict/create' @{
        name = $script:Names.dict_code; code = $script:Names.dict_code; description = 'initial'
        items = @(@{ label = 'first'; value = 'one'; sort = 1 }, @{ label = 'second'; value = 'two'; sort = 2 })
    }
    $matches = @(Find-CreatedIds 'dicts')
    Assert-Test ($matches.Count -eq 1) '本轮唯一字典未精确定位为一行'
    $dictId = [long]$matches[0].id
    $dict = Get-TestDict
    Assert-Test ($dict.status -eq 1 -and @($dict.items).Count -eq 2) '字典省略 status 默认值或初始条目错误'
    $null = Invoke-TestApi '字典省略 items 保持' 'PUT' ('/v1/sys/dict/' + $dictId) @{ description = 'updated' }
    $dict = Get-TestDict
    Assert-Test ($dict.description -ceq 'updated' -and @($dict.items).Count -eq 2 -and $dict.items[0].value -ceq 'one' -and $dict.items[1].value -ceq 'two') '字典省略 items 改变了原条目'
    $null = Invoke-TestApi '字典 null 保持' 'PUT' ('/v1/sys/dict/' + $dictId) @{ items = $null; name = $null; code = $null; status = $null }
    $dict = Get-TestDict
    Assert-Test ($dict.name -ceq $script:Names.dict_code -and $dict.code -ceq $script:Names.dict_code -and $dict.status -eq 1 -and @($dict.items).Count -eq 2 -and $dict.items[0].value -ceq 'one' -and $dict.items[1].value -ceq 'two') '字典 null 未保持原值'
    $null = Invoke-TestApi '字典空数组清空 items' 'PUT' ('/v1/sys/dict/' + $dictId) @{ items = @() }
    $dict = Get-TestDict
    Assert-Test (@($dict.items).Count -eq 0 -and $dict.description -ceq 'updated' -and $dict.status -eq 1) '字典空数组未清空或覆盖省略字段'

    # 两步均为本地纯文本工具，不调用模型、邮件或其他外部有副作用能力。
    $definition = @{ steps = @(
        @{ name = 'first_local_check'; tool = 'text_stats'; input = @{ text = 'local first' }; approval = $true },
        @{ name = 'second_local_check'; tool = 'text_stats'; input = @{ text = 'local second' }; approval = $true }
    ) } | ConvertTo-Json -Depth 8 -Compress
    $script:Attempted.workflows = $true
    Save-Manifest
    $workflowResponse = Invoke-TestApi '创建两审批点本地工作流' 'POST' '/v1/sys/ai/workflows' @{ name = $script:Names.workflow_name; definition = $definition }
    $workflowId = [long]$workflowResponse.result.workflow.id
    Register-Created 'workflows' $workflowId
    $script:Attempted.tasks = $true
    Save-Manifest
    $run = Invoke-TestApi '运行本地工具工作流' 'POST' ('/v1/sys/ai/workflows/' + $workflowId + '/run') @{ input = '{}' }
    $taskId = [long]$run.result.task_id
    Register-Created 'tasks' $taskId
    Assert-Test ($run.result.status -ceq 'pending_approval') '首审批点未暂停'
    $task = Get-TestTask $taskId
    Assert-Test ($task.status -ceq 'pending_approval' -and (Read-Field $task 'next_step') -eq 0 -and $task.result -ceq '') '任务首审批点 next_step 不是 0 或提前执行工具'
    $null = Invoke-TestApi '批准缺 expected_step 拒绝' 'POST' ('/v1/sys/ai/tasks/' + $taskId + '/approve') @{} -ExpectedCode 400
    $null = Invoke-TestApi '拒绝缺 expected_step 拒绝' 'POST' ('/v1/sys/ai/tasks/' + $taskId + '/reject') @{ reason = 'missing expected step' } -ExpectedCode 400
    $task = Get-TestTask $taskId
    Assert-Test ($task.status -ceq 'pending_approval' -and $task.next_step -eq 0 -and $task.result -ceq '') '缺断点请求推进了任务'
    $null = Invoke-TestApi '显式 0 批准首审批点' 'POST' ('/v1/sys/ai/tasks/' + $taskId + '/approve') @{ expected_step = 0 }
    $task = Get-TestTask $taskId
    Assert-Test ($task.status -ceq 'pending_approval' -and $task.next_step -eq 1) '首批准后未暂停第二审批点'
    $firstResult = [string]$task.result
    Assert-Test ([regex]::Matches($firstResult, 'first_local_check:').Count -eq 1 -and [regex]::Matches($firstResult, 'second_local_check:').Count -eq 0) '首工具未执行一次或第二工具提前执行'
    $null = Invoke-TestApi '迟到首断点批准拒绝' 'POST' ('/v1/sys/ai/tasks/' + $taskId + '/approve') @{ expected_step = 0 } -ExpectedCode 400
    $null = Invoke-TestApi '迟到首断点拒绝不得终结任务' 'POST' ('/v1/sys/ai/tasks/' + $taskId + '/reject') @{ expected_step = 0; reason = 'stale first step' } -ExpectedCode 400
    $task = Get-TestTask $taskId
    Assert-Test ($task.status -ceq 'pending_approval' -and $task.next_step -eq 1 -and $task.result -ceq $firstResult) '迟到请求改变了第二审批点或重复执行工具'
    $null = Invoke-TestApi '批准第二审批点' 'POST' ('/v1/sys/ai/tasks/' + $taskId + '/approve') @{ expected_step = 1 }
    $task = Get-TestTask $taskId
    Assert-Test ($task.status -ceq 'succeeded' -and $task.next_step -eq 2 -and $task.error -ceq '') '第二批准后任务未成功完成'
    Assert-Test ([regex]::Matches([string]$task.result, 'first_local_check:').Count -eq 1 -and [regex]::Matches([string]$task.result, 'second_local_check:').Count -eq 1) '本地两个工具没有分别只执行一次'
}
catch {
    $testExitCode = 1
    [Console]::Error.WriteLine('变更 HTTP 验证失败（步骤：' + $script:CurrentStep + '）：' + $script:Failure)
}
finally {
    if ($script:ActiveAccess) {
        # 请求已提交但响应丢失时，按本轮唯一名称/工作流 ID 尝试补登记，绝不选择其他对象。
        foreach ($kind in @('roles', 'menus', 'dicts', 'workflows', 'tasks')) {
            if ($script:Attempted[$kind] -and $script:Created[$kind].Count -eq 0) {
                try { $null = Find-CreatedIds $kind }
                catch {
                    $testExitCode = 1
                    [Console]::Error.WriteLine('未确认本轮 ' + $kind + ' 的恢复 ID；请按清单唯一名称核对，不能记作通过。')
                }
            }
        }
        foreach ($kind in @('roles', 'dicts', 'menus', 'workflows')) {
            $route = @{ roles = '/v1/sys/role/'; dicts = '/v1/sys/dict/'; menus = '/v1/sys/menu/'; workflows = '/v1/sys/ai/workflows/' }[$kind]
            foreach ($id in $script:Created[$kind].ToArray()) {
                if ($script:Deleted[$kind].Contains($id)) { continue }
                try {
                    $null = Invoke-TestApi ('按精确 ID 清理本轮 ' + $kind) 'DELETE' ($route + $id)
                    Register-Deleted $kind $id
                }
                catch {
                    $testExitCode = 1
                    [Console]::Error.WriteLine('未确认本轮 ' + $kind + ' ID=' + $id + ' 已软删除；清单保留，不能记作通过。')
                }
            }
        }
        try { $null = Invoke-TestApi '只退出本次登录会话' 'POST' '/v1/sys/auth/logout' @{ all_devices = $false } }
        catch {
            $testExitCode = 1
            [Console]::Error.WriteLine('未确认本次测试会话退出；请通过设备会话页检查，未退出其他设备。')
        }
    }
    try { Save-Manifest }
    catch { $testExitCode = 1; [Console]::Error.WriteLine('清理清单写入失败；不能记作通过。') }
    $Password = $Captcha = $CaptchaId = $script:ActiveAccess = $loginBody = $login = $options = $null
}

$safeIds = [ordered]@{}
foreach ($kind in @('roles', 'menus', 'dicts', 'workflows', 'tasks')) { $safeIds[$kind] = @($script:Created[$kind].ToArray()) }
Write-Host ('本轮创建 IDs：' + ($safeIds | ConvertTo-Json -Compress))
Write-Host ('checkcount=' + $script:CheckCount)
if ($testExitCode -eq 0) { Write-Host '变更 HTTP 验证通过；API 可清理对象已按本轮 ID 软删除，任务 ID 已写清单。' }
exit $testExitCode
