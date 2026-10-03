#requires -Version 5.1
[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$fixtureRoot = Join-Path ([IO.Path]::GetTempPath()) ('go-zero-env-' + [guid]::NewGuid().ToString('N'))
$fixtureRoot = [IO.Path]::GetFullPath($fixtureRoot)
$utf8 = New-Object Text.UTF8Encoding($false)
$originalProcess = @{}
$fixtureNames = @('DEEPSEEK_API_KEY', 'QWEN_API_KEY', 'GOZERO_ENV_FIXTURE_DEFAULT', 'GOZERO_ENV_FIXTURE_PRIORITY', 'GOZERO_ENV_FIXTURE_FAILURE')
foreach ($name in $fixtureNames) {
    $originalProcess[$name] = [Environment]::GetEnvironmentVariable($name, [EnvironmentVariableTarget]::Process)
}
. (Join-Path $PSScriptRoot 'env.ps1')
$script:envChecks = 0

function Assert-Env([bool]$Condition, [string]$Description) {
    if (-not $Condition) { throw ('Environment regression failed: ' + $Description) }
    $script:envChecks++
}

function Assert-InvalidEnv([scriptblock]$Read, [string]$ExpectedPath, [int]$ExpectedLine) {
    $failure = $null
    try { & $Read | Out-Null } catch { $failure = $_.Exception.Message }
    Assert-Env ($null -ne $failure) 'malformed file must fail'
    Assert-Env ($failure.Contains($ExpectedPath) -and $failure.Contains('line ' + $ExpectedLine + ':')) 'failure must identify the file and line'
    Assert-Env (-not $failure.Contains('fixture-secret')) 'failure must omit values'
}

function Invoke-FixtureChild([string]$Name) {
    $output = Join-Path $fixtureRoot ($Name + '.json')
    $stdout = Join-Path $fixtureRoot ($Name + '.out.log')
    $stderr = Join-Path $fixtureRoot ($Name + '.err.log')
    $process = Start-Process -FilePath powershell.exe -ArgumentList @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', ('"' + $childScript + '"'), ('"' + $output + '"')) -WindowStyle Hidden -Wait -PassThru -RedirectStandardOutput $stdout -RedirectStandardError $stderr
    if ($process.ExitCode -ne 0) { throw 'Fixture child failed; inspect fixture logs.' }
    return ([IO.File]::ReadAllText($output) | ConvertFrom-Json)
}

try {
    New-Item -ItemType Directory -Path $fixtureRoot -Force | Out-Null
    # Never load the real .env.local. All contents below are disposable literal fixtures.
    $validPath = Join-Path $fixtureRoot 'valid.env'
    $literal = @'
  # full-line comment

PLAIN=left=right#literal
SINGLE='$(throw "fixture-secret") ${HOME} `n #='
DOUBLE="$(throw 'fixture-secret') ${HOME} `n #="
EMPTY=""
SPACES = '  surrounding spaces  '
'@
    # Add the trimming fixture's trailing space without source-line whitespace.
    $literal = $literal.Replace("#='", "#=' ")
    [IO.File]::WriteAllText($validPath, ([char]0xFEFF + $literal + "`nUTF8=" + [char]0x4E2D + [char]0x6587), $utf8)
    $parsed = Read-DevEnvFile -Path $validPath
    Assert-Env ($parsed.Count -eq 6) 'UTF-8 BOM, comments and blank lines must parse'
    Assert-Env ($parsed['PLAIN'] -ceq 'left=right#literal') 'equals and hash must remain literal'
    Assert-Env ($parsed['SINGLE'] -ceq '$(throw "fixture-secret") ${HOME} `n #=') 'single quotes must remain literal without interpolation'
    Assert-Env ($parsed['DOUBLE'] -ceq '$(throw ''fixture-secret'') ${HOME} `n #=') 'double quotes must remain literal without interpolation'
    Assert-Env ($parsed['EMPTY'] -ceq '' -and $parsed['SPACES'] -ceq '  surrounding spaces  ') 'quoted empty values and spaces must survive'
    Assert-Env ($parsed['UTF8'] -ceq ([string][char]0x4E2D + [char]0x6587)) 'UTF-8 values must decode correctly'
    Assert-Env ((Read-DevEnvFile -Path (Join-Path $fixtureRoot 'absent.env')).Count -eq 0) 'missing file must be optional'

    $invalidPath = Join-Path $fixtureRoot 'invalid.env'
    foreach ($badLine in @('fixture-secret', 'BAD="fixture-secret', 'BAD=''fixture-secret"', 'BAD=fixture-secret"', '1BAD=fixture-secret')) {
        [IO.File]::WriteAllText($invalidPath, "GOOD=literal`n" + $badLine, $utf8)
        Assert-InvalidEnv { Read-DevEnvFile -Path $invalidPath } $invalidPath 2
    }
    [IO.File]::WriteAllText($invalidPath, "KEY=literal`nkey=fixture-secret", $utf8)
    Assert-InvalidEnv { Read-DevEnvFile -Path $invalidPath } $invalidPath 2
    [IO.File]::WriteAllText($invalidPath, "KEY=literal`nBAD=fixture-secret" + [char]0, $utf8)
    Assert-InvalidEnv { Read-DevEnvFile -Path $invalidPath } $invalidPath 2
    [IO.File]::WriteAllBytes($invalidPath, [byte[]]@(75, 61, 97, 10, 66, 61, 255))
    Assert-InvalidEnv { Read-DevEnvFile -Path $invalidPath } $invalidPath 2

    foreach ($name in $fixtureNames) { [Environment]::SetEnvironmentVariable($name, $null, [EnvironmentVariableTarget]::Process) }
    [Environment]::SetEnvironmentVariable('GOZERO_ENV_FIXTURE_PRIORITY', 'process-wins', [EnvironmentVariableTarget]::Process)
    $defaults = Set-DevEnvironmentDefaults -Values @{ GOZERO_ENV_FIXTURE_DEFAULT = 'file-default'; GOZERO_ENV_FIXTURE_PRIORITY = 'file-loses' }
    Assert-Env ([Environment]::GetEnvironmentVariable('GOZERO_ENV_FIXTURE_DEFAULT', 'Process') -ceq 'file-default') 'file must fill an absent Process value'
    Assert-Env ([Environment]::GetEnvironmentVariable('GOZERO_ENV_FIXTURE_PRIORITY', 'Process') -ceq 'process-wins') 'nonempty Process value must take priority'
    Restore-DevEnvironment -Snapshot $defaults
    Assert-Env ($null -eq [Environment]::GetEnvironmentVariable('GOZERO_ENV_FIXTURE_DEFAULT', 'Process')) 'restore must remove newly added defaults'
    Assert-Env ([Environment]::GetEnvironmentVariable('GOZERO_ENV_FIXTURE_PRIORITY', 'Process') -ceq 'process-wins') 'restore must preserve original Process values'
    $failure = $null
    try { Set-DevEnvironmentDefaults -Values @{ GOZERO_ENV_FIXTURE_DEFAULT = 'file-default'; 'INVALID-NAME' = 'fixture-secret' } | Out-Null }
    catch { $failure = $_.Exception.Message }
    Assert-Env ($null -ne $failure -and $null -eq [Environment]::GetEnvironmentVariable('GOZERO_ENV_FIXTURE_DEFAULT', 'Process')) 'defaults must validate all entries before applying any'

    [Environment]::SetEnvironmentVariable('DEEPSEEK_API_KEY', 'fixture-deepseek', [EnvironmentVariableTarget]::Process)
    [Environment]::SetEnvironmentVariable('QWEN_API_KEY', 'fixture-qwen', [EnvironmentVariableTarget]::Process)
    $modelValues = Get-DevModelEnvironment
    $childScript = Join-Path $fixtureRoot 'child.ps1'
    [IO.File]::WriteAllText($childScript, @'
param([string]$Output)
$result = @{
    DeepSeekPresent = -not [string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable('DEEPSEEK_API_KEY', 'Process'))
    QwenPresent = -not [string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable('QWEN_API_KEY', 'Process'))
}
[IO.File]::WriteAllText($Output, ($result | ConvertTo-Json))
'@, $utf8)
    Invoke-DevWithoutModelKeys -Action {
        $apiChild = Invoke-FixtureChild 'api'
        Assert-Env (-not $apiChild.DeepSeekPresent -and -not $apiChild.QwenPresent) 'API child must not inherit either model key'
        $rpcChild = Invoke-FixtureChild 'rpc'
        Assert-Env (-not $rpcChild.DeepSeekPresent -and -not $rpcChild.QwenPresent) 'business RPC child must not inherit either model key'
        $aiChild = Invoke-DevWithModelKeys -Values $modelValues -Action { Invoke-FixtureChild 'ai-rpc' }
        Assert-Env ($aiChild.DeepSeekPresent -and $aiChild.QwenPresent) 'AI RPC child must inherit the selected model keys'
        Assert-Env ($null -eq [Environment]::GetEnvironmentVariable('DEEPSEEK_API_KEY', 'Process')) 'nested AI RPC injection must restore the key-free outer environment'
    }
    Assert-Env ([Environment]::GetEnvironmentVariable('DEEPSEEK_API_KEY', 'Process') -ceq 'fixture-deepseek') 'key filter must restore caller after success'
    try { Invoke-DevWithoutModelKeys -Action { throw 'controlled action failure' } } catch { }
    Assert-Env ([Environment]::GetEnvironmentVariable('QWEN_API_KEY', 'Process') -ceq 'fixture-qwen') 'key filter must restore caller after failure'

    # Execute a copy of the real entrypoint with service functions replaced by safe stubs.
    # This verifies ordering/finally without builds, Docker, migrations, network or service restarts.
    $fixtureScripts = Join-Path $fixtureRoot 'test/sh'
    New-Item -ItemType Directory -Path $fixtureScripts -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'env.ps1') -Destination (Join-Path $fixtureScripts 'env.ps1')
    $sourcePath = Join-Path $PSScriptRoot 'dev.ps1'
    $source = [IO.File]::ReadAllText($sourcePath)
    $tokens = $null
    $parseErrors = $null
    $ast = [Management.Automation.Language.Parser]::ParseInput($source, [ref]$tokens, [ref]$parseErrors)
    Assert-Env ($parseErrors.Count -eq 0) 'dev entrypoint must have valid PowerShell syntax'
    # Run the actual stop function with process inspection and termination mocked.
    # The manifest is disposable; this scope cannot inspect or stop real processes.
    $stopFunction = @($ast.FindAll({ param($node) $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq 'Stop-DevServices' }, $true))
    Assert-Env ($stopFunction.Count -eq 1) 'stop fixture must use the real service stop function'
    $stopDefinition = [scriptblock]::Create($stopFunction[0].Extent.Text)
    $stopRuntime = Join-Path $fixtureRoot 'stop-cases'
    New-Item -ItemType Directory -Path $stopRuntime -Force | Out-Null
    & {
        param([scriptblock]$Definition, [string]$Runtime)
        $devRuntime = $Runtime
        $devManifest = Join-Path $Runtime 'services.json'
        $stopFixture = @{}
        function Read-DevState { return $stopFixture.State }
        function Get-CimInstance([string]$ClassName, [string]$Filter) {
            if ($ClassName -ne 'Win32_Process' -or $Filter -notmatch '^ProcessId=([0-9]+)$') {
                throw 'Unexpected process inspection in stop fixture.'
            }
            $processId = [int]$Matches[1]
            $stopFixture.Inspected.Add($processId)
            return $stopFixture.Processes[$processId]
        }
        function Stop-Process([int]$Id, [switch]$Force) {
            if (-not $Force -or $stopFixture.Inspected.Count -ne $stopFixture.ExpectedChecks) {
                throw 'Stop must inspect every recorded process before terminating any.'
            }
            $stopFixture.Stopped.Add($Id)
        }
        function New-StopEntry([string]$Name, [int]$ProcessId) {
            return [pscustomobject]@{ Name = $Name; PID = $ProcessId; Binary = (Join-Path $Runtime ('run/applet-' + $Name + '.exe')) }
        }
        function Reset-StopFixture([object[]]$Entries, [hashtable]$Processes) {
            $stopFixture.State = [pscustomobject]@{ Services = $Entries }
            $stopFixture.Processes = $Processes
            $stopFixture.Stopped = New-Object 'Collections.Generic.List[int]'
            $stopFixture.Inspected = New-Object 'Collections.Generic.List[int]'
            $stopFixture.ExpectedChecks = $Entries.Count
            [IO.File]::WriteAllText($devManifest, 'disposable fixture manifest')
        }
        . $Definition
        $business = New-StopEntry 'rpc' 101
        $ai = New-StopEntry 'ai-rpc' 202
        $api = New-StopEntry 'api' 303
        $processes = @{ 101 = [pscustomobject]@{ ExecutablePath = $business.Binary }; 202 = [pscustomobject]@{ ExecutablePath = $ai.Binary }; 303 = [pscustomobject]@{ ExecutablePath = $api.Binary } }
        Reset-StopFixture @($business, $ai, $api) $processes
        Stop-DevServices
        Assert-Env (($stopFixture.Stopped -join ',') -ceq '303,202,101') 'stop must close API before AI RPC and business RPC'
        Assert-Env (-not (Test-Path -LiteralPath $devManifest)) 'successful stop must remove its manifest'

        Reset-StopFixture @($business, $api) $processes
        Stop-DevServices
        Assert-Env (($stopFixture.Stopped -join ',') -ceq '303,101') 'legacy two-service manifests must stop API before business RPC'

        Reset-StopFixture @($business, $ai, $api) @{ 101 = $processes[101]; 303 = $processes[303] }
        Stop-DevServices
        Assert-Env (($stopFixture.Stopped -join ',') -ceq '303,101') 'expired PIDs must be skipped without stopping unrelated processes'

        Reset-StopFixture @($business, $ai, $api) @{ 101 = $processes[101]; 202 = $processes[202]; 303 = [pscustomobject]@{ ExecutablePath = (Join-Path $fixtureRoot 'unrelated.exe') } }
        $stopFailure = $null
        try { Stop-DevServices } catch { $stopFailure = $_.Exception.Message }
        Assert-Env ($stopFailure -like '*no longer belongs*' -and $stopFixture.Stopped.Count -eq 0) 'a reused PID in the last manifest entry must reject before stopping any services'
        Assert-Env (Test-Path -LiteralPath $devManifest) 'ownership rejection must preserve the manifest'

        $outside = [pscustomobject]@{ Name = 'api'; PID = 303; Binary = (Join-Path $fixtureRoot 'outside.exe') }
        Reset-StopFixture @($business, $ai, $outside) @{ 101 = $processes[101]; 202 = $processes[202]; 303 = [pscustomobject]@{ ExecutablePath = $outside.Binary } }
        $stopFailure = $null
        try { Stop-DevServices } catch { $stopFailure = $_.Exception.Message }
        Assert-Env ($stopFailure -like '*outside this project*' -and $stopFixture.Stopped.Count -eq 0) 'a later binary outside the runtime must not partially stop earlier services'

        $unknown = New-StopEntry 'unknown' 404
        Reset-StopFixture @($business, $ai, $unknown) $processes
        $stopFailure = $null
        try { Stop-DevServices } catch { $stopFailure = $_.Exception.Message }
        Assert-Env ($stopFailure -like '*service is unknown*' -and $stopFixture.Stopped.Count -eq 0) 'unknown manifest services must reject before stopping any services'
    } $stopDefinition $stopRuntime
    $replacements = @{
        'Stop-DevServices' = @'
function Stop-DevServices { [IO.File]::WriteAllText((Join-Path $devRoot 'stop-called'), 'yes') }
'@
        'Test-DevPort' = 'function Test-DevPort([int]$Port) { return $false }'
        'Start-DevServices' = @'
function Start-DevServices([System.Collections.IDictionary]$ModelEnvironment) {
    $result = @{
        DefaultApplied = [Environment]::GetEnvironmentVariable('GOZERO_ENV_FIXTURE_DEFAULT', 'Process') -ceq 'file-default'
        ProcessWins = [Environment]::GetEnvironmentVariable('GOZERO_ENV_FIXTURE_PRIORITY', 'Process') -ceq 'process-wins'
        KeysHidden = [string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable('DEEPSEEK_API_KEY', 'Process')) -and [string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable('QWEN_API_KEY', 'Process'))
        DeepSeekFromProcess = $ModelEnvironment['DEEPSEEK_API_KEY'] -ceq 'fixture-deepseek'
        QwenFromFile = $ModelEnvironment['QWEN_API_KEY'] -ceq 'fixture-qwen-file'
        Services = @(Get-DevServiceDefinitions)
        Launches = @()
    }
    foreach ($service in $result.Services) {
        $result.Launches += Invoke-DevServiceLaunch -ServiceName $service.Name -ModelEnvironment $ModelEnvironment -Launch {
            @{
                Name = $service.Name
                DeepSeekPresent = -not [string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable('DEEPSEEK_API_KEY', 'Process'))
                QwenPresent = -not [string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable('QWEN_API_KEY', 'Process'))
            }
        }
    }
    [IO.File]::WriteAllText((Join-Path $devRoot 'start-checks.json'), ($result | ConvertTo-Json))
    if ([Environment]::GetEnvironmentVariable('GOZERO_ENV_FIXTURE_FAILURE', 'Process') -eq '1') { throw 'controlled startup failure' }
}
'@
    }
    $functions = @($ast.FindAll({ param($node) $node -is [Management.Automation.Language.FunctionDefinitionAst] }, $true) | Where-Object { $replacements.ContainsKey($_.Name) } | Sort-Object { $_.Extent.StartOffset } -Descending)
    Assert-Env ($functions.Count -eq $replacements.Count) 'fixture must replace every service operation'
    foreach ($function in $functions) {
        $source = $source.Remove($function.Extent.StartOffset, $function.Extent.EndOffset - $function.Extent.StartOffset).Insert($function.Extent.StartOffset, $replacements[$function.Name])
    }
    $fixtureDev = Join-Path $fixtureScripts 'dev.ps1'
    [IO.File]::WriteAllText($fixtureDev, $source, $utf8)
    $fixtureEnv = Join-Path $fixtureRoot '.env.local'
    [IO.File]::WriteAllText($fixtureEnv, ("GOZERO_ENV_FIXTURE_DEFAULT=file-default`n" + 'BAD="fixture-secret'), $utf8)
    Assert-InvalidEnv { & $fixtureDev -Action Restart } $fixtureEnv 2
    Assert-Env (-not (Test-Path -LiteralPath (Join-Path $fixtureRoot 'stop-called'))) 'invalid file must fail before Restart stops anything'
    Assert-Env ($null -eq [Environment]::GetEnvironmentVariable('GOZERO_ENV_FIXTURE_DEFAULT', 'Process')) 'invalid file must not partially apply defaults'
    & $fixtureDev -Action Status
    & $fixtureDev -Action Stop
    Assert-Env (Test-Path -LiteralPath (Join-Path $fixtureRoot 'stop-called')) 'Stop/Status must work without reading the malformed file'
    Remove-Item -LiteralPath (Join-Path $fixtureRoot 'stop-called') -Force

    Remove-Item -LiteralPath $fixtureEnv -Force
    & $fixtureDev -Action Start
    Assert-Env (Test-Path -LiteralPath (Join-Path $fixtureRoot 'start-checks.json')) 'Start must accept a missing file and an empty defaults dictionary'
    [IO.File]::WriteAllText($fixtureEnv, '', $utf8)
    & $fixtureDev -Action Restart
    Assert-Env (Test-Path -LiteralPath (Join-Path $fixtureRoot 'stop-called')) 'Restart must accept an empty file before stopping and starting'
    Assert-Env ($null -eq [Environment]::GetEnvironmentVariable('GOZERO_ENV_FIXTURE_DEFAULT', 'Process') -and [Environment]::GetEnvironmentVariable('DEEPSEEK_API_KEY', 'Process') -ceq 'fixture-deepseek') 'missing and empty files must preserve caller environment'
    Remove-Item -LiteralPath (Join-Path $fixtureRoot 'stop-called') -Force

    [Environment]::SetEnvironmentVariable('QWEN_API_KEY', $null, [EnvironmentVariableTarget]::Process)
    [IO.File]::WriteAllText($fixtureEnv, @'
GOZERO_ENV_FIXTURE_DEFAULT=file-default
GOZERO_ENV_FIXTURE_PRIORITY=file-loses
DEEPSEEK_API_KEY=fixture-deepseek-file
QWEN_API_KEY=fixture-qwen-file
'@, $utf8)
    & $fixtureDev -Action Start
    $checks = [IO.File]::ReadAllText((Join-Path $fixtureRoot 'start-checks.json')) | ConvertFrom-Json
    Assert-Env ($checks.DefaultApplied -and $checks.ProcessWins) 'entrypoint must apply file defaults while retaining Process priority'
    Assert-Env ($checks.KeysHidden -and $checks.DeepSeekFromProcess -and $checks.QwenFromFile) 'only AI RPC launch may receive effective model keys'
    Assert-Env ($checks.Services.Count -eq 3 -and ($checks.Services.Name -join ',') -ceq 'rpc,ai-rpc,api') 'services must start in business RPC, AI RPC, API order'
    Assert-Env (($checks.Services.Port -join ',') -ceq '6001,6002,7001') 'three managed services must have distinct expected ports'
    Assert-Env ($checks.Services[1].Package -ceq './application/ai/rpc' -and $checks.Services[1].Config -ceq 'application/ai/rpc/etc/ai.yaml') 'AI service must use its independent package and config'
    Assert-Env ($checks.Launches.Count -eq 3 -and ($checks.Launches.Name -join ',') -ceq 'rpc,ai-rpc,api') 'entrypoint must route all three launch environments'
    Assert-Env (-not $checks.Launches[0].DeepSeekPresent -and -not $checks.Launches[0].QwenPresent -and -not $checks.Launches[2].DeepSeekPresent -and -not $checks.Launches[2].QwenPresent) 'business RPC and API launch must both hide model keys'
    Assert-Env ($checks.Launches[1].DeepSeekPresent -and $checks.Launches[1].QwenPresent) 'AI RPC launch must receive both effective model keys'
    Assert-Env ($null -eq [Environment]::GetEnvironmentVariable('GOZERO_ENV_FIXTURE_DEFAULT', 'Process') -and $null -eq [Environment]::GetEnvironmentVariable('QWEN_API_KEY', 'Process')) 'successful entrypoint must restore newly added variables'
    Assert-Env ([Environment]::GetEnvironmentVariable('DEEPSEEK_API_KEY', 'Process') -ceq 'fixture-deepseek') 'successful entrypoint must preserve caller keys'
    [Environment]::SetEnvironmentVariable('GOZERO_ENV_FIXTURE_FAILURE', '1', [EnvironmentVariableTarget]::Process)
    $failure = $null
    try { & $fixtureDev -Action Restart } catch { $failure = $_.Exception.Message }
    Assert-Env ($failure -eq 'controlled startup failure' -and (Test-Path -LiteralPath (Join-Path $fixtureRoot 'stop-called'))) 'valid Restart must stop before starting and surface startup failure'
    Assert-Env ($null -eq [Environment]::GetEnvironmentVariable('GOZERO_ENV_FIXTURE_DEFAULT', 'Process') -and $null -eq [Environment]::GetEnvironmentVariable('QWEN_API_KEY', 'Process')) 'failed entrypoint must restore all file defaults'
    Assert-Env ([Environment]::GetEnvironmentVariable('DEEPSEEK_API_KEY', 'Process') -ceq 'fixture-deepseek') 'failed entrypoint must preserve caller keys'
    # Finally must release the runtime operation lock even when startup fails.
    & $fixtureDev -Action Status
    Write-Host ('PASS: ' + $script:envChecks + ' environment regression assertions (temporary fixtures only).')
} finally {
    Restore-DevEnvironment -Snapshot $originalProcess
    $expectedPrefix = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar + 'go-zero-env-'
    if (-not $fixtureRoot.StartsWith($expectedPrefix, [StringComparison]::OrdinalIgnoreCase)) { throw 'Refusing to remove a fixture outside its temporary directory.' }
    if (Test-Path -LiteralPath $fixtureRoot) {
        $resolvedFixtureRoot = [IO.Path]::GetFullPath((Resolve-Path -LiteralPath $fixtureRoot).ProviderPath)
        if (-not [string]::Equals($resolvedFixtureRoot, $fixtureRoot, [StringComparison]::OrdinalIgnoreCase)) {
            throw 'Refusing to remove a fixture whose resolved path differs from the directory created by this run.'
        }
        Remove-Item -LiteralPath $resolvedFixtureRoot -Recurse -Force
    }
}
