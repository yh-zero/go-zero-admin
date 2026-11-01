#requires -Version 5.1
<#
独立认证 HTTP 验证，仅使用调用者提供的专用测试账号，不读取数据库或临时 probe。
验证码启用时，先通过正常流程取得 captcha_id，并人工识别图片得到 captcha；一次有效，
每次运行须重新取得。关闭时只需账号密码。可通过同名参数或 LOGIN_E2E_* 环境变量提供以下值：
BASE_URL、USERNAME、PASSWORD、CAPTCHA_ID、CAPTCHA。
BASE_URL 是 API 服务根地址（不包含 /v1）。建议通过环境变量传凭据，避免命令历史记录。
脚本不打印凭据、验证码、token 或响应正文；只退出本次创建的会话，不退出其他设备。
#>
[CmdletBinding()]
param(
    [string]$BaseUrl = $env:LOGIN_E2E_BASE_URL,
    [string]$Username = $env:LOGIN_E2E_USERNAME,
    [string]$Password = $env:LOGIN_E2E_PASSWORD,
    [string]$CaptchaId = $env:LOGIN_E2E_CAPTCHA_ID,
    [string]$Captcha = $env:LOGIN_E2E_CAPTCHA
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$script:CheckCount = 0
$script:CurrentStep = '参数检查'
$script:Failure = '发生未预期错误'
$activeAccess = ''
$logoutConfirmed = $false
$testExitCode = 0

function Stop-Test([string]$Reason) {
    $script:Failure = $Reason
    throw 'Authentication E2E assertion failed.'
}

function Assert-Test([bool]$Condition, [string]$Reason) {
    if (-not $Condition) { Stop-Test $Reason }
    $script:CheckCount++
}

function Read-StringField($Object, [string]$Name) {
    if ($null -eq $Object -or $Object.PSObject.Properties.Name -notcontains $Name) { return '' }
    $value = $Object.$Name
    if ($value -isnot [string]) { return '' }
    return $value
}

function Invoke-TestApi {
    param(
        [string]$Step,
        [string]$Method,
        [string]$Path,
        [hashtable]$Body = $null,
        [string]$Token = '',
        [int]$ExpectedCode = 200
    )
    $script:CurrentStep = $Step
    $script:Failure = '请求或响应处理失败'
    $requestArgs = @{
        Uri = $script:ApiRoot + $Path
        Method = $Method
        UseBasicParsing = $true
        TimeoutSec = 20
        ErrorAction = 'Stop'
        Headers = @{ Accept = 'application/json' }
    }
    if ($Token) { $requestArgs.Headers.Authorization = 'Bearer ' + $Token }
    if ($null -ne $Body) {
        $requestArgs.ContentType = 'application/json; charset=utf-8'
        $requestArgs.Body = [Text.Encoding]::UTF8.GetBytes(($Body | ConvertTo-Json -Depth 8 -Compress))
    }
    try { $response = Invoke-WebRequest @requestArgs }
    catch { Stop-Test '未收到正常 HTTP 响应；网络错误和 HTTP 错误不能视为预期拒绝' }
    Assert-Test ([int]$response.StatusCode -eq 200) '业务接口应返回 HTTP 200 统一信封'
    try { $envelope = $response.Content | ConvertFrom-Json }
    catch { Stop-Test '响应不是有效 JSON' }
    Assert-Test ($null -ne $envelope -and $envelope.PSObject.Properties.Name -contains 'code' -and $envelope.PSObject.Properties.Name -contains 'msg') '统一信封缺少 code 或 msg'
    Assert-Test (($envelope.code -is [int] -or $envelope.code -is [long]) -and $envelope.msg -is [string]) '统一信封字段类型不正确'
    Assert-Test ($envelope.code -eq $ExpectedCode) ('业务码与预期不符；预期为 ' + $ExpectedCode)
    if ($ExpectedCode -eq 200) {
        Assert-Test ($envelope.msg -ceq '操作成功') '成功信封 msg 不正确'
        Assert-Test ($envelope.PSObject.Properties.Name -contains 'result' -and $null -ne $envelope.result) '成功信封缺少 result'
    }
    return $envelope
}

function Assert-LoginFields($Result) {
    foreach ($name in @('access_token', 'refresh_token', 'session_id')) {
        Assert-Test (-not [string]::IsNullOrWhiteSpace((Read-StringField $Result $name))) ('登录结果缺少 ' + $name)
    }
    foreach ($name in @('expires_at', 'refresh_expires_at')) {
        Assert-Test ($Result.PSObject.Properties.Name -contains $name) ('登录结果缺少 ' + $name)
        Assert-Test ($Result.$name -is [int] -or $Result.$name -is [long]) ('登录期限字段不是整数：' + $name)
    }
    $now = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    Assert-Test ($Result.expires_at -gt $now) '访问令牌期限无效'
    Assert-Test ($Result.refresh_expires_at -ge $Result.expires_at) '刷新令牌期限不应早于访问令牌'
}

function Assert-CurrentSession($Result, [string]$SessionId) {
    Assert-Test ($Result.PSObject.Properties.Name -contains 'list' -and $null -ne $Result.list) '设备会话结果缺少 list'
    $sessions = @($Result.list)
    $matching = @($sessions | Where-Object { $_.id -ceq $SessionId })
    $current = @($sessions | Where-Object { $_.is_current -eq $true })
    Assert-Test ($matching.Count -eq 1) '本人设备列表必须包含本次登录会话且无重复'
    Assert-Test ($current.Count -eq 1 -and $current[0].id -ceq $SessionId) '当前会话标记不正确'
}

try {
    foreach ($required in @($BaseUrl, $Username, $Password)) {
        if ([string]::IsNullOrWhiteSpace($required)) { Stop-Test '缺少参数；请提供服务根地址和测试账号' }
    }
    $targetUri = $null
    Assert-Test ([Uri]::TryCreate($BaseUrl, [UriKind]::Absolute, [ref]$targetUri)) '服务根地址不是有效 URI'
    Assert-Test ($targetUri.Scheme -in @('http', 'https') -and -not $targetUri.UserInfo -and -not $targetUri.Query -and -not $targetUri.Fragment -and $targetUri.AbsolutePath -eq '/') '服务根地址必须使用 HTTP(S)，且不含凭据、路径、查询或片段'
    $script:ApiRoot = $BaseUrl.TrimEnd('/')

    $ping = Invoke-TestApi '存活检查' 'GET' '/ping'
    Assert-Test ($ping.result.msg -ceq 'pong') 'ping 内容不是 pong'
    $ready = Invoke-TestApi '就绪检查' 'GET' '/ready'
    Assert-Test ($ready.result.ready -is [bool] -and $ready.result.ready) 'ready 未确认服务就绪'

    $captchaOptions = Invoke-TestApi '验证码配置' 'GET' '/v1/sys/auth/captcha'
    Assert-Test ($captchaOptions.result.PSObject.Properties.Name -contains 'enabled' -and $captchaOptions.result.enabled -is [bool]) '验证码配置缺少 enabled 布尔值'
    $loginBody = @{ username = $Username; password = $Password }
    if ($captchaOptions.result.enabled) {
        if ([string]::IsNullOrWhiteSpace($CaptchaId) -or [string]::IsNullOrWhiteSpace($Captcha)) { Stop-Test '验证码已启用；请提供本次正常流程取得的验证码' }
        $loginBody.captcha_id = $CaptchaId
        $loginBody.captcha = $Captcha
    } else {
        Assert-Test ($captchaOptions.result.captcha_id -ceq '' -and $captchaOptions.result.image_base64 -ceq '' -and $captchaOptions.result.expires_at -eq 0) '关闭验证码时不应生成验证码'
    }
    $login = Invoke-TestApi '账号登录' 'POST' '/v1/sys/auth/login' $loginBody
    $activeAccess = Read-StringField $login.result 'access_token'
    Assert-LoginFields $login.result
    $firstAccess = $activeAccess
    $firstRefresh = $login.result.refresh_token
    $sessionId = $login.result.session_id

    $sessions = Invoke-TestApi '本人设备会话' 'GET' '/v1/sys/auth/sessions' -Token $activeAccess
    Assert-CurrentSession $sessions.result $sessionId

    $rotated = Invoke-TestApi '刷新令牌轮换' 'POST' '/v1/sys/auth/refresh' @{ refresh_token = $firstRefresh }
    $activeAccess = Read-StringField $rotated.result 'access_token'
    Assert-LoginFields $rotated.result
    Assert-Test ($rotated.result.session_id -ceq $sessionId) '刷新不应创建另一会话'
    Assert-Test ($rotated.result.access_token -cne $firstAccess) '访问令牌未轮换'
    Assert-Test ($rotated.result.refresh_token -cne $firstRefresh) '刷新令牌未轮换'

    $null = Invoke-TestApi '拒绝轮换前访问令牌' 'GET' '/v1/sys/auth/sessions' -Token $firstAccess -ExpectedCode 401
    $null = Invoke-TestApi '拒绝重复使用刷新令牌' 'POST' '/v1/sys/auth/refresh' @{ refresh_token = $firstRefresh } -ExpectedCode 401
    $sessions = Invoke-TestApi '轮换后会话仍有效' 'GET' '/v1/sys/auth/sessions' -Token $activeAccess
    Assert-CurrentSession $sessions.result $sessionId

    $null = Invoke-TestApi '退出当前会话' 'POST' '/v1/sys/auth/logout' @{ all_devices = $false } -Token $activeAccess
    $logoutConfirmed = $true
    $null = Invoke-TestApi '退出后访问令牌失效' 'GET' '/v1/sys/auth/sessions' -Token $activeAccess -ExpectedCode 401
    $null = Invoke-TestApi '退出后刷新令牌失效' 'POST' '/v1/sys/auth/refresh' @{ refresh_token = $rotated.result.refresh_token } -ExpectedCode 401
}
catch {
    $testExitCode = 1
    [Console]::Error.WriteLine('认证验证失败（步骤：' + $script:CurrentStep + '）：' + $script:Failure)
}
finally {
    if ($activeAccess -and -not $logoutConfirmed) {
        # 失败也尝试退出本次会话；不输出异常对象，防止请求正文或令牌出现在诊断信息中。
        try {
            $cleanupArgs = @{
                Uri = $script:ApiRoot + '/v1/sys/auth/logout'; Method = 'POST'; UseBasicParsing = $true
                TimeoutSec = 10; ErrorAction = 'Stop'; ContentType = 'application/json'
                Headers = @{ Authorization = 'Bearer ' + $activeAccess }; Body = '{"all_devices":false}'
            }
            $cleanupResponse = Invoke-WebRequest @cleanupArgs
            $cleanupEnvelope = $cleanupResponse.Content | ConvertFrom-Json
            if ([int]$cleanupResponse.StatusCode -ne 200 -or $cleanupEnvelope.code -notin @(200, 401)) {
                throw 'Session cleanup not confirmed.'
            }
        }
        catch {
            $testExitCode = 1
            [Console]::Error.WriteLine('未确认测试会话已清理，请通过测试账号的设备会话页面检查；本次不能记作通过。')
        }
    }
    $Password = $Captcha = $activeAccess = $null
    $firstAccess = $firstRefresh = $login = $rotated = $sessions = $null
}

if ($testExitCode -eq 0) { Write-Host ('认证 HTTP 验证通过，共 ' + $script:CheckCount + ' 项断言。') }
exit $testExitCode
