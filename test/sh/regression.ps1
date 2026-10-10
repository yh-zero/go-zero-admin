#requires -Version 5.1
# Isolated regression suite for this project.
#
#   .\test\sh\regression.ps1 -Scope env   环境变量解析与模型密钥隔离（不需要 Docker）
#   .\test\sh\regression.ps1 -Scope db    MySQL 迁移 + 全量 SQL 导入（需要开发用 MySQL 容器）
#   .\test\sh\regression.ps1             两套都跑（默认）
#
# 两个场景都只使用临时隔离资源；db 场景只操作新建的临时数据库。
[CmdletBinding()]
param([ValidateSet('env', 'db', 'all')][string]$Scope = 'all')

$ErrorActionPreference = 'Stop'

function Invoke-EnvRegression {
$ErrorActionPreference = 'Stop'
$fixtureRoot = Join-Path ([IO.Path]::GetTempPath()) ('go-zero-env-' + [guid]::NewGuid().ToString('N'))
$fixtureRoot = [IO.Path]::GetFullPath($fixtureRoot)
$utf8 = New-Object Text.UTF8Encoding($false)
$originalProcess = @{}
$fixtureNames = @('API_KEY_DEEPSEEK', 'API_KEY_QWEN', 'GOZERO_ENV_FIXTURE_DEFAULT', 'GOZERO_ENV_FIXTURE_PRIORITY', 'GOZERO_ENV_FIXTURE_FAILURE')
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

    [Environment]::SetEnvironmentVariable('API_KEY_DEEPSEEK', 'fixture-deepseek', [EnvironmentVariableTarget]::Process)
    [Environment]::SetEnvironmentVariable('API_KEY_QWEN', 'fixture-qwen', [EnvironmentVariableTarget]::Process)
    $modelValues = Get-DevModelEnvironment
    $childScript = Join-Path $fixtureRoot 'child.ps1'
    [IO.File]::WriteAllText($childScript, @'
param([string]$Output)
$result = @{
    DeepSeekPresent = -not [string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable('API_KEY_DEEPSEEK', 'Process'))
    QwenPresent = -not [string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable('API_KEY_QWEN', 'Process'))
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
        Assert-Env ($null -eq [Environment]::GetEnvironmentVariable('API_KEY_DEEPSEEK', 'Process')) 'nested AI RPC injection must restore the key-free outer environment'
    }
    Assert-Env ([Environment]::GetEnvironmentVariable('API_KEY_DEEPSEEK', 'Process') -ceq 'fixture-deepseek') 'key filter must restore caller after success'
    try { Invoke-DevWithoutModelKeys -Action { throw 'controlled action failure' } } catch { }
    Assert-Env ([Environment]::GetEnvironmentVariable('API_KEY_QWEN', 'Process') -ceq 'fixture-qwen') 'key filter must restore caller after failure'

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
        KeysHidden = [string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable('API_KEY_DEEPSEEK', 'Process')) -and [string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable('API_KEY_QWEN', 'Process'))
        DeepSeekFromProcess = $ModelEnvironment['API_KEY_DEEPSEEK'] -ceq 'fixture-deepseek'
        QwenFromFile = $ModelEnvironment['API_KEY_QWEN'] -ceq 'fixture-qwen-file'
        Services = @(Get-DevServiceDefinitions)
        Launches = @()
    }
    foreach ($service in $result.Services) {
        $result.Launches += Invoke-DevServiceLaunch -ServiceName $service.Name -ModelEnvironment $ModelEnvironment -Launch {
            @{
                Name = $service.Name
                DeepSeekPresent = -not [string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable('API_KEY_DEEPSEEK', 'Process'))
                QwenPresent = -not [string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable('API_KEY_QWEN', 'Process'))
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
    Assert-Env ($null -eq [Environment]::GetEnvironmentVariable('GOZERO_ENV_FIXTURE_DEFAULT', 'Process') -and [Environment]::GetEnvironmentVariable('API_KEY_DEEPSEEK', 'Process') -ceq 'fixture-deepseek') 'missing and empty files must preserve caller environment'
    Remove-Item -LiteralPath (Join-Path $fixtureRoot 'stop-called') -Force

    [Environment]::SetEnvironmentVariable('API_KEY_QWEN', $null, [EnvironmentVariableTarget]::Process)
    [IO.File]::WriteAllText($fixtureEnv, @'
GOZERO_ENV_FIXTURE_DEFAULT=file-default
GOZERO_ENV_FIXTURE_PRIORITY=file-loses
API_KEY_DEEPSEEK=fixture-deepseek-file
API_KEY_QWEN=fixture-qwen-file
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
    Assert-Env ($null -eq [Environment]::GetEnvironmentVariable('GOZERO_ENV_FIXTURE_DEFAULT', 'Process') -and $null -eq [Environment]::GetEnvironmentVariable('API_KEY_QWEN', 'Process')) 'successful entrypoint must restore newly added variables'
    Assert-Env ([Environment]::GetEnvironmentVariable('API_KEY_DEEPSEEK', 'Process') -ceq 'fixture-deepseek') 'successful entrypoint must preserve caller keys'
    [Environment]::SetEnvironmentVariable('GOZERO_ENV_FIXTURE_FAILURE', '1', [EnvironmentVariableTarget]::Process)
    $failure = $null
    try { & $fixtureDev -Action Restart } catch { $failure = $_.Exception.Message }
    Assert-Env ($failure -eq 'controlled startup failure' -and (Test-Path -LiteralPath (Join-Path $fixtureRoot 'stop-called'))) 'valid Restart must stop before starting and surface startup failure'
    Assert-Env ($null -eq [Environment]::GetEnvironmentVariable('GOZERO_ENV_FIXTURE_DEFAULT', 'Process') -and $null -eq [Environment]::GetEnvironmentVariable('API_KEY_QWEN', 'Process')) 'failed entrypoint must restore all file defaults'
    Assert-Env ([Environment]::GetEnvironmentVariable('API_KEY_DEEPSEEK', 'Process') -ceq 'fixture-deepseek') 'failed entrypoint must preserve caller keys'
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
}

function Invoke-DbRegression {
$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
$id = [guid]::NewGuid().ToString('N')
$database = 'gozero_test_' + $id
$testRoot = Join-Path $root ('bin/db-regression-' + $id)
$helper = '/tmp/gozero-db-test-' + $id + '.sh'
$baseline = '/tmp/gozero-db-test-' + $id + '.sql'
$utf8 = New-Object Text.UTF8Encoding($false)
function Docker([string[]]$Arguments) {
    $output = & docker.exe @Arguments
    if ($LASTEXITCODE -ne 0) { throw ('Docker failed: ' + ($Arguments -join ' ')) }
    return $output
}
function Query([string]$Sql) { Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'query', $Sql) }
function Migrate([bool]$ShouldSucceed = $true) {
    $ErrorActionPreference = 'Continue'
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $testRoot 'test/sh/db.ps1') -Action Migrate -Database $database 2>&1 | Out-Host
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = 'Stop'
    if (($exitCode -eq 0) -ne $ShouldSucceed) { throw 'Unexpected migration exit status.' }
}
function Expect([string]$Sql, [string]$Expected) {
    $actual = (Query $Sql | Out-String).Trim()
    if ($actual -cne $Expected) { throw "Expected $Expected, got $actual" }
}
function Expect-Uninitialized([string]$Action) {
    $ErrorActionPreference = 'Continue'
    $output = @(& powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $testRoot 'test/sh/db.ps1') -Action $Action -Database $database 2>&1)
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = 'Stop'
    if ($exitCode -eq 0 -or ($output | Out-String) -notmatch 'data/db/gozero-admin[.]sql') {
        throw ('Uninitialized database must refuse ' + $Action + ' and explain the manual SQL import.')
    }
    Expect 'SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE();' '0'
}
$container = (Docker @('compose', '-f', (Join-Path $root 'docker-compose.yml'), 'ps', '-q', 'mysql') | Out-String).Trim()
if (-not $container) { throw 'Start development MySQL first.' }
New-Item -ItemType Directory -Path (Join-Path $testRoot 'test/sh'), (Join-Path $testRoot 'data/db/migrations') -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $root 'docker-compose.yml') -Destination $testRoot
Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'db.ps1') -Destination (Join-Path $testRoot 'test/sh')
[IO.File]::WriteAllText((Join-Path $testRoot 'test/sh/mysql-client.sh'), [IO.File]::ReadAllText((Join-Path $PSScriptRoot 'mysql-client.sh')).Replace("`r`n", "`n"), $utf8)
$compatName = '20261003_frontend_routes_compat.sql'
$agentName = '20261003_zz_ai_agent.sql'
Get-ChildItem -LiteralPath (Join-Path $root 'data/db/migrations') -Filter '*.sql' | Where-Object { $_.Name -notin @($compatName, $agentName) } | Copy-Item -Destination (Join-Path $testRoot 'data/db/migrations')
$created = $false
try {
    Docker @('cp', (Join-Path $testRoot 'test/sh/mysql-client.sh'), ($container + ':' + $helper)) | Out-Null
    Docker @('exec', $container, 'sh', $helper, 'query', "CREATE DATABASE $database CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;") | Out-Null
    $created = $true
    Expect-Uninitialized 'Status'
    Expect-Uninitialized 'Migrate'
    Docker @('cp', (Join-Path $root 'data/db/archive/gozero-admin-20240129.sql'), ($container + ':' + $baseline)) | Out-Null
    Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'run', $baseline) | Out-Null
    # Reproduce the old 13-migration bug: the legacy seed misses internal //.
    Query "INSERT INTO sys_base_menus (created_at,updated_at,parent_id,path,name,component,title,hidden) VALUES(NOW(3),NOW(3),0,'/admin//audit','custom-double-audit','views/custom/audit.vue','Custom double audit',0); INSERT INTO sys_authority_menus(sys_base_menu_id,sys_authority_authority_id) SELECT id,1 FROM sys_base_menus WHERE name='custom-double-audit';" | Out-Null
    Migrate
    $count = @(Get-ChildItem -LiteralPath (Join-Path $testRoot 'data/db/migrations') -Filter '*.sql').Count
    Expect 'SELECT COUNT(*) FROM schema_migrations;' ([string]$count)
    Migrate
    Expect 'SELECT COUNT(*) FROM schema_migrations;' ([string]$count)
    Expect "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND non_unique=0 AND index_name IN ('uk_sys_menus_active_name','uk_sys_menus_active_path','uk_sys_apis_active_route','uk_sys_dictionaries_active_type','uk_sys_dictionary_info_active_value') AND seq_in_index=1;" '5'
    Expect "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='sys_users' AND column_name='session_version';" '1'
    Expect "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name IN ('sys_policy_versions','sys_audit_logs','sys_departments','sys_positions','sys_user_departments','sys_user_positions','sys_role_data_scopes','sys_role_scope_departments','sys_file_resources','sys_file_references','sys_device_sessions');" '11'
    Expect "SELECT COUNT(*) FROM sys_apis WHERE deleted_at IS NULL AND (path LIKE '/v1/sys/organization/%' OR path LIKE '/v1/sys/files/%' OR path LIKE '/v1/sys/session/admin/%' OR path='/v1/sys/audit/getAuditLogList');" '20'
    Expect "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0='1' AND (v1 LIKE '/v1/sys/organization/%' OR v1 LIKE '/v1/sys/files/%' OR v1 LIKE '/v1/sys/session/admin/%' OR v1='/v1/sys/audit/getAuditLogList');" '20'
    Expect "SELECT COUNT(*) FROM sys_policy_versions WHERE id=1 AND version>0;" '1'
    Expect "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND ((table_name='sys_file_resources' AND column_name='object_key') OR (table_name='sys_file_references' AND column_name IN ('object_type','object_id'))) AND collation_name='utf8mb4_bin';" '3'
    Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name IN ('audit','files','sessions','organization-departments','organization-positions') AND deleted_at IS NULL;" '5'
    Expect "SELECT COUNT(*) FROM sys_authority_menus granted JOIN sys_base_menus menu ON menu.id=granted.sys_base_menu_id WHERE granted.sys_authority_authority_id=1 AND menu.name IN ('superAdmin','audit','files','sessions','organization-departments','organization-positions') AND menu.deleted_at IS NULL;" '6'
    Expect "SELECT COUNT(*) FROM sys_authority_btns granted JOIN sys_base_menu_btns button ON button.id=granted.sys_base_menu_btn_id JOIN sys_base_menus menu ON menu.id=granted.sys_menu_id WHERE granted.authority_id=1 AND menu.deleted_at IS NULL AND button.deleted_at IS NULL AND (menu.name IN ('files','sessions','organization-departments','organization-positions') OR (menu.name='user' AND button.name='membership') OR (menu.name='authority' AND button.name='dataScope'));" '12'
    $canonicalAudit = "SELECT COUNT(*) FROM sys_base_menus menu LEFT JOIN sys_base_menus parent ON parent.id=menu.parent_id WHERE menu.deleted_at IS NULL AND LOWER(TRIM(TRAILING '/' FROM REGEXP_REPLACE(IF(LEFT(menu.path,1)='/',menu.path,CONCAT('/',parent.path,'/',menu.path)), '/{2,}', '/')))='/admin/audit';"
    Expect $canonicalAudit '2'
    # Also occupy the first fallback through a canonical alias. The repair must
    # detect it and choose its next candidate without changing either custom row.
    Query "INSERT INTO sys_base_menus (created_at,updated_at,parent_id,path,name,component,title,hidden) SELECT NOW(3),NOW(3),0,CONCAT('/admin//audit-module-',id),'custom-fallback-audit','views/custom/audit.vue','Custom fallback audit',0 FROM sys_base_menus WHERE name='audit' AND deleted_at IS NULL;" | Out-Null
    Copy-Item -LiteralPath (Join-Path $root ('data/db/migrations/' + $compatName)) -Destination (Join-Path $testRoot 'data/db/migrations')
    Migrate
    $count++
    Expect 'SELECT COUNT(*) FROM schema_migrations;' ([string]$count)
    Expect $canonicalAudit '1'
    Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='audit' AND path=CONCAT('audit-module-',id,'-1') AND component='views/business/system/audit/index.vue' AND deleted_at IS NULL;" '1'
    Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='custom-double-audit' AND path='/admin//audit' AND component='views/custom/audit.vue' AND title='Custom double audit' AND deleted_at IS NULL;" '1'
    Expect "SELECT COUNT(*) FROM sys_authority_menus granted JOIN sys_base_menus menu ON menu.id=granted.sys_base_menu_id WHERE granted.sys_authority_authority_id=1 AND menu.name IN ('audit','custom-double-audit') AND menu.deleted_at IS NULL;" '2'
    Expect "SELECT COUNT(*) FROM sys_base_menus WHERE deleted_at IS NULL AND ((name='files' AND path='files') OR (name='sessions' AND path='sessions') OR (name='organization-departments' AND path='organization/departments') OR (name='organization-positions' AND path='organization/positions'));" '4'
    $compatMigration = '/tmp/gozero-frontend-route-compat-' + $id + '.sql'
    Docker @('cp', (Join-Path $root ('data/db/migrations/' + $compatName)), ($container + ':' + $compatMigration)) | Out-Null
    try {
        Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'run', $compatMigration) | Out-Host
        Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='audit' AND path=CONCAT('audit-module-',id,'-1') AND deleted_at IS NULL;" '1'
        Expect "SELECT COUNT(*) FROM sys_audit_logs WHERE action='repairFrontendRoute';" '1'
        # A customized seed remains untouched, with an explicit SKIPPED report.
        Query "UPDATE sys_base_menus SET title='Custom sessions' WHERE name='sessions' AND deleted_at IS NULL; INSERT INTO sys_base_menus (created_at,updated_at,parent_id,path,name,component,title,hidden) VALUES(NOW(3),NOW(3),0,'/admin//sessions','custom-double-sessions','views/custom/sessions.vue','Custom sessions route',0);" | Out-Null
        $compatOutput = @(Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'run', $compatMigration))
        $compatOutput | Out-Host
        if (-not @($compatOutput | Where-Object { $_ -match '^sessions\s+SKIPPED:' }).Count) { throw 'Customized conflict was not explicitly reported as SKIPPED.' }
        Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='sessions' AND path='sessions' AND title='Custom sessions' AND deleted_at IS NULL;" '1'
        Expect "SELECT COUNT(*) FROM sys_authority_btns granted JOIN sys_base_menu_btns button ON button.id=granted.sys_base_menu_btn_id JOIN sys_base_menus menu ON menu.id=granted.sys_menu_id WHERE granted.authority_id=1 AND menu.name='sessions' AND button.name='revoke' AND button.deleted_at IS NULL;" '1'
        Expect "SELECT COUNT(*) FROM sys_audit_logs WHERE action='repairFrontendRoute';" '1'
    } finally { Docker @('exec', $container, 'rm', '-f', '--', $compatMigration) | Out-Null }
    $menuMigration = '/tmp/gozero-frontend-modules-' + $id + '.sql'
    Docker @('cp', (Join-Path $root 'data/db/migrations/20261003_frontend_modules.sql'), ($container + ':' + $menuMigration)) | Out-Null
    try {
        # Re-running the actual SQL must not add duplicates or overwrite customization.
        Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'run', $menuMigration) | Out-Host
        Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name IN ('audit','files','sessions','organization-departments','organization-positions') AND deleted_at IS NULL;" '5'
        Expect "SELECT COUNT(*) FROM sys_audit_logs WHERE action='seedFrontendModules' AND object='20261003_frontend_modules.sql';" '1'
        Query "UPDATE sys_base_menus SET component='views/custom/storage.vue',path='custom-storage',title='Custom storage' WHERE name='files' AND deleted_at IS NULL; UPDATE sys_base_menus SET name='custom-audit',path='old-audit' WHERE name='audit' AND deleted_at IS NULL; INSERT INTO sys_base_menus (created_at,updated_at,parent_id,path,name,component,title,hidden) VALUES(NOW(3),NOW(3),0,'/admin/audit','custom-absolute-audit','views/custom/audit.vue','Custom audit',0);" | Out-Null
        Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'run', $menuMigration) | Out-Host
        Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='files' AND component='views/custom/storage.vue' AND path='custom-storage' AND title='Custom storage' AND deleted_at IS NULL;" '1'
        Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='audit' AND deleted_at IS NULL;" '0'
    } finally { Docker @('exec', $container, 'rm', '-f', '--', $menuMigration) | Out-Null }
    # Install the agent module after reproducing the old route compatibility bug.
    # Existing ordinary-role grants must survive initial install and direct reruns.
    Query "INSERT INTO casbin_rule(ptype,v0,v1,v2) VALUES('p','888','/v1/ai/info','GET');" | Out-Null
    Copy-Item -LiteralPath (Join-Path $root ('data/db/migrations/' + $agentName)) -Destination (Join-Path $testRoot 'data/db/migrations')
    Migrate
    $count++
    Expect 'SELECT COUNT(*) FROM schema_migrations;' ([string]$count)
    Expect "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name IN ('sys_ai_conversations','sys_ai_messages','sys_ai_runs');" '3'
    Expect "SELECT COUNT(*) FROM sys_apis WHERE deleted_at IS NULL AND path LIKE '/v1/ai/%';" '6'
    Expect "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 LIKE '/v1/ai/%';" '6'
    Expect "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0='888' AND v1='/v1/ai/info' AND v2='GET';" '1'
    Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='ai-agent' AND path='ai-agent' AND component='views/business/ai-agent/index.vue' AND deleted_at IS NULL;" '1'
    Expect "SELECT COUNT(*) FROM sys_authority_menus granted JOIN sys_base_menus menu ON menu.id=granted.sys_base_menu_id WHERE granted.sys_authority_authority_id=1 AND menu.name IN ('ai-agent','superAdmin') AND menu.deleted_at IS NULL;" '2'
    Expect "SELECT COUNT(*) FROM sys_authority_btns granted JOIN sys_base_menu_btns button ON button.id=granted.sys_base_menu_btn_id JOIN sys_base_menus menu ON menu.id=granted.sys_menu_id WHERE granted.authority_id=1 AND menu.name='ai-agent' AND button.name IN ('run','cancel') AND button.deleted_at IS NULL;" '2'
    Query "INSERT INTO sys_ai_conversations(id,owner_id,title,created_at,updated_at) VALUES('10000000-0000-4000-8000-000000000001',1,'Kept history',NOW(3),NOW(3)); INSERT INTO sys_ai_runs(id,owner_id,authority_id,session_version,request_id,request_hash,conversation_id,sequence,question,status,text,extra_json,queue_expires_at,created_at,updated_at) VALUES('10000000-0000-4000-8000-000000000002',1,1,0,'10000000-0000-4000-8000-000000000003',REPEAT('a',64),'10000000-0000-4000-8000-000000000001',1,'kept question','succeeded','kept answer','[]',NOW(3),NOW(3),NOW(3)); INSERT INTO sys_ai_messages(id,owner_id,conversation_id,run_id,role,content,created_at) VALUES('10000000-0000-4000-8000-000000000004',1,'10000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000002','assistant','kept answer',NOW(3));" | Out-Null
    $agentMigration = '/tmp/gozero-agent-migration-' + $id + '.sql'
    Docker @('cp', (Join-Path $root ('data/db/migrations/' + $agentName)), ($container + ':' + $agentMigration)) | Out-Null
    try {
        $policyVersion = (Query 'SELECT version FROM sys_policy_versions WHERE id=1;' | Out-String).Trim()
        Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'run', $agentMigration) | Out-Host
        Expect 'SELECT version FROM sys_policy_versions WHERE id=1;' $policyVersion
        Expect "SELECT COUNT(*) FROM sys_ai_runs WHERE text='kept answer';" '1'
        Expect "SELECT COUNT(*) FROM sys_ai_messages WHERE content='kept answer';" '1'
        Expect "SELECT COUNT(*) FROM sys_audit_logs WHERE action='seedAgentModule';" '1'
        # Preserve a customized component and report it, without changing grants.
        Query "UPDATE sys_base_menus SET path='custom-ai',component='views/custom/assistant.vue',title='Custom assistant' WHERE name='ai-agent' AND deleted_at IS NULL;" | Out-Null
        $agentOutput = @(Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'run', $agentMigration))
        if (-not @($agentOutput | Where-Object { $_ -match '^ai-agent\s+SKIPPED:' }).Count) { throw 'Customized AI menu was not explicitly reported as SKIPPED.' }
        Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='ai-agent' AND path='custom-ai' AND component='views/custom/assistant.vue' AND title='Custom assistant' AND deleted_at IS NULL;" '1'
        Expect "SELECT COUNT(*) FROM sys_authority_btns granted JOIN sys_base_menu_btns button ON button.id=granted.sys_base_menu_btn_id JOIN sys_base_menus menu ON menu.id=granted.sys_menu_id WHERE granted.authority_id=1 AND menu.name='ai-agent' AND button.name IN ('run','cancel') AND button.deleted_at IS NULL;" '2'
        # Canonical /admin//ai-agent must prevent insertion of a duplicate seed.
        Query "UPDATE sys_base_menus SET name='custom-existing-ai' WHERE name='ai-agent' AND deleted_at IS NULL; INSERT INTO sys_base_menus(created_at,updated_at,parent_id,path,name,component,title,hidden) VALUES(NOW(3),NOW(3),0,'/admin//ai-agent','custom-absolute-ai','views/custom/assistant.vue','Custom absolute AI',0);" | Out-Null
        $agentOutput = @(Docker @('exec', '-e', ('MYSQL_DATABASE=' + $database), $container, 'sh', $helper, 'run', $agentMigration))
        if (-not @($agentOutput | Where-Object { $_ -match '^ai-agent\s+SKIPPED:' }).Count) { throw 'Canonical AI conflict was not explicitly reported as SKIPPED.' }
        Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='ai-agent' AND deleted_at IS NULL;" '0'
        Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name='custom-absolute-ai' AND path='/admin//ai-agent' AND deleted_at IS NULL;" '1'
        Expect "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0='888' AND v1='/v1/ai/info' AND v2='GET';" '1'
    } finally { Docker @('exec', $container, 'rm', '-f', '--', $agentMigration) | Out-Null }
    # The failing batch must stop before later statements and must not be recorded.
    $probe = Join-Path $testRoot 'data/db/migrations/99991230_probe.sql'
    [IO.File]::WriteAllText($probe, "SELECT * FROM intentionally_missing_regression_table;`nINSERT INTO schema_migrations(filename,checksum) VALUES('must_not_run','bad');`n", $utf8)
    Migrate $false
    Expect "SELECT COUNT(*) FROM schema_migrations WHERE filename IN ('99991230_probe.sql','must_not_run');" '0'
    [IO.File]::WriteAllText($probe, "SELECT 1;`n", $utf8)
    Migrate
    Expect "SELECT COUNT(*) FROM schema_migrations WHERE filename='99991230_probe.sql';" '1'
    [IO.File]::WriteAllText($probe, "SELECT 2;`n", $utf8)
    Migrate $false
    Write-Host 'PASS: fresh install, repeat migration, schema checks, frontend seed, old canonical conflict reproduced, compatibility repair, fallback collisions, grants preserved, customized conflict skipped, agent schema/permissions/menu/history preservation, SQL failure stop, retry, checksum guard.'

    # ── 导入回归测试（原 db-import-regression.ps1） ──
    Write-Host 'Running import regression...'
    $importDb = 'gozero_import_test_' + $id
    $importHelper = '/tmp/gozero-import-test-' + $id + '.sh'
    $importSql = '/tmp/gozero-import-test-' + $id + '.sql'
    $importHelperLocal = Join-Path ([IO.Path]::GetTempPath()) ('gozero-import-test-' + $id + '.sh')
    $importCreated = $false
    function Import-Query([string]$Sql) { Docker @('exec', '-e', ('MYSQL_DATABASE=' + $importDb), $container, 'sh', $importHelper, 'query', $Sql) }
    function Import-Expect([string]$Sql, [string]$Expected) {
        $actual = (Import-Query $Sql | Out-String).Trim()
        if ($actual -cne $Expected) { throw ('Import expected ' + $Expected + ', got ' + $actual) }
    }
    try {
        [IO.File]::WriteAllText($importHelperLocal, [IO.File]::ReadAllText((Join-Path $PSScriptRoot 'mysql-client.sh')).Replace("`r`n", "`n"), $utf8)
        Docker @('cp', $importHelperLocal, ($container + ':' + $importHelper)) | Out-Null
        Docker @('exec', $container, 'sh', $importHelper, 'query', ('CREATE DATABASE ' + $importDb + ' CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;')) | Out-Null
        $importCreated = $true
        $fullSql = Join-Path $root 'data/db/gozero-admin.sql'
        if (-not (Test-Path -LiteralPath $fullSql -PathType Leaf)) { throw 'The current full installation SQL is missing.' }
        $sqlText = [IO.File]::ReadAllText($fullSql)
        if ($sqlText -match '(?im)^\s*(CREATE\s+DATABASE|USE\s+)') { throw 'The installation SQL must use the explicitly selected database.' }
        Docker @('cp', $fullSql, ($container + ':' + $importSql)) | Out-Null
        Docker @('exec', '-e', ('MYSQL_DATABASE=' + $importDb), $container, 'sh', '-c', 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --user=root --database="$MYSQL_DATABASE" < "$1"', 'sh', $importSql) | Out-Null
        Import-Expect 'SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE();' '30'
        Import-Expect "SELECT COUNT(*) FROM sys_users WHERE id=1 AND username='admin' AND authority_id=1 AND enable=1 AND deleted_at IS NULL;" '1'
        Import-Expect 'SELECT COUNT(*) FROM sys_users;' '1'
        Import-Expect 'SELECT COUNT(*) FROM sys_user_authority WHERE sys_user_id=1 AND sys_authority_authority_id=1;' '1'
        Import-Expect 'SELECT (SELECT COUNT(*) FROM sys_device_sessions)+(SELECT COUNT(*) FROM sys_audit_logs)+(SELECT COUNT(*) FROM sys_ai_runs)+(SELECT COUNT(*) FROM sys_ai_messages)+(SELECT COUNT(*) FROM sys_ai_conversations)+(SELECT COUNT(*) FROM sys_file_resources)+(SELECT COUNT(*) FROM sys_file_references);' '0'
        Import-Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name IN ('audit','files','sessions','organization-departments','organization-positions','ai-agent') AND deleted_at IS NULL;" '6'
        Import-Expect "SELECT COUNT(*) FROM sys_base_menus WHERE name IN ('test1','test2','test3','test4','test5');" '0'
        Import-Expect 'SELECT COUNT(*) FROM sys_authority_menus grant_row LEFT JOIN sys_base_menus menu ON menu.id=grant_row.sys_base_menu_id WHERE menu.id IS NULL;' '0'
        Import-Expect 'SELECT COUNT(*) FROM sys_authorities;' '1'
        Import-Expect 'SELECT COUNT(*) FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL;' '1'
        Import-Expect 'SELECT COUNT(*) FROM schema_migrations;' '16'
        foreach ($migration in @(Get-ChildItem -LiteralPath (Join-Path $root 'data/db/migrations') -Filter '*.sql')) {
            $normalized = [IO.File]::ReadAllText($migration.FullName).Replace("`r`n", "`n")
            $algorithm = [Security.Cryptography.SHA256]::Create()
            try { $hash = ([BitConverter]::ToString($algorithm.ComputeHash([Text.Encoding]::UTF8.GetBytes($normalized)))).Replace('-', '').ToLowerInvariant() } finally { $algorithm.Dispose() }
            Import-Expect ("SELECT COUNT(*) FROM schema_migrations WHERE filename='" + $migration.Name + "' AND checksum='" + $hash + "';") '1'
        }
        $status = @(& powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $testRoot 'test/sh/db.ps1') -Action Status -Database $importDb 2>&1)
        if ($LASTEXITCODE -ne 0 -or @($status | Where-Object { [string]$_ -match '^PENDING ' }).Count -gt 0) { throw 'A fresh SQL import must be ready without additional migrations.' }
        Import-Query "INSERT INTO sys_audit_logs(event_type,module,action,result,status_code,params) VALUES('operation','import-test','preserveExistingRows','success',200,'{}');" | Out-Null
        $ErrorActionPreference = 'Continue'
        & docker.exe exec -e ('MYSQL_DATABASE=' + $importDb) $container sh $importHelper run $importSql 2>&1 | Out-Null
        $repeatExit = $LASTEXITCODE
        $ErrorActionPreference = 'Stop'
        if ($repeatExit -eq 0) { throw 'Importing over an existing database must fail instead of resetting it.' }
        Import-Expect "SELECT COUNT(*) FROM sys_audit_logs WHERE action='preserveExistingRows';" '1'
        Import-Expect 'SELECT COUNT(*) FROM sys_users;' '1'
        Import-Expect 'SELECT COUNT(*) FROM schema_migrations;' '16'
        Write-Host 'PASS: current SQL imports into a custom empty database, is ready for startup, and preserves existing data on repeat import.'
    } finally {
        if ($importCreated) { Import-Query ('DROP DATABASE ' + $importDb + ';') | Out-Null }
        if ($container) { & docker.exe exec $container rm -f -- $importHelper $importSql | Out-Null }
        if (Test-Path -LiteralPath $importHelperLocal) { Remove-Item -LiteralPath $importHelperLocal -Force }
    }
} finally {
    if ($created) { Docker @('exec', $container, 'sh', $helper, 'query', "DROP DATABASE $database;") | Out-Null }
    Docker @('exec', $container, 'rm', '-f', '--', $helper, $baseline) | Out-Null
    # Keep the test copies and backups under ignored bin/ for failure inspection.
}
}

if ($Scope -in @('env', 'all')) { Invoke-EnvRegression }
if ($Scope -in @('db', 'all')) { Invoke-DbRegression }
