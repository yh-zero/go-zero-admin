#requires -Version 5.1
# Definitions only: dot-sourcing this file does not read or change environment variables.

function Restore-DevEnvironment {
    [CmdletBinding()]
    param([AllowNull()][System.Collections.IDictionary]$Snapshot)

    if ($null -eq $Snapshot) { return }
    foreach ($name in $Snapshot.Keys) {
        [Environment]::SetEnvironmentVariable([string]$name, $Snapshot[$name], [EnvironmentVariableTarget]::Process)
    }
}

function Get-DevModelEnvironment {
    $values = @{}
    foreach ($name in @('API_KEY_DEEPSEEK', 'API_KEY_QWEN')) {
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
        foreach ($name in @('API_KEY_DEEPSEEK', 'API_KEY_QWEN')) {
            [Environment]::SetEnvironmentVariable($name, $Values[$name], [EnvironmentVariableTarget]::Process)
        }
        & $Action
    } finally { Restore-DevEnvironment -Snapshot $snapshot }
}

function Invoke-DevWithoutModelKeys {
    [CmdletBinding()]
    param([Parameter(Mandatory = $true)][scriptblock]$Action)

    Invoke-DevWithModelKeys -Values @{ API_KEY_DEEPSEEK = $null; API_KEY_QWEN = $null } -Action $Action
}
