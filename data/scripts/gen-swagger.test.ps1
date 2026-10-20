#requires -Version 5.1
$ErrorActionPreference = 'Stop'
$source = Join-Path $PSScriptRoot 'gen.ps1'
$parseTokens = $null
$parseErrors = $null
$ast = [Management.Automation.Language.Parser]::ParseFile($source, [ref]$parseTokens, [ref]$parseErrors)
foreach ($name in @('ConvertTo-JsonMap', 'Sort-JsonObject', 'Copy-JsonMap')) {
    $functionAst = $ast.Find({ param($node) $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq $name }, $true)
    if ($null -eq $functionAst) { throw "Missing JSON helper: $name" }
    . ([scriptblock]::Create($functionAst.Extent.Text))
}
$inputJson = '{"consumes":["application/json"],"security":[{"BearerAuth":[]}],"empty":[],"nested":{"values":[{"name":"one"}]}}'
$converted = ConvertTo-JsonMap (ConvertFrom-Json -InputObject $inputJson)
# Exercise the same audience-clone step as the generator, including WinPS wrappers.
$copy = Copy-JsonMap $converted
$outputJson = Sort-JsonObject $copy | ConvertTo-Json -Depth 20
$parsed = ConvertFrom-Json -InputObject $outputJson
if ($parsed.security -isnot [array] -or $parsed.security.Count -ne 1) { throw 'Security must remain a single-item array' }
if ($parsed.security[0].BearerAuth -isnot [array] -or $parsed.security[0].BearerAuth.Count -ne 0) { throw 'BearerAuth must remain an empty array' }
if ($parsed.consumes -isnot [array] -or $parsed.consumes[0] -ne 'application/json') { throw 'Consumes array changed' }
if ($parsed.empty -isnot [array] -or $parsed.empty.Count -ne 0) { throw 'Empty array changed' }
if ($parsed.nested.values -isnot [array] -or $parsed.nested.values[0].name -ne 'one') { throw 'Nested array changed' }
Write-Output 'Swagger JSON empty, singleton and nested arrays preserved.'

# Check the real multipart override: the generator must retain private uploads.
$upload = @{}
$uploadParameters = $ast.Find({ param($node)
    $node -is [Management.Automation.Language.AssignmentStatementAst] -and $node.Left.Extent.Text -eq '$upload.parameters'
}, $true)
if ($null -eq $uploadParameters) { throw 'Missing upload parameters override' }
. ([scriptblock]::Create($uploadParameters.Extent.Text))
$files = @($upload.parameters | Where-Object { $_.name -eq 'file_img' })
$visibility = @($upload.parameters | Where-Object { $_.name -eq 'visibility' })
if ($files.Count -ne 1 -or $files[0].in -ne 'formData' -or $files[0].type -ne 'file' -or -not $files[0].required) {
    throw 'Upload must accept one required file_img part'
}
if ($visibility.Count -ne 1 -or $visibility[0].in -ne 'formData' -or $visibility[0].type -ne 'string' -or $visibility[0].required -or $visibility[0].default -ne 'public' -or ($visibility[0].enum -join ',') -ne 'public,private') {
    throw 'Upload must document optional public/private visibility with the public default'
}
Write-Output 'Swagger multipart upload includes file_img and public/private visibility.'
