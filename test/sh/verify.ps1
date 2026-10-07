#requires -Version 5.1
[CmdletBinding()]
param([switch]$IncludeDBRegression)

$ErrorActionPreference = 'Stop'
$verifyRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))

function Invoke-GoCheck([string[]]$GoArgs) {
    Write-Host ('Running: go ' + ($GoArgs -join ' '))
    & go @GoArgs
    if ($LASTEXITCODE -ne 0) { throw ('Go check failed: go ' + ($GoArgs -join ' ')) }
}

Push-Location $verifyRoot
try {
    foreach ($verifyTool in @('git', 'go', 'gofmt')) {
        if (-not (Get-Command $verifyTool -ErrorAction SilentlyContinue)) {
            throw ('Required tool is unavailable: ' + $verifyTool + '. Install/configure it before running verification.')
        }
    }
    # Use Git's ignore rules rather than recursively scanning caches or generated
    # temporary workspaces. Include both tracked files and new source files.
    $verifySources = @(& git ls-files --cached --others --exclude-standard -- '*.go')
    if ($LASTEXITCODE -ne 0) { throw 'Unable to list Go source files.' }
    $verifySources = @($verifySources | Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } | Sort-Object -Unique)
    $unformatted = @()
    for ($verifyIndex = 0; $verifyIndex -lt $verifySources.Count; $verifyIndex += 100) {
        $verifyLast = [Math]::Min($verifyIndex + 99, $verifySources.Count - 1)
        $verifyBatch = $verifySources[$verifyIndex..$verifyLast]
        $verifyOutput = @(& gofmt -l @verifyBatch)
        if ($LASTEXITCODE -ne 0) { throw 'gofmt could not parse the source files.' }
        $unformatted += $verifyOutput
    }
    if ($unformatted.Count -gt 0) {
        $unformatted | ForEach-Object { Write-Host ('Formatting required: ' + $_) }
        throw ('gofmt check failed for ' + $unformatted.Count + ' files. Format the listed source files and retry.')
    }
    Write-Host ('gofmt check passed (' + $verifySources.Count + ' source files).')
    Invoke-GoCheck @('vet', './...')
    Invoke-GoCheck @('test', './...', '-count=1')
    Invoke-GoCheck @('build', './...')
    if ($IncludeDBRegression) {
        Write-Host 'Running isolated development-MySQL full SQL import regression.'
        & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot 'db-import-regression.ps1')
        if ($LASTEXITCODE -ne 0) { throw 'Database import regression failed.' }
        Write-Host 'Running isolated development-MySQL migration regression.'
        & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot 'db-regression.ps1')
        if ($LASTEXITCODE -ne 0) { throw 'Database regression failed.' }
    }
    Write-Host 'All requested verification checks passed.'
} finally {
    Pop-Location
}
