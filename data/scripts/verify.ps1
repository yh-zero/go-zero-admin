#requires -Version 5.1
[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$verifyRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))

function Invoke-GoCheck([string[]]$GoArgs) {
    Write-Host ('Running: go ' + ($GoArgs -join ' '))
    & go @GoArgs
    if ($LASTEXITCODE -ne 0) { throw ('Go check failed: go ' + ($GoArgs -join ' ')) }
}

function Get-UnformattedGoSources([string[]]$Sources) {
    # Git may check out Go files with CRLF on Windows. Ignore only that newline
    # difference while retaining gofmt's checks for actual source formatting.
    $formatPrefix = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd([char]92, [char]47) + [IO.Path]::DirectorySeparatorChar + 'gozero-gofmt-'
    $formatRoot = $formatPrefix + [guid]::NewGuid().ToString('N')
    $formatUtf8 = New-Object Text.UTF8Encoding($false, $true)
    $formatCopies = New-Object 'Collections.Generic.List[string]'
    $originalByCopy = @{}
    try {
        [IO.Directory]::CreateDirectory($formatRoot) | Out-Null
        foreach ($source in $Sources) {
            $copy = Join-Path $formatRoot ('source-' + $formatCopies.Count + '.go')
            $text = $formatUtf8.GetString([IO.File]::ReadAllBytes((Join-Path $verifyRoot $source)))
            [IO.File]::WriteAllBytes($copy, $formatUtf8.GetBytes($text.Replace("`r`n", "`n")))
            $formatCopies.Add($copy)
            $originalByCopy[$copy] = $source
        }
        for ($index = 0; $index -lt $formatCopies.Count; $index += 100) {
            $last = [Math]::Min($index + 99, $formatCopies.Count - 1)
            $batch = @($formatCopies[$index..$last])
            $output = @(& gofmt -l @batch)
            if ($LASTEXITCODE -ne 0) { throw 'gofmt could not parse the source files.' }
            foreach ($copy in $output) {
                if (-not $originalByCopy.ContainsKey($copy)) { throw 'gofmt returned an unknown source path.' }
                $originalByCopy[$copy]
            }
        }
    } finally {
        if (-not $formatRoot.StartsWith($formatPrefix, [StringComparison]::OrdinalIgnoreCase)) {
            throw 'Refusing to remove a format fixture outside its temporary directory.'
        }
        if (Test-Path -LiteralPath $formatRoot) {
            $resolvedRoot = [IO.Path]::GetFullPath((Resolve-Path -LiteralPath $formatRoot).ProviderPath)
            if (-not [string]::Equals($resolvedRoot, $formatRoot, [StringComparison]::OrdinalIgnoreCase)) {
                throw 'Refusing to remove a format fixture whose resolved path has changed.'
            }
            Remove-Item -LiteralPath $resolvedRoot -Recurse -Force
        }
    }
}

Push-Location $verifyRoot
try {
    foreach ($verifyTool in @('git', 'go', 'gofmt')) {
        if (-not (Get-Command $verifyTool -ErrorAction SilentlyContinue)) {
            throw ('Required tool is unavailable: ' + $verifyTool + '. Install/configure it before running verification.')
        }
    }
    & (Join-Path $PSScriptRoot 'build-fresh-sql.ps1') -Check
    if ($LASTEXITCODE -ne 0) { throw 'data/db/gozero-admin.sql is stale. Run data/scripts/build-fresh-sql.ps1 and commit the result.' }
    # Use Git's ignore rules rather than recursively scanning caches or generated
    # temporary workspaces. Include both tracked files and new source files.
    $verifySources = @(& git ls-files --cached --others --exclude-standard -- '*.go')
    if ($LASTEXITCODE -ne 0) { throw 'Unable to list Go source files.' }
    $verifySources = @($verifySources | Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } | Sort-Object -Unique)
    $unformatted = @(Get-UnformattedGoSources -Sources $verifySources)
    if ($unformatted.Count -gt 0) {
        $unformatted | ForEach-Object { Write-Host ('Formatting required: ' + $_) }
        throw ('gofmt check failed for ' + $unformatted.Count + ' files. Format the listed source files and retry.')
    }
    Write-Host ('gofmt check passed (' + $verifySources.Count + ' source files).')
    Invoke-GoCheck @('vet', './...')
    Invoke-GoCheck @('test', './...', '-count=1')
    Invoke-GoCheck @('build', './...')
    Write-Host 'All requested verification checks passed.'
} finally {
    Pop-Location
}
