#requires -Version 5.1
$ErrorActionPreference = 'Stop'
$backend = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$fixtureRoot = Join-Path $env:TEMP ('gozero-verify-test-' + [guid]::NewGuid().ToString('N'))
$fixture = Join-Path $fixtureRoot 'project'
$bin = Join-Path $fixtureRoot 'bin'
$originalPath = $env:PATH
try {
    foreach ($relative in @('application', 'pkg', 'docker/etc', 'data/scripts')) {
        New-Item -ItemType Directory -Path (Join-Path $fixture $relative) -Force | Out-Null
    }
    New-Item -ItemType Directory -Path $bin | Out-Null
    Copy-Item -LiteralPath (Join-Path $backend 'data/scripts/verify.ps1') -Destination (Join-Path $fixture 'data/scripts/verify.ps1')
    [IO.File]::WriteAllText((Join-Path $fixture 'application/probe.go'), "package probe`n")
    [IO.File]::WriteAllText((Join-Path $fixture 'go.mod'), "module fixture`ngo 1.27.0`n")
    [IO.File]::WriteAllText((Join-Path $fixture 'go.sum'), '')
    [IO.File]::WriteAllText((Join-Path $bin 'gofmt.cmd'), "@echo off`r`nexit /b 2`r`n", [Text.Encoding]::ASCII)
    [IO.File]::WriteAllText((Join-Path $bin 'go.cmd'), "@echo off`r`nif `"%1`"==`"list`" echo fixture/probe`r`nexit /b 0`r`n", [Text.Encoding]::ASCII)
    [IO.File]::WriteAllText((Join-Path $bin 'goctl.cmd'), "@echo off`r`nexit /b 0`r`n", [Text.Encoding]::ASCII)
    $env:PATH = $bin + ';' + $originalPath
    $ErrorActionPreference = 'Continue'
    try {
        $output = @(& powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $fixture 'data/scripts/verify.ps1') 2>&1)
        $verifyExit = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = 'Stop'
    }
    if ($verifyExit -eq 0) { throw 'gofmt failure was incorrectly reported as success.' }
    if (($output -join "`n") -notmatch 'gofmt.*2') { throw 'The script failed outside the expected gofmt exit-code assertion.' }
    Write-Host 'verify script regression passed.'
} finally {
    $env:PATH = $originalPath
    $resolvedRoot = [IO.Path]::GetFullPath($fixtureRoot)
    $tempPrefix = [IO.Path]::GetFullPath($env:TEMP).TrimEnd('\') + '\gozero-verify-test-'
    if (-not $resolvedRoot.StartsWith($tempPrefix, [StringComparison]::OrdinalIgnoreCase)) { throw 'Refusing cleanup outside the test temporary directory.' }
    Remove-Item -LiteralPath $resolvedRoot -Recurse -Force -ErrorAction SilentlyContinue
}
# 子进程的预期失败不能成为本回归脚本的最终退出码；异常不会执行到这里。
exit 0
