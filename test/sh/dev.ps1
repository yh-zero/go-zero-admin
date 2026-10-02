#requires -Version 5.1
[CmdletBinding()]
param(
    [ValidateSet('Start', 'Stop', 'Restart', 'Status')][string]$Action = 'Start',
    [switch]$Migrate,
    [switch]$SkipDependencies
)

$ErrorActionPreference = 'Stop'
$devRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
$devRuntime = Join-Path $devRoot 'bin/dev/managed'
$devManifest = Join-Path $devRuntime 'services.json'
$devLockPath = Join-Path $devRuntime 'operation.lock'
$devUtf8 = New-Object Text.UTF8Encoding($false)
$script:devDidStart = $false

function Save-DevState($State) {
    [IO.File]::WriteAllText($devManifest, ($State | ConvertTo-Json -Depth 6), $devUtf8)
}

function Read-DevState {
    if (Test-Path -LiteralPath $devManifest) { return ([IO.File]::ReadAllText($devManifest) | ConvertFrom-Json) }
    return $null
}

function Test-DevPort([int]$Port) {
    $client = New-Object Net.Sockets.TcpClient
    try {
        $connect = $client.BeginConnect('127.0.0.1', $Port, $null, $null)
        if (-not $connect.AsyncWaitHandle.WaitOne(300)) { return $false }
        $client.EndConnect($connect)
        return $true
    } catch { return $false }
    finally { $client.Close() }
}

function Stop-DevServices {
    $state = Read-DevState
    if (-not $state) { Write-Host 'No managed backend processes are recorded.'; return }
    foreach ($entry in @($state.Services | Sort-Object Name)) {
        $running = Get-CimInstance Win32_Process -Filter ('ProcessId=' + [int]$entry.PID)
        if (-not $running) { continue }
        $expectedPath = [IO.Path]::GetFullPath([string]$entry.Binary)
        if (-not $expectedPath.StartsWith($devRuntime + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
            throw 'Recorded binary is outside this project runtime directory; refusing to stop it.'
        }
        if (-not [string]::Equals([string]$running.ExecutablePath, $expectedPath, [StringComparison]::OrdinalIgnoreCase)) {
            throw ('PID ' + $entry.PID + ' no longer belongs to the recorded backend binary; refusing to stop it.')
        }
        Stop-Process -Id ([int]$entry.PID) -Force
        Write-Host ('Stopped ' + $entry.Name + ' PID ' + $entry.PID)
    }
    Remove-Item -LiteralPath $devManifest -Force
}

function Wait-DevService($Process, [int]$Port, [string]$ErrorLog) {
    $deadline = [DateTime]::UtcNow.AddSeconds(30)
    while ([DateTime]::UtcNow -lt $deadline) {
        $Process.Refresh()
        if ($Process.HasExited) {
            if (Test-Path -LiteralPath $ErrorLog) { Get-Content -LiteralPath $ErrorLog -Tail 15 | Out-Host }
            throw ('Backend process exited before listening on ' + $Port + '; inspect its stdout/stderr logs.')
        }
        if (Test-DevPort $Port) { return }
        Start-Sleep -Milliseconds 250
    }
    throw ('Backend did not become ready on port ' + $Port + '; inspect its logs.')
}

function Start-DevServices {
    foreach ($tool in @('go', 'docker')) {
        if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) { throw ('Required tool is missing: ' + $tool) }
    }
    foreach ($port in @(6001, 7001)) {
        if (Test-DevPort $port) { throw ('Port ' + $port + ' is already occupied. Use Status and stop that exact project process before starting.') }
    }
    if (-not $SkipDependencies) {
        $ready = $true
        foreach ($name in @('mysql', 'redis', 'etcd')) {
            $container = (& docker compose -f (Join-Path $devRoot 'docker-compose.yml') ps -q $name | Out-String).Trim()
            if ($LASTEXITCODE -ne 0) { throw 'Unable to inspect Docker dependencies.' }
            if (-not $container) { $ready = $false; continue }
            $health = (& docker inspect --format '{{.State.Health.Status}}' $container | Out-String).Trim()
            if ($LASTEXITCODE -ne 0 -or $health -ne 'healthy') { $ready = $false }
        }
        if (-not $ready) {
            & docker compose -f (Join-Path $devRoot 'docker-compose.yml') up -d --wait --wait-timeout 60 mysql redis etcd
            if ($LASTEXITCODE -ne 0) { throw 'Docker dependencies failed to start; existing volumes were retained.' }
        }
    }
    $dbAction = if ($Migrate) { 'Migrate' } else { 'Status' }
    $migrationOutput = @(& powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot 'db.ps1') -Action $dbAction 2>&1)
    $migrationOutput | Out-Host
    if ($LASTEXITCODE -ne 0) { throw 'Database migration/status check failed.' }
    if (-not $Migrate -and @($migrationOutput | Where-Object { [string]$_ -match '^PENDING ' }).Count -gt 0) {
        throw 'Database migrations are pending. Run this script with -Migrate to back up and apply them before startup.'
    }
    New-Item -ItemType Directory -Path $devRuntime -Force | Out-Null
    $stamp = (Get-Date -Format 'yyyyMMdd-HHmmss') + '-' + [guid]::NewGuid().ToString('N').Substring(0, 8)
    $runRoot = Join-Path $devRuntime $stamp
    New-Item -ItemType Directory -Path $runRoot -Force | Out-Null
    $state = [ordered]@{ StartedAt = (Get-Date).ToString('o'); Services = @() }
    foreach ($service in @(
        @{ Name = 'rpc'; Port = 6001; Package = './application/applet/rpc'; Config = 'application/applet/rpc/etc/applet.yaml' },
        @{ Name = 'api'; Port = 7001; Package = './application/applet/api'; Config = 'application/applet/api/etc/applet-api.yaml' }
    )) {
        $binary = Join-Path $runRoot ('applet-' + $service.Name + '.exe')
        Write-Host ('Building ' + $service.Name + '...')
        & go build -o $binary $service.Package
        if ($LASTEXITCODE -ne 0) { throw ('Build failed: ' + $service.Name) }
        $stdout = Join-Path $runRoot ($service.Name + '.out.log')
        $stderr = Join-Path $runRoot ($service.Name + '.err.log')
        $config = Join-Path $devRoot $service.Config
        $process = Start-Process -FilePath $binary -ArgumentList @('-f', ('"' + $config + '"')) -WorkingDirectory $devRoot -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr -PassThru
        $script:devDidStart = $true
        $state.Services += [ordered]@{ Name = $service.Name; Port = $service.Port; PID = $process.Id; Binary = $binary; Config = $config; Stdout = $stdout; Stderr = $stderr }
        Save-DevState $state
        Wait-DevService $process $service.Port $stderr
        Write-Host ('Started ' + $service.Name + ': 127.0.0.1:' + $service.Port + ', PID ' + $process.Id)
    }
    $captcha = Invoke-RestMethod -Uri 'http://127.0.0.1:7001/v1/sys/randomImage' -TimeoutSec 5
    if ($captcha.code -ne 200 -or -not $captcha.result.captchaId) { throw 'API CAPTCHA readiness check failed.' }
    Write-Host ('Backend ready. PID/log manifest: ' + $devManifest)
}

Push-Location $devRoot
$devOperationLock = $null
try {
    New-Item -ItemType Directory -Path $devRuntime -Force | Out-Null
    try {
        $devOperationLock = [IO.File]::Open($devLockPath, [IO.FileMode]::OpenOrCreate, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None)
    } catch [IO.IOException] {
        throw 'Another backend operation is in progress. Retry after it finishes.'
    }
    switch ($Action) {
        'Status' {
            $state = Read-DevState
            if ($state) { $state.Services | Format-Table Name, Port, PID, Stdout -AutoSize | Out-Host }
            else { Write-Host 'No managed backend processes are recorded.' }
            foreach ($port in @(6001, 7001)) { Write-Host ('Port ' + $port + ': ' + $(if (Test-DevPort $port) { 'listening' } else { 'closed' })) }
        }
        'Stop' { Stop-DevServices }
        'Restart' { Stop-DevServices; Start-DevServices }
        'Start' { Start-DevServices }
    }
} catch {
    if ($script:devDidStart) { Stop-DevServices }
    throw
} finally {
    if ($devOperationLock) { $devOperationLock.Dispose() }
    Pop-Location
}
