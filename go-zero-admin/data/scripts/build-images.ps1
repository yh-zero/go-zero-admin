# 交叉编译 Linux 二进制到 docker/.build/（离线环境：Docker Hub 不可达，替代 Dockerfile 内构建）
# 用法：powershell -NoProfile -ExecutionPolicy Bypass -File .\data\scripts\build-images.ps1
# 产出后执行：docker compose up -d --build
$ErrorActionPreference = 'Stop'
$backend = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$out = Join-Path $backend 'docker/.build'
New-Item -ItemType Directory -Force $out | Out-Null

$env:GOOS = 'linux'; $env:GOARCH = 'amd64'; $env:CGO_ENABLED = '0'
$targets = [ordered]@{
  'api'        = './application/admin/api'
  'admin-rpc'  = './application/admin/rpc'
  'app-rpc'    = './application/app/rpc'
  'file-rpc'   = './application/file/rpc'
  'ai-rpc'     = './application/ai/rpc'
  'tenant-rpc' = './application/tenant/rpc'
}
Push-Location $backend
try {
  foreach ($t in $targets.GetEnumerator()) {
    Write-Host "build $($t.Key) ..."
    go build -ldflags='-s -w' -o (Join-Path $out $t.Key) $t.Value
    if ($LASTEXITCODE -ne 0) { throw "go build $($t.Key) 失败" }
  }
} finally {
  Pop-Location
  Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
}
Write-Host "done -> docker/.build/"
