#requires -Version 5.1
# Definitions only: dot-sourcing this file does not read or change environment variables.

function ConvertFrom-DevEnvText {
    [CmdletBinding()]
    param(
        [AllowEmptyString()][string]$Text,
        [Parameter(Mandatory = $true)][string]$Path
    )

    $values = @{}
    $lines = [regex]::Split($Text, "\r\n|\n|\r")
    for ($index = 0; $index -lt $lines.Length; $index++) {
        $line = [string]$lines[$index]
        if ($index -eq 0) { $line = $line.TrimStart([char]0xFEFF) }
        $lineNumber = $index + 1
        $failure = "Invalid environment file '$Path' at line ${lineNumber}: "
        if ($line.IndexOf([char]0) -ge 0) { throw ($failure + 'NUL characters are not allowed.') }
        $trimmed = $line.Trim()
        if (-not $trimmed -or $trimmed.StartsWith('#')) { continue }
        if ($line -notmatch '^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=(.*)$') {
            throw ($failure + 'expected KEY=value.')
        }
        $name = $Matches[1]
        $value = $Matches[2].Trim()
        if ($values.ContainsKey($name)) { throw ($failure + 'duplicate variable name.') }
        if ($value.Length -gt 0) {
            $first = $value[0]
            $last = $value[$value.Length - 1]
            if ($first -eq [char]39 -or $first -eq [char]34) {
                if ($value.Length -lt 2 -or $last -ne $first) {
                    throw ($failure + 'quoted values must use matching enclosing quotes.')
                }
                $value = $value.Substring(1, $value.Length - 2)
            } elseif ($last -eq [char]39 -or $last -eq [char]34) {
                throw ($failure + 'quoted values must use matching enclosing quotes.')
            }
        }
        # Values are literal, including $, backticks, = and #. Never evaluate file content.
        $values.Add($name, $value)
    }
    return $values
}

function Read-DevEnvFile {
    [CmdletBinding()]
    param([Parameter(Mandatory = $true)][string]$Path)

    if (-not (Test-Path -LiteralPath $Path)) { return @{} }
    try { $bytes = [IO.File]::ReadAllBytes($Path) }
    catch { throw ("Cannot read environment file '$Path'.") }
    $encoding = New-Object Text.UTF8Encoding($false, $true)
    try { $text = $encoding.GetString($bytes) }
    catch {
        $lineNumber = 1
        $cause = $_.Exception
        while ($cause.InnerException) { $cause = $cause.InnerException }
        if ($cause -is [Text.DecoderFallbackException]) {
            for ($index = 0; $index -lt [Math]::Min($cause.Index, $bytes.Length); $index++) {
                if ($bytes[$index] -eq 10 -or ($bytes[$index] -eq 13 -and ($index + 1 -ge $bytes.Length -or $bytes[$index + 1] -ne 10))) {
                    $lineNumber++
                }
            }
        }
        throw ("Invalid environment file '$Path' at line ${lineNumber}: expected UTF-8 text.")
    }
    return ConvertFrom-DevEnvText -Text $text -Path $Path
}

function Restore-DevEnvironment {
    [CmdletBinding()]
    param([AllowNull()][System.Collections.IDictionary]$Snapshot)

    if ($null -eq $Snapshot) { return }
    foreach ($name in $Snapshot.Keys) {
        [Environment]::SetEnvironmentVariable([string]$name, $Snapshot[$name], [EnvironmentVariableTarget]::Process)
    }
}

function Set-DevEnvironmentDefaults {
    [CmdletBinding()]
    param([Parameter(Mandatory = $true)][System.Collections.IDictionary]$Values)

    # Validate the whole input before changing Process scope, including direct function callers.
    foreach ($name in $Values.Keys) {
        if ([string]$name -notmatch '^[A-Za-z_][A-Za-z0-9_]*$' -or ([string]$Values[$name]).IndexOf([char]0) -ge 0) {
            throw 'Invalid environment defaults; no variables were changed.'
        }
    }
    $snapshot = @{}
    try {
        foreach ($name in $Values.Keys) {
            $previous = [Environment]::GetEnvironmentVariable([string]$name, [EnvironmentVariableTarget]::Process)
            if (-not [string]::IsNullOrEmpty($previous)) { continue }
            $snapshot.Add($name, $previous)
            [Environment]::SetEnvironmentVariable([string]$name, [string]$Values[$name], [EnvironmentVariableTarget]::Process)
        }
    } catch {
        Restore-DevEnvironment -Snapshot $snapshot
        throw 'Unable to apply environment defaults; original Process variables were restored.'
    }
    return $snapshot
}

function Get-DevModelEnvironment {
    $values = @{}
    foreach ($name in @('DEEPSEEK_API_KEY', 'QWEN_API_KEY')) {
        $values[$name] = [Environment]::GetEnvironmentVariable($name, [EnvironmentVariableTarget]::Process)
    }
    return $values
}

function Invoke-DevWithModelKeys {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)][System.Collections.IDictionary]$Values,
        [Parameter(Mandatory = $true)][scriptblock]$Action
    )

    $snapshot = Get-DevModelEnvironment
    try {
        foreach ($name in @('DEEPSEEK_API_KEY', 'QWEN_API_KEY')) {
            [Environment]::SetEnvironmentVariable($name, $Values[$name], [EnvironmentVariableTarget]::Process)
        }
        & $Action
    } finally { Restore-DevEnvironment -Snapshot $snapshot }
}

function Invoke-DevWithoutModelKeys {
    [CmdletBinding()]
    param([Parameter(Mandatory = $true)][scriptblock]$Action)

    Invoke-DevWithModelKeys -Values @{ DEEPSEEK_API_KEY = $null; QWEN_API_KEY = $null } -Action $Action
}
