#requires -Version 5.1
# go-zero-admin 提交前自检（AGENTS.md 第 4 节）
#   .\data\scripts\verify.ps1
[CmdletBinding()] param(
    [switch]$AllowYamlSecrets  # 私有仓库部署允许 yaml 明文密钥时显式传入（公开主线禁止）
)
$ErrorActionPreference = 'Stop'
$projectRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
Push-Location $projectRoot
try {
    Write-Host '[1/7] gofmt 检查（CRLF 工作区经临时副本归一为 LF 后比较） ...'
    # gofmt 输出为 LF，CRLF 检出的工作区会被整体误判；归一临时副本只用于只读比较
    $tmp = Join-Path $env:TEMP ('gofmt-' + [guid]::NewGuid().ToString('N'))
    try {
        foreach ($d in @('application', 'pkg')) {
            robocopy $d (Join-Path $tmp $d) /E /NFL /NDL /NJH /NJS | Out-Null
            if ($LASTEXITCODE -ge 8) { throw ('robocopy 拷贝 ' + $d + ' 失败，退出码 ' + $LASTEXITCODE) }
        }
        Get-ChildItem -Recurse -Filter *.go $tmp | ForEach-Object {
            $t = [IO.File]::ReadAllText($_.FullName)
            if ($t.Contains("`r`n")) { [IO.File]::WriteAllText($_.FullName, ($t -replace "`r`n", "`n")) }
        }
        Push-Location $tmp
        try {
            $dirty = @(gofmt -l application pkg 2>$null)
            if ($dirty.Count -gt 0) { throw ('gofmt 未格式化: ' + ($dirty -join ', ')) }
        } finally {
            Pop-Location
        }
    } finally {
        $resolvedTempRoot = [IO.Path]::GetFullPath($env:TEMP).TrimEnd('\') + '\'
        $resolvedTempTarget = [IO.Path]::GetFullPath($tmp)
        if (-not $resolvedTempTarget.StartsWith($resolvedTempRoot + 'gofmt-', [StringComparison]::OrdinalIgnoreCase)) {
            throw '临时格式检查目录越界，拒绝清理。'
        }
        Remove-Item -LiteralPath $resolvedTempTarget -Recurse -Force -ErrorAction SilentlyContinue
    }

    $prevEap = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'   # PS5.1：原生命令 stderr 警告在 Stop 下会误抛
    $pkgs = @(& go list ./... 2>$null)
    $listExit = $LASTEXITCODE
    $ErrorActionPreference = $prevEap
    if ($listExit -ne 0) { throw 'go list ./... 失败：不得当作骨架阶段静默跳过，先修复模块加载错误。' }
    if ($pkgs.Count -eq 0) { throw 'go list ./... 返回空包列表：模块结构异常，不得静默跳过。' }
    Write-Host '[2/7] go vet ...'
    & go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'go vet failed.' }

    Write-Host '[3/7] go build ./... ...'
    & go build ./...
    if ($LASTEXITCODE -ne 0) { throw 'go build failed.' }

    Write-Host '[4/7] go mod tidy -diff（检测未批准依赖） ...'
    & go mod tidy -diff
    if ($LASTEXITCODE -ne 0) { throw 'go.mod/go.sum 不整洁：可能引入了未批准依赖，见 AGENTS.md 第 2.5 条。' }

    Write-Host '[5/7] goctl api validate ...'
    & goctl api validate --api application/admin/api/desc/admin.api
    if ($LASTEXITCODE -ne 0) { throw 'admin.api 校验失败。' }

    Write-Host '[6/7] 顶层目录白名单 ...'
    $allow = @('application', 'data', 'docker', 'pkg', 'test', '.agents', '.claude', '.github', '.git', '.dsh-build')
    $extra = @(Get-ChildItem -Directory | Where-Object { $allow -notcontains $_.Name } | ForEach-Object { $_.Name })
    if ($extra.Count -gt 0) { throw ('未批准的顶层目录: ' + ($extra -join ', ') + '（见 AGENTS.md 第 2.4 条）') }

    Write-Host '[7/7] yaml 密钥扫描 ...'
    if (-not $AllowYamlSecrets) {
        # 本地开发约定的非真实凭据（公开占位值），其余 Key/Secret/Password 字段只允许空值或 ${ENV} 占位符
        $devDefaults = @(
            'go-zero-admin-local-development-jwt-secret',
            'go-zero-app-local-development-secret-change-me',
            'GOZEROLOCALROOT',
            'go-zero-local-storage-root-change-me',
            'Admin@123456',
            'Demo@123456'
        )
        $hits = @()
        $yamlFiles = @(Get-ChildItem -Recurse -Filter *.yaml application, docker\etc | Where-Object { $_.FullName -match '\\etc\\' })
        foreach ($yf in $yamlFiles) {
            $n = 0
            foreach ($line in [IO.File]::ReadAllLines($yf.FullName)) {
                $n++
                if ($line -match '^\s*[A-Za-z_]*(?:Key|Secret|Password)[A-Za-z_]*\s*:\s*([^#]+?)\s*(?:#.*)?$') {
                    $val = $Matches[1].Trim().Trim('"').Trim("'")
                    if ($val -eq '' -or $val -match '^\$\{[A-Z0-9_]+\}$' -or $devDefaults -contains $val) { continue }
                    $hits += ($yf.FullName + ':' + $n)
                }
            }
        }
        if ($hits.Count -gt 0) { throw ("检测到 yaml 明文密钥：`n" + ($hits -join "`n") + "`n公开仓库禁止提交真实密钥；私有仓库部署请执行 verify.ps1 -AllowYamlSecrets") }
    }

    Write-Host 'verify: 全部通过 ✔'
} finally {
    Pop-Location
}
