#requires -Version 5.1
[CmdletBinding()] param([string[]]$ComposeFiles)
$ErrorActionPreference = 'Stop'
$backend = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
if (-not $ComposeFiles) { $ComposeFiles = @((Join-Path $backend 'docker-compose.yml')) }
$arguments = @('compose')
$localEnv = Join-Path $backend '.env.local'
if (Test-Path -LiteralPath $localEnv -PathType Leaf) { $arguments += @('--env-file', $localEnv) }
foreach ($file in $ComposeFiles) { $arguments += @('-f', $file) }

function ConvertTo-IPv4Number([string]$Value) {
    $address = $null
    if ($Value -notmatch '^\d{1,3}(\.\d{1,3}){3}$' -or -not [Net.IPAddress]::TryParse($Value, [ref]$address) -or $address.AddressFamily -ne [Net.Sockets.AddressFamily]::InterNetwork) {
        throw '代理网络字段必须使用有效 IPv4 地址。'
    }
    $bytes = $address.GetAddressBytes()
    return [uint64]$bytes[0] * 16777216 + [uint64]$bytes[1] * 65536 + [uint64]$bytes[2] * 256 + [uint64]$bytes[3]
}

function Get-IPv4Range([string]$Value) {
    $parts = $Value.Split('/')
    $bits = 0
    if ($parts.Count -ne 2 -or -not [int]::TryParse($parts[1], [ref]$bits) -or $bits -lt 1 -or $bits -gt 32) {
        throw '代理网络 subnet/ip_range 必须使用有效 IPv4 CIDR。'
    }
    $number = ConvertTo-IPv4Number $parts[0]
    $size = [uint64][Math]::Pow(2, 32 - $bits)
    $first = $number - ($number % $size)
    if ($first -ne $number) { throw '代理网络 subnet/ip_range 必须使用规范的网络起始地址。' }
    return @{ First = $first; Last = $first + $size - 1 }
}

# 只渲染配置，不访问 Docker 引擎；完整 JSON 仅在进程内解析，禁止打印其中的凭据。
Push-Location $backend
try {
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $rendered = @(& docker @arguments config --format json 2>&1)
        $composeExit = $LASTEXITCODE
    } finally { $ErrorActionPreference = $previousPreference }
    if ($composeExit -ne 0) { throw 'Compose 配置解析失败，请核对网络字段和 Compose 文件。' }
    try { $configuration = ($rendered -join "`n") | ConvertFrom-Json } catch { throw 'Compose 配置不是有效 JSON。' }
} finally { Pop-Location }

$networks = @(
    @{ Name = 'default'; Service = 'api'; Variable = 'GOZERO_PLATFORM_PROXY_IP' },
    @{ Name = 'gozero-edge'; Service = 'tenant-rpc'; Variable = 'GOZERO_EDGE_PROXY_IP' }
)
foreach ($item in $networks) {
    $name = $item.Name
    $network = $configuration.networks.PSObject.Properties[$name].Value
    $proxy = $configuration.services.nginx.networks.PSObject.Properties[$name].Value.ipv4_address
    $number = ConvertTo-IPv4Number $proxy
    $injected = $configuration.services.PSObject.Properties[$item.Service].Value.environment.PSObject.Properties[$item.Variable].Value
    if ($injected -ne $proxy) { throw ($name + ' 网络的 Nginx 地址与服务注入的可信代理地址不一致。') }
    # 外部网络的 IPAM 由宿主管理，不能因 Compose 未重复声明而拒绝；部署前另行只读核对实际网络。
    if ($network.external -eq $true) {
        Write-Host ($name + ' 为外部网络，代理注入一致；子网和动态池由部署前运行态核对。')
        continue
    }
    $entries = @($network.ipam.config)
    if ($entries.Count -ne 1) { throw ($name + ' 网络必须明确声明一个 IPv4 subnet 和 ip_range。') }
    $subnet = Get-IPv4Range $entries[0].subnet
    $pool = Get-IPv4Range $entries[0].ip_range
    if ($pool.First -lt $subnet.First -or $pool.Last -gt $subnet.Last) { throw ($name + ' 网络的动态 ip_range 必须位于 subnet 内。') }
    if ($number -le $subnet.First -or $number -ge $subnet.Last) { throw ($name + ' 网络的代理 IP 必须位于 subnet 的可用地址范围内。') }
    $gateway = $entries[0].gateway
    $gatewayNumber = if ([string]::IsNullOrWhiteSpace($gateway)) { $subnet.First + 1 } else { ConvertTo-IPv4Number $gateway }
    if ($number -eq $gatewayNumber) { throw ($name + ' 网络的代理 IP 不能占用显式或默认网关。') }
    if ($number -ge $pool.First -and $number -le $pool.Last) { throw ($name + ' 网络的代理 IP 不能落入动态 ip_range。') }
}
Write-Host 'Compose 代理网络配置检查通过。'
