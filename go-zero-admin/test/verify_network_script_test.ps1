#requires -Version 5.1
$ErrorActionPreference = 'Stop'
$backend = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$fixtureRoot = Join-Path $env:TEMP ('gozero-proxy-network-test-' + [guid]::NewGuid().ToString('N'))
$fixtureBackend = Join-Path $fixtureRoot 'project'
$script = Join-Path $fixtureBackend 'data/scripts/verify-network.ps1'
$defaults = @{
    GOZERO_PLATFORM_SUBNET = '172.20.0.0/16'
    GOZERO_PLATFORM_IP_RANGE = '172.20.128.0/17'
    GOZERO_PLATFORM_PROXY_IP = '172.20.0.3'
    GOZERO_EDGE_SUBNET = '172.21.0.0/16'
    GOZERO_EDGE_IP_RANGE = '172.21.128.0/17'
    GOZERO_EDGE_PROXY_IP = '172.21.0.2'
}
$original = @{}
$fixturePath = Join-Path $fixtureBackend 'network-override.yaml'
$failures = [Collections.Generic.List[string]]::new()
$sentinel = 'network-fixture-value-must-not-be-printed'
foreach ($key in $defaults.Keys) { $original[$key] = [Environment]::GetEnvironmentVariable($key, 'Process') }

function Invoke-NetworkCase([string]$Name, [bool]$Pass, [string[]]$Files, [string]$ExpectedReason) {
    $invokeArguments = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $script)
    if ($Files) { $invokeArguments += @('-ComposeFiles') + $Files }
    $ErrorActionPreference = 'Continue'
    try {
        $output = @(& powershell @invokeArguments 2>&1)
        $code = $LASTEXITCODE
    } finally { $ErrorActionPreference = 'Stop' }
    if (($output -join "`n").Contains($sentinel)) { $failures.Add($Name + '泄漏了配置内容') }
    if (-not $Pass -and $code -ne 0 -and $ExpectedReason -and ($output -join "`n") -notmatch $ExpectedReason) { $failures.Add($Name + '未命中预期网络字段校验') }
    if (($code -eq 0) -ne $Pass) {
        $failures.Add($Name + '的网络校验结果不符合预期，退出码=' + $code)
        Write-Host ($Name + '：结果不符合预期')
    } else { Write-Host ($Name + '：符合预期') }
}

try {
    New-Item -ItemType Directory -Path (Split-Path -Parent $script) -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $backend 'data/scripts/verify-network.ps1') -Destination $script
    Copy-Item -LiteralPath (Join-Path $backend 'docker-compose.yml') -Destination (Join-Path $fixtureBackend 'docker-compose.yml')
    $cases = @(
        @{ Name = '默认网络'; Values = @{}; Pass = $true },
        @{ Name = '整体覆盖'; Values = @{ GOZERO_PLATFORM_SUBNET = '172.28.0.0/16'; GOZERO_PLATFORM_IP_RANGE = '172.28.128.0/17'; GOZERO_PLATFORM_PROXY_IP = '172.28.0.3'; GOZERO_EDGE_SUBNET = '172.29.0.0/16'; GOZERO_EDGE_IP_RANGE = '172.29.128.0/17'; GOZERO_EDGE_PROXY_IP = '172.29.0.2' }; Pass = $true },
        @{ Name = '代理地址越出子网'; Values = @{ GOZERO_PLATFORM_PROXY_IP = '172.30.0.3' }; Pass = $false },
        @{ Name = '代理落入动态池'; Values = @{ GOZERO_EDGE_PROXY_IP = '172.21.128.9' }; Pass = $false },
        @{ Name = '动态池越出子网'; Values = @{ GOZERO_PLATFORM_IP_RANGE = '172.30.128.0/17' }; Pass = $false }
    )
    foreach ($case in $cases) {
        foreach ($key in $defaults.Keys) { [Environment]::SetEnvironmentVariable($key, $defaults[$key], 'Process') }
        foreach ($key in $case.Values.Keys) { [Environment]::SetEnvironmentVariable($key, $case.Values[$key], 'Process') }
        Invoke-NetworkCase $case.Name $case.Pass
    }

    foreach ($key in $defaults.Keys) { [Environment]::SetEnvironmentVariable($key, $null, 'Process') }
    $fixtureEnv = Join-Path $fixtureBackend '.env.local'
    [IO.File]::WriteAllText($fixtureEnv, "GOZERO_PLATFORM_PROXY_IP=172.30.0.3`nGOZERO_NETWORK_TEST_SECRET=$sentinel`n", [Text.UTF8Encoding]::new($false))
    Invoke-NetworkCase '仅本地env文件的非法覆盖必须被加载' $false @() '代理 IP'
    Remove-Item -LiteralPath $fixtureEnv
    foreach ($key in $defaults.Keys) { [Environment]::SetEnvironmentVariable($key, $defaults[$key], 'Process') }
    $env:GOZERO_PLATFORM_PROXY_IP = '172.20.0.1'
    Invoke-NetworkCase '代理占用Docker默认网关' $false @() '网关'
    $env:GOZERO_PLATFORM_PROXY_IP = '172.20.0.9'
    $gatewayFixture = @{
        services = @{
            nginx = @{ image = 'nginx:1.27-alpine'; networks = @{ default = @{ ipv4_address = '172.20.0.9' }; 'gozero-edge' = @{ ipv4_address = '172.21.0.2' } } }
            api = @{ image = 'busybox'; environment = @{ GOZERO_PLATFORM_PROXY_IP = '172.20.0.9' } }
            'tenant-rpc' = @{ image = 'busybox'; environment = @{ GOZERO_EDGE_PROXY_IP = '172.21.0.2' } }
        }
        networks = @{
            default = @{ ipam = @{ config = @(@{ subnet = '172.20.0.0/16'; ip_range = '172.20.128.0/17'; gateway = '172.20.0.9' }) } }
            'gozero-edge' = @{ ipam = @{ config = @(@{ subnet = '172.21.0.0/16'; ip_range = '172.21.128.0/17' }) } }
        }
    }
    [IO.File]::WriteAllText($fixturePath, ($gatewayFixture | ConvertTo-Json -Depth 15), [Text.UTF8Encoding]::new($false))
    Invoke-NetworkCase '代理占用显式网关' $false @($fixturePath) '网关'
    $gatewayFixture.services.nginx.networks.default.ipv4_address = '172.20.0.3'
    $gatewayFixture.services.api.environment.GOZERO_PLATFORM_PROXY_IP = '172.20.0.3'
    [IO.File]::WriteAllText($fixturePath, ($gatewayFixture | ConvertTo-Json -Depth 15), [Text.UTF8Encoding]::new($false))
    Invoke-NetworkCase '显式网关与独立代理合法' $true @($fixturePath)

    $externalFixture = @'
services:
  nginx:
    image: nginx:1.27-alpine
    networks:
      default:
        ipv4_address: 172.20.0.3
      gozero-edge:
        ipv4_address: 172.21.0.2
  api:
    image: busybox
    environment:
      GOZERO_PLATFORM_PROXY_IP: 172.20.0.3
  tenant-rpc:
    image: busybox
    environment:
      GOZERO_EDGE_PROXY_IP: 172.21.0.2
networks:
  default:
    external: true
    name: gozero-proxy-test-default
  gozero-edge:
    external: true
    name: gozero-proxy-test-edge
'@
    [IO.File]::WriteAllText($fixturePath, $externalFixture, [Text.UTF8Encoding]::new($false))
    Invoke-NetworkCase '外部网络无 IPAM' $true @($fixturePath)
} finally {
    foreach ($key in $original.Keys) { [Environment]::SetEnvironmentVariable($key, $original[$key], 'Process') }
    $resolvedRoot = [IO.Path]::GetFullPath($fixtureRoot)
    $tempPrefix = [IO.Path]::GetFullPath($env:TEMP).TrimEnd('\') + '\gozero-proxy-network-test-'
    if (-not $resolvedRoot.StartsWith($tempPrefix, [StringComparison]::OrdinalIgnoreCase)) { throw '拒绝清理测试临时目录以外的路径' }
    if (Test-Path -LiteralPath $resolvedRoot) { Remove-Item -LiteralPath $resolvedRoot -Recurse -Force }
}
if ($failures.Count -gt 0) { throw ($failures -join '；') }
exit 0
