#requires -Version 5.1
# go-zero-admin 代码生成入口（api / rpc / model / db / swagger）
#
#   .\data\scripts\gen.ps1 api                  生成 API 代码（官方默认模板，无 -home）
#   .\data\scripts\gen.ps1 rpc <module>         生成 RPC 代码（admin|app|file|ai|tenant）
#   .\data\scripts\gen.ps1 model <module> <table>  生成数据层（goctl model，带缓存；需先 db up）
#   .\data\scripts\gen.ps1 db up                执行 goose 迁移
#   .\data\scripts\gen.ps1 db create <name>     新建迁移文件
#   .\data\scripts\gen.ps1 swagger              生成 Swagger（官方原生输出）
#
# 前提：goctl v1.10.2、goose 已安装，本地开发库已启动。生成后请检查差异，保留手写业务逻辑。
[CmdletBinding()]
param(
    [Parameter(Position = 0)][ValidateSet('api', 'rpc', 'model', 'db', 'swagger')][string]$Command,
    [Parameter(Position = 1)][string]$Name,
    [Parameter(Position = 2)][string]$Arg,
    [string]$Dsn = 'root:123456@tcp(127.0.0.1:3307)/gozero_admin'
)

$ErrorActionPreference = 'Stop'
$projectRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
$modules = @('admin', 'app', 'file', 'ai', 'tenant')

Push-Location $projectRoot
try {
    switch ($Command) {
        'api' {
            & goctl api go -api application/admin/api/desc/admin.api -dir application/admin/api -style go_zero
            if ($LASTEXITCODE -ne 0) { throw 'goctl api generation failed.' }
        }
        'rpc' {
            if ($modules -notcontains $Name) { throw ('Usage: gen.ps1 rpc <module>  (' + ($modules -join '|') + ')') }
            $rpcDir = 'application/' + $Name + '/rpc'
            $proto = Join-Path $rpcDir ('desc/' + $Name + '.proto')
            if (-not (Test-Path -LiteralPath $proto -PathType Leaf)) { throw ('proto not found: ' + $proto) }

            # 生成前记录 pb.go 中已有的 Register 调用
            $mainFile = Join-Path $rpcDir ($Name + '.go')
            if (-not (Test-Path $mainFile)) { $mainFile = Join-Path $rpcDir 'pb.go' }
            $before = @{}
            if (Test-Path $mainFile) {
                foreach ($m in [regex]::Matches((Get-Content $mainFile -Raw), 'Register(w+)Server')) { $before[$m.Groups[1].Value] = $true }
            }

            & goctl rpc protoc $proto --go_out=$rpcDir --go-grpc_out=$rpcDir --zrpc_out=$rpcDir -m --style go_zero --name-from-filename
            if ($LASTEXITCODE -ne 0) { throw 'goctl rpc generation failed.' }

            # goctl 对已存在的 main 文件跳过生成——新增 service 需手动注册，此处检测并告警
            if (Test-Path $mainFile) {
                $content = Get-Content $mainFile -Raw
                $protoContent = Get-Content $proto -Raw
                $svcPattern = [regex]'(?m)^service\s+(\w+)'
                $allServices = @($svcPattern.Matches($protoContent) | ForEach-Object { $_.Groups[1].Value })
                $missing = @($allServices | Where-Object { $content.IndexOf('Register' + $_ + 'Server') -lt 0 })
                if ($missing.Count -gt 0) {
                    Write-Host ''
                    Write-Host ('[警告] 以下 service 未在 ' + (Split-Path $mainFile -Leaf) + ' 注册（goctl 对已有 main 文件跳过生成）：') -ForegroundColor Yellow
                    foreach ($s in $missing) {
                        $pkg = $s.Substring(0,1).ToLower() + $s.Substring(1)
                        Write-Host ('  pb.Register' + $s + 'Server(grpcServer, ' + $pkg + 'Server.New' + $s + 'Server(ctx))') -ForegroundColor Yellow
                    }
                    Write-Host '请手动添加上述注册行后重新构建。' -ForegroundColor Yellow
                }
            }
        }
        'model' {
            if ($modules -notcontains $Name) { throw ('Usage: gen.ps1 model <module> <table>  (' + ($modules -join '|') + ')') }
            if (-not $Arg) { throw 'Usage: gen.ps1 model <module> <table>' }
            $modelDir = 'application/' + $Name + '/rpc/internal/model'
            & goctl model mysql datasource --dir $modelDir --table $Arg --cache true -style go_zero --url $Dsn
            if ($LASTEXITCODE -ne 0) { throw 'goctl model generation failed.' }
        }
        'db' {
            # 迁移集：business（租户业务库，默认）/ platform（平台控制库）
            $set = if ($Arg -and $Name -ne 'create') { $Arg } elseif ($Name -eq 'create' -and $env:DB_SET) { $env:DB_SET } else { 'business' }
            if ($Name -eq 'up') {
                $setDir = 'data/db/migrations/' + $(if ($Arg) { $Arg } else { 'business' })
                & goose -dir $setDir mysql $Dsn up
            } elseif ($Name -eq 'create') {
                if (-not $Arg) { throw 'Usage: gen.ps1 db create <name> [business|platform]' }
                $createSet = 'business'
                & goose -dir ('data/db/migrations/' + $createSet) create $Arg sql
            } else {
                throw 'Usage: gen.ps1 db up [business|platform] | db create <name>'
            }
            if ($LASTEXITCODE -ne 0) { throw 'goose failed.' }
        }
        'swagger' {
            & goctl api swagger --api application/admin/api/desc/admin.api --dir data/api/generated
            if ($LASTEXITCODE -ne 0) { throw 'goctl swagger generation failed.' }
        }
        default {
            Write-Host 'Usage: gen.ps1 <api|rpc|model|db|swagger> [args...]'
            exit 1
        }
    }
} finally {
    Pop-Location
}
