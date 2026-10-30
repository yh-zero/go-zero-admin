#requires -Version 5.1
# 用原生进程读取完整 UTF-8 输出，避免 PowerShell 管道丢失末尾换行或改写中文。
function Invoke-VerifyTool {
    param([string]$FileName, [string[]]$ToolArguments, [string]$WorkingDirectory)
    $start = New-Object System.Diagnostics.ProcessStartInfo
    $start.FileName = $FileName
    $start.WorkingDirectory = $WorkingDirectory
    $start.UseShellExecute = $false
    $start.CreateNoWindow = $true
    $start.RedirectStandardOutput = $true
    $start.RedirectStandardError = $true
    $start.StandardOutputEncoding = New-Object System.Text.UTF8Encoding($false)
    $start.StandardErrorEncoding = New-Object System.Text.UTF8Encoding($false)
    # 调用仅传入固定 Go 参数和以文件名结尾的绝对路径，不经过 shell。
    $quoted = foreach ($argument in $ToolArguments) {
        if ($argument.Contains('"') -or $argument.EndsWith('\')) { throw '自检工具参数包含不支持的引号或尾部反斜杠。' }
        '"' + $argument + '"'
    }
    $start.Arguments = $quoted -join ' '
    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $start
    try {
        if (-not $process.Start()) { throw ('无法启动自检工具：' + $FileName) }
        $output = $process.StandardOutput.ReadToEndAsync()
        $errorOutput = $process.StandardError.ReadToEndAsync()
        $process.WaitForExit()
        return [pscustomobject]@{
            ExitCode = $process.ExitCode
            Output = $output.GetAwaiter().GetResult()
            ErrorOutput = $errorOutput.GetAwaiter().GetResult()
        }
    } finally { $process.Dispose() }
}

function Get-VerifyDiagnostic($Result) {
    $lines = @(($Result.ErrorOutput + "`n" + $Result.Output).Trim() -split '\r?\n')
    $text = ($lines | Select-Object -First 16) -join "`n"
    if ($lines.Count -gt 16) { $text += "`n（后续诊断已省略）" }
    return $text
}

function Get-GoFormatDifferences {
    param([string[]]$Paths)
    $utf8 = New-Object System.Text.UTF8Encoding($false, $true)
    $files = foreach ($path in $Paths) {
        $entry = Get-Item -LiteralPath $path
        if ($entry.PSIsContainer) { Get-ChildItem -LiteralPath $entry.FullName -Filter '*.go' -File -Recurse }
        elseif ($entry.Extension -eq '.go') { $entry }
    }
    foreach ($file in ($files | Sort-Object FullName -Unique)) {
        $formatted = Invoke-VerifyTool 'gofmt' @($file.FullName) (Get-Location).Path
        if ($formatted.ExitCode -ne 0) { throw ('gofmt 检查失败：' + $file.FullName + "`n" + (Get-VerifyDiagnostic $formatted)) }
        $original = $utf8.GetString([IO.File]::ReadAllBytes($file.FullName))
        if ($original.Replace("`r`n","`n") -cne $formatted.Output.Replace("`r`n","`n")) { $file.FullName }
    }
}

function Assert-GoModuleTidy {
    param([string]$ProjectRoot)
    $moduleRoot = [IO.Path]::GetFullPath($ProjectRoot)
    $help = Invoke-VerifyTool 'go' @('help','build') $moduleRoot
    if ($help.ExitCode -ne 0 -or $help.Output -notmatch '(?m)^\s+-modfile\s') {
        throw ('本地 Go 未证明支持 -modfile，保留 tidy 检查失败，不改写原锁文件。' + "`n" + (Get-VerifyDiagnostic $help))
    }
    $temporaryName = 'gozero-verify-mod-' + [guid]::NewGuid().ToString('N')
    $temporaryRoot = Join-Path ([IO.Path]::GetTempPath()) $temporaryName
    $utf8 = New-Object System.Text.UTF8Encoding($false, $true)
    New-Item -ItemType Directory -Path $temporaryRoot | Out-Null
    try {
        # 只归一 CRLF；保留内容、顺序和其他空白，让真正的 tidy 差异照常失败。
        foreach ($extension in @('mod','sum')) {
            $originalPath = Join-Path $moduleRoot ('go.' + $extension)
            if (Test-Path -LiteralPath $originalPath -PathType Leaf) {
                $content = $utf8.GetString([IO.File]::ReadAllBytes($originalPath)).Replace("`r`n","`n")
                [IO.File]::WriteAllText((Join-Path $temporaryRoot ('verify.' + $extension)), $content, $utf8)
            }
        }
        # 模块根目录保持原项目目录；-modfile 的配套 .sum 也只使用临时副本。
        $result = Invoke-VerifyTool 'go' @('mod','tidy','-diff','-modfile',(Join-Path $temporaryRoot 'verify.mod')) $moduleRoot
        if ($result.ExitCode -ne 0) {
            throw ('go.mod/go.sum 不整洁或 tidy 执行失败：见 AGENTS.md 第 2.5 条。' + "`n" + (Get-VerifyDiagnostic $result))
        }
    } finally {
        $resolved = (Resolve-Path -LiteralPath $temporaryRoot).Path
        $expected = [IO.Path]::GetFullPath((Join-Path ([IO.Path]::GetTempPath()) $temporaryName))
        if ($resolved -cne $expected -or $temporaryName -notmatch '^gozero-verify-mod-[0-9a-f]{32}$') { throw '拒绝清理意外的模块自检临时目录。' }
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
}
