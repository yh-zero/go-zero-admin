#requires -Version 5.1
[CmdletBinding()]
param(
    [string]$GoctlPath,
    [string]$ApiHost = 'localhost:7001'
)

$ErrorActionPreference = 'Stop'
$projectRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
$sourceDir = Join-Path $projectRoot 'application/applet/api/desc'
$outputFile = Join-Path $projectRoot 'data/api/generated/go-zero-admin.swagger.json'
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)

# Windows PowerShell 5.1 has no ConvertFrom-Json -AsHashtable. Keep arrays
# (including empty/single-item arrays) intact while converting JSON objects.
function ConvertTo-JsonMap($value) {
    if ($value -is [System.Collections.IDictionary]) {
        $result = @{}
        foreach ($key in $value.Keys) { $result[$key] = ConvertTo-JsonMap $value[$key] }
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
    foreach ($file in $apiCopies) {
        $content = [IO.File]::ReadAllText($file.FullName)
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
                foreach ($key in @($value.Keys)) { if ($key -ne '$ref') { $value.Remove($key) } }
            }
            if ($value.properties -is [System.Collections.IDictionary]) { $value.properties.Remove('-') }
            if ($value.required -is [array]) {
                $value.required = @($value.required | Where-Object { $_ -ne '-' } | Select-Object -Unique)
                if ($value.required.Count -eq 0) { $value.Remove('required') }
            }
            foreach ($key in @($value.Keys)) { Repair-Schema $value[$key] }
        } elseif ($value -is [array]) {
            foreach ($item in $value) { Repair-Schema $item }
        }
    }
    Repair-Schema $document
    $document.Remove('x-date') # A generation timestamp would change every run.
    $document.host = $ApiHost
    $document.schemes = @('http', 'https')
    $document.info.description = 'Generated from current .api definitions by test/sh/swagger.ps1. Success: code/message/result/returnData/success/timestamp. Business errors may use HTTP 200 with code/message. GET parameters use query strings. Login requires the captchaId and captcha from a fresh image; a captcha expires after 120 seconds and is consumed on submission.'
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
    foreach ($path in $document.paths.Keys) {
        foreach ($method in $document.paths[$path].Keys) {
            $operation = $document.paths[$path][$method]
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
    $upload.parameters = @(@{ name = 'file_img'; in = 'formData'; type = 'file'; required = $true; description = 'Image file; requires configured OSS storage.' })
    $document.paths['/v1/sys/register'].post.responses['200'].schema.properties.result = @{
        description = 'Current registration logic returns null on success.'; 'x-nullable' = $true
    }

    # Sort JSON object keys so repeated generation does not introduce noisy diffs.
    function Sort-JsonObject($value) {
        if ($value -is [System.Collections.IDictionary]) {
            $sorted = [ordered]@{}
            foreach ($key in ($value.Keys | Sort-Object -CaseSensitive)) { $sorted[$key] = Sort-JsonObject $value[$key] }
            return $sorted
        }
        if ($value -is [array]) { return ,@($value | ForEach-Object { Sort-JsonObject $_ }) }
        return $value
    }

    New-Item -ItemType Directory -Path (Split-Path $outputFile) -Force | Out-Null
    $json = Sort-JsonObject $document | ConvertTo-Json -Depth 100
    # Preserve the existing formatting when only the PowerShell version differs.
    $existingJson = if (Test-Path -LiteralPath $outputFile) {
        Sort-JsonObject (Read-JsonMap $outputFile) | ConvertTo-Json -Depth 100
    }
    if ($json -cne $existingJson) {
        [IO.File]::WriteAllText($outputFile, $json + [Environment]::NewLine, $utf8NoBom)
    }
    Write-Output "Generated $count operations: $outputFile ($generatorVersion)"
} finally {
    # Only remove the unique temporary directory created by this invocation.
    $resolvedTemp = [IO.Path]::GetFullPath($tempRoot)
    $tempParent = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    if (-not $resolvedTemp.StartsWith($tempParent, [StringComparison]::OrdinalIgnoreCase) -or (Split-Path $resolvedTemp -Leaf) -notlike 'go-zero-swagger-*') {
        throw "Unsafe temporary directory: $resolvedTemp"
    }
    Remove-Item -LiteralPath $resolvedTemp -Recurse -Force
}
