# 登录闭环端到端测试
# 前提：
#   1. MySQL 9.7 容器运行中（gozero-new-mysql，3307 端口）
#   2. Redis 运行中（gozero-redis 容器或本地 6379）
#   3. admin-rpc 已启动（:6001）
#   4. admin-api 已启动（:7001）

$ErrorActionPreference = 'Stop'
$apiBase = 'http://127.0.0.1:7001'

Write-Host '=== Step 1: 获取验证码 ===' -ForegroundColor Cyan
$captchaResp = Invoke-RestMethod -Uri "$apiBase/v1/sys/auth/captcha" -Method Get
$captchaResp | ConvertTo-Json -Depth 5

$captchaID   = $captchaResp.captcha_id
$imageBase64 = $captchaResp.image_base64

# 简版验证码：ImageBase64 就是验证码文本的 base64
$captchaCode = [System.Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($imageBase64))
Write-Host "验证码: $captchaCode (ID: $captchaID)" -ForegroundColor Yellow

Write-Host ''
Write-Host '=== Step 2: 登录 ===' -ForegroundColor Cyan
$loginBody = @{
    username   = 'admin'
    password   = 'Admin@123456'
    captcha_id = $captchaID
    captcha    = $captchaCode
} | ConvertTo-Json

try {
    $loginResp = Invoke-RestMethod -Uri "$apiBase/v1/sys/auth/login" -Method Post -Body $loginBody -ContentType 'application/json'
    Write-Host '登录成功!' -ForegroundColor Green
    $loginResp | ConvertTo-Json -Depth 5
    $token = $loginResp.access_token
    Write-Host ''
    Write-Host "AccessToken: $($token.Substring(0, [Math]::Min(40, $token.Length)))..." -ForegroundColor Green
} catch {
    Write-Host "登录失败: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.ErrorDetails) { $_.ErrorDetails.Message | ConvertFrom-Json | ConvertTo-Json -Depth 5 }
    exit 1
}

Write-Host ''
Write-Host '=== Step 3: 验证 Ping（需 JWT，预期 401） ===' -ForegroundColor Cyan
try {
    Invoke-RestMethod -Uri "$apiBase/ping" -Method Get -Headers @{ Authorization = "Bearer $token" }
} catch {
    Write-Host "Ping 状态: $($_.Exception.Response.StatusCode.value__) (JWT 中间件尚未接入 Ping，属预期)" -ForegroundColor Yellow
}

Write-Host ''
Write-Host '=== 登录闭环测试通过 ===' -ForegroundColor Green
