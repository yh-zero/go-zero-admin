#requires -Version 5.1
# go-zero-admin 提交前自检（AGENTS.md 第 4 节）
#   .\data\scripts\verify.ps1
[CmdletBinding()] param()
$ErrorActionPreference = 'Stop'
$projectRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
Push-Location $projectRoot
try {
    Write-Host '[1/6] gofmt 检查 ...'
    $dirty = @(gofmt -l application pkg 2>$null)
    if ($dirty.Count -gt 0) { throw ('gofmt 未格式化: ' + ($dirty -join ', ')) }

    $prevEap = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'   # PS5.1：原生命令 stderr 警告在 Stop 下会误抛
    $pkgs = @(& go list ./... 2>$null)
    $ErrorActionPreference = $prevEap
    if ($pkgs.Count -eq 0) {
        Write-Host '[2/6] go vet —— 跳过（骨架阶段，尚无 Go 包）'
        Write-Host '[3/6] go build —— 跳过（骨架阶段，尚无 Go 包）'
        Write-Host '[4/6] go mod tidy -diff —— 跳过（骨架阶段，尚无 Go 包）'
    } else {
        Write-Host '[2/6] go vet ...'
        & go vet ./...
        if ($LASTEXITCODE -ne 0) { throw 'go vet failed.' }

        Write-Host '[3/6] go build ./... ...'
        & go build ./...
        if ($LASTEXITCODE -ne 0) { throw 'go build failed.' }

        Write-Host '[4/6] go mod tidy -diff（检测未批准依赖） ...'
        & go mod tidy -diff
        if ($LASTEXITCODE -ne 0) { throw 'go.mod/go.sum 不整洁：可能引入了未批准依赖，见 AGENTS.md 第 2.5 条。' }
    }

    Write-Host '[5/6] goctl api validate ...'
    & goctl api validate --api application/admin/api/desc/admin.api
    if ($LASTEXITCODE -ne 0) { throw 'admin.api 校验失败。' }

    Write-Host '[6/6] 顶层目录白名单 ...'
    $allow = @('application', 'data', 'docker', 'pkg', 'test', '.agents', '.claude', '.github', '.git', '.dsh-build')
    $extra = @(Get-ChildItem -Directory | Where-Object { $allow -notcontains $_.Name } | ForEach-Object { $_.Name })
    if ($extra.Count -gt 0) { throw ('未批准的顶层目录: ' + ($extra -join ', ') + '（见 AGENTS.md 第 2.4 条）') }

    Write-Host 'verify: 全部通过 ✔'
} finally {
    Pop-Location
}
