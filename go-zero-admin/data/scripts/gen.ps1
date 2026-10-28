#requires -Version 5.1
# go-zero-admin 代码生成入口（api / rpc / model / swagger）
#
#   .\data\scripts\gen.ps1 api   <module>             生成 API 代码（使用项目自定义模板）
#   .\data\scripts\gen.ps1 rpc   <module> <proto>      生成 RPC 代码
#   .\data\scripts\gen.ps1 model <module> <table>      从 MySQL 表生成 Model 代码
#   .\data\scripts\gen.ps1 swagger [-ApiHost host]    生成 Swagger 文档
#
# 前提：已安装 goctl v1.10.2（go install github.com/zeromicro/go-zero/tools/goctl@v1.10.2）；
#       rpc 另需 protobuf 工具。在项目根目录或任意目录均可运行。
# 注意：生成后请检查差异，保留已有的自定义业务逻辑。
[CmdletBinding()]
param(
    [Parameter(Position = 0)][ValidateSet('api', 'rpc', 'model', 'swagger')][string]$Command,
    [Parameter(Position = 1)][string]$Name,
    [Parameter(Position = 2)][string]$Target,
    [string]$GoctlPath,
    [string]$ApiHost = 'localhost:7001'
)

$ErrorActionPreference = 'Stop'
$projectRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))

function Join-Parts([string[]]$Parts) {
    $joined = $Parts[0]
    for ($index = 1; $index -lt $Parts.Count; $index++) { $joined = Join-Path $joined $Parts[$index] }
    return $joined
}

function Invoke-ApiGen {
    if (-not $Name) { throw 'Usage: gen.ps1 api <module>   (example: gen.ps1 api applet)' }
    foreach ($template in @('data/goctl/api/handler.tpl', 'data/goctl/api/main.tpl')) {
        if (-not (Test-Path -LiteralPath $template -PathType Leaf)) { throw ('Custom template not found: ' + $template) }
    }
    Write-Host ('Generating API code for ' + $Name + '...')
    $apiFile = Join-Parts @('application', $Name, 'api', 'desc', ($Name + '.api'))
    $apiDir = Join-Parts @('application', $Name, 'api')
    & goctl api go -api $apiFile -dir $apiDir -home data/goctl -style=go_zero
    if ($LASTEXITCODE -ne 0) { throw 'goctl api generation failed.' }
}

function Invoke-RpcGen {
    if (-not $Name -or -not $Target) { throw 'Usage: gen.ps1 rpc <module> <proto>   (example: gen.ps1 rpc applet applet)' }
    Write-Host ('Generating RPC code for ' + $Name + '/' + $Target + '...')
    $protoFile = Join-Parts @('application', $Name, 'rpc', 'desc', ($Target + '.proto'))
    $rpcDir = Join-Parts @('application', $Name, 'rpc')
    & goctl rpc protoc $protoFile --go_out=$rpcDir --go-grpc_out=$rpcDir --zrpc_out=$rpcDir -m --style=go_zero --name-from-filename
    if ($LASTEXITCODE -ne 0) { throw 'goctl rpc generation failed.' }
}

function Invoke-ModelGen {
    if (-not $Name -or -not $Target) { throw 'Usage: gen.ps1 model <module> <table>   (example: gen.ps1 model applet sys_users)' }
    Write-Host ('Generating Model code for ' + $Name + '.' + $Target + '...')
    $modelDir = Join-Parts @('application', $Name, 'rpc', 'internal', 'model')
    $dsn = 'root:123456@tcp(127.0.0.1:3306)/goZero-admin'
    & goctl model mysql datasource --dir $modelDir --table $Target --cache true --url=$dsn
    if ($LASTEXITCODE -ne 0) { throw 'goctl model generation failed.' }
}

function Invoke-SwaggerGen {
$projectRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
$sourceDir = Join-Path $projectRoot 'application/applet/api/desc'
$outputFile = Join-Path $projectRoot 'data/api/generated/go-zero-admin.swagger.json'
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)

# Windows PowerShell 5.1 has no ConvertFrom-Json -AsHashtable. Keep arrays
# (including empty/single-item arrays) intact while converting JSON objects.
function ConvertTo-JsonMap($value) {
    if ($value -is [System.Collections.IDictionary]) {
        $result = @{}
        foreach ($key in $value.PSBase.Keys) { $result[$key] = ConvertTo-JsonMap $value[$key] }
        return $result
    }
    if ($value -is [System.Management.Automation.PSCustomObject]) {
        $result = @{}
        foreach ($property in $value.PSObject.Properties) { $result[$property.Name] = ConvertTo-JsonMap $property.Value }
        return $result
    }
    if ($value -is [array]) {
        $result = New-Object 'object[]' $value.Length
        for ($i = 0; $i -lt $value.Length; $i++) { $result[$i] = ConvertTo-JsonMap $value[$i] }
        return ,$result
    }
    return $value
}

function Read-JsonMap([string]$path) {
    return ConvertTo-JsonMap (ConvertFrom-Json -InputObject ([IO.File]::ReadAllText($path)))
}

if (-not $GoctlPath) {
    $command = Get-Command goctl -ErrorAction SilentlyContinue
    if ($command) { $GoctlPath = $command.Source }
    else {
        $goBin = (& go env GOBIN).Trim()
        if (-not $goBin) { $goBin = Join-Path ((& go env GOPATH).Trim().Split([IO.Path]::PathSeparator)[0]) 'bin' }
        $onWindows = [Environment]::OSVersion.Platform -eq [PlatformID]::Win32NT
        $GoctlPath = Join-Path $goBin $(if ($onWindows) { 'goctl.exe' } else { 'goctl' })
    }
}
if (-not (Test-Path -LiteralPath $GoctlPath -PathType Leaf)) {
    throw 'goctl not found. Install: go install github.com/zeromicro/go-zero/tools/goctl@v1.10.2'
}
$generatorVersion = (& $GoctlPath --version | Out-String).Trim()
if ($LASTEXITCODE -ne 0) { throw 'Cannot read goctl version.' }

# Only temporary documentation copies are annotated; application sources stay untouched.
$tempRoot = Join-Path ([IO.Path]::GetTempPath()) ('go-zero-swagger-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tempRoot | Out-Null
try {
    Copy-Item -LiteralPath $sourceDir -Destination (Join-Path $tempRoot 'desc') -Recurse
    $entry = Join-Path $tempRoot 'swagger.api'
    $entryContent = @'
syntax = "v1"
info (
    title: "go-zero-admin API"
    version: "v1"
    useDefinitions: true
    securityDefinitionsFromJson: `{"BearerAuth":{"type":"apiKey","name":"Authorization","in":"header","description":"Enter Bearer <accessToken>"}}`
)
import "./desc/applet.api"
'@
    [IO.File]::WriteAllText($entry, $entryContent, $utf8NoBom)

    $apiCopies = @(Get-ChildItem -LiteralPath (Join-Path $tempRoot 'desc') -Filter '*.api' -Recurse)
    $casbinRoutes = @{}
    foreach ($file in $apiCopies) {
        $content = [IO.File]::ReadAllText($file.FullName)
        # Mark actual Authority-protected routes, not every JWT-only account route.
        foreach ($service in [regex]::Matches($content, '(?s)@server\s*\((.*?)\)\s*service\s+[^\s{]+\s*\{(.*?)\}')) {
            $annotation = $service.Groups[1].Value
            $prefixMatch = [regex]::Match($annotation, '(?m)^\s*prefix\s*:\s*([^\s]+)')
            $prefix = $prefixMatch.Groups[1].Value.TrimEnd('/')
            $protected = $annotation -match '(?m)^\s*middleware\s*:[^\r\n]*\bAuthority\b'
            foreach ($route in [regex]::Matches($service.Groups[2].Value, '(?m)^\s*(get|post|put|delete|patch|head|options)\s+([^\s(]+)')) {
                $casbinRoutes[$route.Groups[1].Value + ' ' + $prefix + $route.Groups[2].Value] = $protected
            }
        }
        $content = [regex]::Replace($content, '(?s)@server\s*\((.*?)\)', {
            param($match)
            $annotation = $match.Groups[1].Value
            if ($annotation -match '(?m)^\s*jwt\s*:' -and $annotation -notmatch '\bauthType\s*:') {
                $annotation += "`n    authType: BearerAuth`n"
            }
            if ($annotation -notmatch '\btags\s*:' -and $annotation -match '(?m)^\s*group\s*:\s*([^\s]+)') {
                $annotation += "`n    tags: $($Matches[1])`n"
            }
            return '@server(' + $annotation + ')'
        })
        [IO.File]::WriteAllText($file.FullName, $content, $utf8NoBom)
    }

    # Keep parameters from the original form/json tags.
    & $GoctlPath api swagger --api $entry --dir $tempRoot --filename parameters
    if ($LASTEXITCODE -ne 0) { throw 'goctl parameter generation failed.' }
    $document = Read-JsonMap (Join-Path $tempRoot 'parameters.json')

    # goctl omits dual form/json fields from JSON definitions. Generate those models
    # once more from a documentation copy with only the redundant form tag removed.
    foreach ($file in $apiCopies) {
        $content = [IO.File]::ReadAllText($file.FullName)
        $content = [regex]::Replace($content, '`[^`\r\n]*`', {
            param($match)
            if ($match.Value -match '\bjson:"') {
                return [regex]::Replace($match.Value, '\s+form:"[^"]*"', '')
            }
            return $match.Value
        })
        [IO.File]::WriteAllText($file.FullName, $content, $utf8NoBom)
    }
    & $GoctlPath api swagger --api $entry --dir $tempRoot --filename models
    if ($LASTEXITCODE -ne 0) { throw 'goctl model generation failed.' }
    $models = Read-JsonMap (Join-Path $tempRoot 'models.json')
    $document.definitions = $models.definitions

    # $ref siblings emitted by goctl are redundant; json:"-" must never be a field.
    function Repair-Schema($value) {
        if ($value -is [System.Collections.IDictionary]) {
            if ($value.Contains('$ref')) {
                foreach ($key in @($value.PSBase.Keys)) { if ($key -ne '$ref') { $value.Remove($key) } }
            }
            if ($value.properties -is [System.Collections.IDictionary]) { $value.properties.Remove('-') }
            if ($value.required -is [array]) {
                $value.required = @($value.required | Where-Object { $_ -ne '-' } | Select-Object -Unique)
                if ($value.required.Count -eq 0) { $value.Remove('required') }
            }
            foreach ($key in @($value.PSBase.Keys)) { Repair-Schema $value[$key] }
        } elseif ($value -is [array]) {
            foreach ($item in $value) { Repair-Schema $item }
        }
    }
    Repair-Schema $document
    $document.Remove('x-date') # A generation timestamp would change every run.
    $document.host = $ApiHost
    $document.schemes = @('http', 'https')
    $document.info.description = 'Generated from current .api definitions by data/scripts/gen.ps1. Success: code/message/result/returnData/success/timestamp. Business errors may use HTTP 200 with code/message. GET parameters use query strings. Login requires the captchaId and captcha from a fresh image; a captcha expires after 120 seconds and is consumed on submission.'
    $document['x-generator'] = $generatorVersion
    $document.definitions.BusinessError = @{
        type = 'object'; required = @('code', 'message')
        properties = @{ code = @{ type = 'integer' }; message = @{ type = 'string' } }
    }
    $document.definitions.MiddlewareError = @{
        type = 'object'; required = @('message')
        properties = @{ message = @{ type = 'string' } }
    }
    $count = 0
    foreach ($path in $document.paths.PSBase.Keys) {
        foreach ($method in $document.paths[$path].PSBase.Keys) {
            $operation = $document.paths[$path][$method]
            # Swagger uses {id}; go-zero and Casbin use :id.
            $casbinPath = [regex]::Replace($path, '\{([A-Za-z_][A-Za-z0-9_]*)\}', ':$1')
            $routeKey = $method + ' ' + $casbinPath
            if (-not $casbinRoutes.ContainsKey($routeKey)) { throw "Missing middleware metadata: $routeKey" }
            $operation['x-casbin-resource'] = $casbinRoutes[$routeKey]
            $operation['x-identity-domain'] = if ($path.StartsWith('/v1/app/')) { 'app' } else { 'admin' }
            $operation.Remove('schemes') # Inherit the configured HTTP/HTTPS schemes.
            $body = @($operation.parameters | Where-Object { $_.in -eq 'body' })
            if ($body.Count -gt 0) {
                # Swagger 2 cannot combine body and formData. JSON contains both sets
                # of fields and is the documented representation for these endpoints.
                $operation.parameters = @($operation.parameters | Where-Object { $_.in -ne 'formData' })
                $operation.consumes = @('application/json')
                if ($method -eq 'get') {
                    # Dual-tag fields are already documented as query parameters.
                    $bodySchema = ConvertTo-JsonMap $document.definitions[$body[0].schema['$ref'].Split('/')[-1]]
                    $queryNames = @($operation.parameters | Where-Object { $_.in -eq 'query' } | ForEach-Object { $_.name })
                    foreach ($name in $queryNames) { $bodySchema.properties.Remove($name) }
                    $required = @($bodySchema.required | Where-Object { $_ -and $_ -notin $queryNames })
                    if ($required.Count) { $bodySchema.required = $required } else { $bodySchema.Remove('required') }
                    $body[0].schema = $bodySchema
                    $body[0].required = $required.Count -gt 0
                    $operation.description = '当前 .api 存在仅使用 json 标签的 GET 字段，浏览器不能发送 GET body。form 标签对应字段已列为 query；可选 body 可省略，必填 body 对应的接口需先修正后端参数定义再联调。'
                }
            }
            $response = $operation.responses['200']
            $resultSchema = $response.schema
            if (-not $resultSchema) { $resultSchema = @{} }
            $response.description = 'Success: code=200. Business errors can also return HTTP 200 with only code/message; check code before reading result.'
            $response.schema = @{
                type = 'object'
                required = @('code', 'message')
                properties = @{
                    code = @{ type = 'integer'; example = 200 }
                    message = @{ type = 'string'; example = '操作成功!' }
                    result = $resultSchema
                    returnData = @{ description = 'Currently null'; 'x-nullable' = $true }
                    success = @{ type = 'boolean'; example = $true }
                    timestamp = @{ type = 'integer'; format = 'int64'; description = 'Unix time in seconds' }
                }
            }
            $response['x-business-error-schema'] = @{ '$ref' = '#/definitions/BusinessError' }
            if ($operation.security) {
                $response.description += ' JWT failure also uses HTTP 200, code=100003.'
                $operation.responses['403'] = @{ description = '接口权限不足'; schema = @{ '$ref' = '#/definitions/MiddlewareError' } }
                $operation.responses['500'] = @{ description = '鉴权服务异常'; schema = @{ '$ref' = '#/definitions/MiddlewareError' } }
            }
            $count++
        }
    }

    $upload = $document.paths['/v1/sys/base/uploadFileImg'].post
    $upload.consumes = @('multipart/form-data')
    $upload.parameters = @(
        @{ name = 'file_img'; in = 'formData'; type = 'file'; required = $true; description = 'PNG/JPEG/GIF/WebP image, up to 10MB; stored in RustFS.' }
        @{ name = 'visibility'; in = 'formData'; type = 'string'; required = $false; enum = @('public', 'private'); default = 'public'; description = 'Public files use a permanent URL; private files require a short-lived signed URL.' }
    )
    $document.paths['/v1/sys/register'].post.responses['200'].schema.properties.result = @{
        description = 'Current registration logic returns null on success.'; 'x-nullable' = $true
    }

    # Sort JSON object keys so repeated generation does not introduce noisy diffs.
    function Sort-JsonObject($value) {
        if ($value -is [System.Collections.IDictionary]) {
            $sorted = [ordered]@{}
            foreach ($key in ($value.PSBase.Keys | Sort-Object -CaseSensitive)) {
                if ($key -isnot [string]) { throw "Non-string JSON key: $key ($($key.GetType().FullName))" }
                $sorted[$key] = Sort-JsonObject $value[$key]
            }
            return $sorted
        }
        if ($value -is [array]) { return ,@($value | ForEach-Object { Sort-JsonObject $_ }) }
        return $value
    }

    function Copy-JsonMap($value) {
        # Rebuild arrays before cloning: WinPS 5.1 may otherwise serialize
        # ConvertFrom-Json's array wrapper as a {Count,value} object.
        return ConvertTo-JsonMap (ConvertFrom-Json -InputObject (Sort-JsonObject $value | ConvertTo-Json -Depth 100))
    }

    # Produce separate audiences from the same source contracts. The RPC embeds
    # only the admin file; App operations never enter its permission catalogue.
    function Write-AudienceDocument([string]$audience, [string]$destination) {
        $copy = Copy-JsonMap $document
        foreach ($path in @($copy.paths.PSBase.Keys)) {
            if ($path.StartsWith('/v1/app/') -ne ($audience -eq 'app')) { $copy.paths.Remove($path) }
        }
        $copy['x-identity-domain'] = $audience
        if ($audience -eq 'app') {
            $copy.info.title = 'go-zero App API'
            $copy.info.description = 'Email/phone + password registration creates a pending account. Administrator approval is required before login. App access and rotating refresh tokens are independent from administrator tokens. Membership grants are managed separately. Inspect the response code before result.'
        }
        # Keep only schemas reachable from this audience's operations.
        $reachable = @{}
        $queue = New-Object 'Collections.Generic.Queue[string]'
        foreach ($ref in [regex]::Matches(($copy.paths | ConvertTo-Json -Depth 100), '#/definitions/([^"\s]+)')) { $queue.Enqueue($ref.Groups[1].Value) }
        while ($queue.Count -gt 0) {
            $name = $queue.Dequeue()
            if ($reachable.ContainsKey($name)) { continue }
            $reachable[$name] = $true
            if (-not $copy.definitions.ContainsKey($name)) { throw "Missing schema: $name" }
            foreach ($ref in [regex]::Matches(($copy.definitions[$name] | ConvertTo-Json -Depth 100), '#/definitions/([^"\s]+)')) { $queue.Enqueue($ref.Groups[1].Value) }
        }
        foreach ($name in @($copy.definitions.PSBase.Keys)) { if (-not $reachable.ContainsKey($name)) { $copy.definitions.Remove($name) } }
        New-Item -ItemType Directory -Path (Split-Path $destination) -Force | Out-Null
        $json = Sort-JsonObject $copy | ConvertTo-Json -Depth 100
        $existingJson = if (Test-Path -LiteralPath $destination) { Sort-JsonObject (Read-JsonMap $destination) | ConvertTo-Json -Depth 100 }
        if ($json -cne $existingJson) { [IO.File]::WriteAllText($destination, $json + [Environment]::NewLine, $utf8NoBom) }
        Write-Output "Generated $audience API: $destination ($generatorVersion)"
    }
    Write-AudienceDocument 'admin' $outputFile
    Write-AudienceDocument 'app' (Join-Path $projectRoot 'data/api/generated/go-zero-app.swagger.json')
} finally {
    # Only remove the unique temporary directory created by this invocation.
    $resolvedTemp = [IO.Path]::GetFullPath($tempRoot)
    $tempParent = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    if (-not $resolvedTemp.StartsWith($tempParent, [StringComparison]::OrdinalIgnoreCase) -or (Split-Path $resolvedTemp -Leaf) -notlike 'go-zero-swagger-*') {
        throw "Unsafe temporary directory: $resolvedTemp"
    }
    Remove-Item -LiteralPath $resolvedTemp -Recurse -Force
}
}

Push-Location $projectRoot
try {
    switch ($Command) {
        'api' { Invoke-ApiGen }
        'rpc' { Invoke-RpcGen }
        'model' { Invoke-ModelGen }
        'swagger' { Invoke-SwaggerGen }
        default {
            Write-Host 'Usage: gen.ps1 <api|rpc|model|swagger> [args...]'
            Write-Host '  api   <module>          example: gen.ps1 api applet'
            Write-Host '  rpc   <module> <proto>  example: gen.ps1 rpc applet applet'
            Write-Host '  model <module> <table>  example: gen.ps1 model applet sys_users'
            Write-Host '  swagger                 example: gen.ps1 swagger'
            exit 1
        }
    }
} finally {
    Pop-Location
}
