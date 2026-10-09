#requires -Version 5.1
[CmdletBinding()]
param(
    [string]$FrontendRoot,
    [string]$OutputRoot
)

$ErrorActionPreference = 'Stop'
$packageBackendRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
$packageAllowedRoot = [IO.Path]::GetFullPath((Join-Path $packageBackendRoot 'bin/cloud-releases'))
if (-not $FrontendRoot) { $FrontendRoot = Join-Path $packageBackendRoot '../go-zero-admin-vben' }
if (-not $OutputRoot) { $OutputRoot = $packageAllowedRoot }
$packageFrontendRoot = [IO.Path]::GetFullPath($FrontendRoot)
$packageOutputRoot = [IO.Path]::GetFullPath($OutputRoot)

# Never replace an existing release or write through a junction outside bin.
$packageAllowedPrefix = $packageAllowedRoot.TrimEnd([char]92, [char]47) + [IO.Path]::DirectorySeparatorChar
if (-not $packageOutputRoot.Equals($packageAllowedRoot, [StringComparison]::OrdinalIgnoreCase) -and
    -not $packageOutputRoot.StartsWith($packageAllowedPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'OutputRoot must stay inside this backend repository''s bin/cloud-releases directory.'
}
$packageAncestor = $packageOutputRoot
while ($packageAncestor -and -not $packageAncestor.Equals($packageBackendRoot, [StringComparison]::OrdinalIgnoreCase)) {
    if (Test-Path -LiteralPath $packageAncestor) {
        $packageItem = Get-Item -LiteralPath $packageAncestor -Force
        if (-not $packageItem.PSIsContainer -or ($packageItem.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
            throw 'OutputRoot and its ancestors must be regular directories, without symbolic links or junctions.'
        }
    }
    $packageAncestor = Split-Path -Parent $packageAncestor
}
if (-not (Test-Path -LiteralPath (Join-Path $packageFrontendRoot 'apps/web-antdv-next/package.json') -PathType Leaf)) {
    throw 'FrontendRoot must contain the go-zero-admin-vben apps/web-antdv-next application.'
}
foreach ($packageTool in @('go', 'pnpm', 'tar')) {
    if (-not (Get-Command $packageTool -ErrorAction SilentlyContinue)) {
        throw ('Required packaging tool is unavailable: ' + $packageTool)
    }
}
foreach ($packageRequired in @(
    'docker/Dockerfile.prebuilt', 'docker/entrypoint.sh', 'docker/api.yaml.template',
    'docker/rpc.yaml.template', 'docker/ai-rpc.yaml.template', 'test/sh/db.sh',
    'test/sh/mysql-client.sh', 'data/db/gozero-admin.sql',
    'data/db/operator/20261008_permission_admin_recovery_bootstrap.sql', '.dockerignore', 'README.md',
    'docker/demo-rebuild.py', 'QUICK_RECOVERY.md'
)) {
    if (-not (Test-Path -LiteralPath (Join-Path $packageBackendRoot $packageRequired) -PathType Leaf)) {
        throw ('Required package source is missing: ' + $packageRequired)
    }
}
. (Join-Path $PSScriptRoot 'env.ps1')

function Copy-DemoPackageFile {
    param([string]$Source, [string]$Destination)
    $sourceItem = Get-Item -LiteralPath $Source -Force
    if ($sourceItem.PSIsContainer -or ($sourceItem.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
        throw 'Package input files must be regular files, without symbolic links.'
    }
    [IO.Directory]::CreateDirectory((Split-Path -Parent $Destination)) | Out-Null
    if ($sourceItem.Extension -match '^\.(sh|py|template|yml|yaml)$' -or
        $sourceItem.Name -in @('Dockerfile', 'Dockerfile.prebuilt', 'Caddyfile.demo')) {
        # A Windows checkout may use CRLF. Container entrypoints need UTF-8,
        # without a BOM, and LF even when Git has autocrlf enabled locally.
        $packageTextEncoding = New-Object Text.UTF8Encoding($false, $true)
        $packageText = [IO.File]::ReadAllText($Source, $packageTextEncoding)
        $packageText = $packageText.TrimStart([char]0xFEFF).Replace("`r`n", "`n").Replace("`r", "`n")
        [IO.File]::WriteAllText($Destination, $packageText, $packageTextEncoding)
    } else {
        Copy-Item -LiteralPath $Source -Destination $Destination -ErrorAction Stop
    }
}

function Copy-DemoPackageTree {
    param([string]$Source, [string]$Destination, [scriptblock]$Include)
    $sourceRoot = [IO.Path]::GetFullPath($Source).TrimEnd([char]92, [char]47)
    $sourceItem = Get-Item -LiteralPath $sourceRoot -Force
    if (-not $sourceItem.PSIsContainer -or ($sourceItem.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
        throw 'Package input directories must be regular directories, without symbolic links or junctions.'
    }
    $sourceEntries = @(Get-ChildItem -LiteralPath $sourceRoot -Recurse -Force)
    foreach ($entry in $sourceEntries) {
        if ($entry.Attributes -band [IO.FileAttributes]::ReparsePoint) {
            throw 'Package input trees must not contain symbolic links or junctions.'
        }
        if ($entry.PSIsContainer) { continue }
        $relative = $entry.FullName.Substring($sourceRoot.Length + 1)
        # Template names such as deploy.env.template are public source files;
        # actual .env* files and development output never enter the payload.
        if ($relative -match '(^|[\\/])(\.env[^\\/]*|\.git|node_modules|\.cache)([\\/]|$)' -or
            $entry.Name -match '^dump-.*\.sql$' -or $entry.Extension -match '^\.(log|bak|exe|test|out)$') { continue }
        if ($Include -and -not (& $Include $entry)) { continue }
        Copy-DemoPackageFile -Source $entry.FullName -Destination (Join-Path $Destination $relative)
    }
}

$packageName = (Get-Date -Format 'yyyyMMdd-HHmmss') + '-' + ([Guid]::NewGuid().ToString('N').Substring(0, 12))
[IO.Directory]::CreateDirectory($packageOutputRoot) | Out-Null
$packageRelease = Join-Path $packageOutputRoot $packageName
# New-Item without -Force deliberately fails on an unlikely name collision.
New-Item -ItemType Directory -Path $packageRelease -ErrorAction Stop | Out-Null
$packagePayload = Join-Path $packageRelease 'payload'
$packageBackend = Join-Path $packagePayload 'backend'
$packageFrontend = Join-Path $packagePayload 'frontend/dist'
$packageRuntime = Join-Path $packageBackend 'bin/runtime'
[IO.Directory]::CreateDirectory($packageRuntime) | Out-Null
[IO.Directory]::CreateDirectory($packageFrontend) | Out-Null

$packageBuildSnapshot = @{}
foreach ($packageVariable in @('GOOS', 'GOARCH', 'CGO_ENABLED', 'VITE_PUBLIC_DEMO', 'VITE_GLOB_API_URL', 'VITE_BASE')) {
    $packageBuildSnapshot[$packageVariable] = [Environment]::GetEnvironmentVariable($packageVariable, [EnvironmentVariableTarget]::Process)
}
try {
    Invoke-DevWithoutModelKeys -Action {
        [Environment]::SetEnvironmentVariable('GOOS', 'linux', [EnvironmentVariableTarget]::Process)
        [Environment]::SetEnvironmentVariable('GOARCH', 'amd64', [EnvironmentVariableTarget]::Process)
        [Environment]::SetEnvironmentVariable('CGO_ENABLED', '0', [EnvironmentVariableTarget]::Process)
        Push-Location $packageBackendRoot
        try {
            foreach ($packageBuild in @(
                @{ Name = 'applet-api'; Entry = './application/applet/api' },
                @{ Name = 'applet-rpc'; Entry = './application/applet/rpc' },
                @{ Name = 'applet-ai-rpc'; Entry = './application/ai/rpc' }
            )) {
                Write-Host ('Building Linux amd64: ' + $packageBuild.Name)
                & go build -trimpath '-ldflags=-s -w' -o (Join-Path $packageRuntime $packageBuild.Name) $packageBuild.Entry
                if ($LASTEXITCODE -ne 0) { throw ('Backend build failed: ' + $packageBuild.Name) }
            }
        } finally { Pop-Location }

        [Environment]::SetEnvironmentVariable('VITE_PUBLIC_DEMO', 'true', [EnvironmentVariableTarget]::Process)
        [Environment]::SetEnvironmentVariable('VITE_GLOB_API_URL', '/api', [EnvironmentVariableTarget]::Process)
        [Environment]::SetEnvironmentVariable('VITE_BASE', '/', [EnvironmentVariableTarget]::Process)
        Push-Location $packageFrontendRoot
        try {
            Write-Host 'Building the public demo frontend.'
            # Turbo strict mode otherwise drops unlisted VITE_* process values.
            # Force a fresh build instead of reusing an ordinary local build.
            & pnpm run build:antdv-next --force --env-mode=loose
            if ($LASTEXITCODE -ne 0) { throw 'Public demo frontend build failed.' }
        } finally { Pop-Location }
    }
} finally { Restore-DevEnvironment -Snapshot $packageBuildSnapshot }

$packageFrontendDist = Join-Path $packageFrontendRoot 'apps/web-antdv-next/dist'
if (-not (Test-Path -LiteralPath (Join-Path $packageFrontendDist 'index.html') -PathType Leaf)) {
    throw 'The frontend build did not produce dist/index.html.'
}
Copy-DemoPackageTree -Source $packageFrontendDist -Destination $packageFrontend
Copy-DemoPackageTree -Source (Join-Path $packageBackendRoot 'docker') -Destination (Join-Path $packageBackend 'docker') -Include {
    param($entry)
    return $entry.Extension -match '^\.(yml|yaml|template|sh|py|md|js)$' -or $entry.Name -in @('Dockerfile', 'Dockerfile.prebuilt', 'Caddyfile.demo')
}
foreach ($packageRuntimeFile in @('entrypoint.sh', 'api.yaml.template', 'rpc.yaml.template', 'ai-rpc.yaml.template')) {
    Copy-DemoPackageFile -Source (Join-Path $packageBackendRoot ('docker/' + $packageRuntimeFile)) -Destination (Join-Path $packageRuntime $packageRuntimeFile)
}
Copy-DemoPackageFile -Source (Join-Path $packageBackendRoot 'docker/Dockerfile.prebuilt') -Destination (Join-Path $packageRuntime 'Dockerfile')
foreach ($packageSource in @('test/sh/db.sh', 'test/sh/mysql-client.sh', 'data/db/gozero-admin.sql', '.dockerignore', 'README.md', 'CLOUD_DEPLOYMENT_NOTES.md', 'QUICK_RECOVERY.md')) {
    Copy-DemoPackageFile -Source (Join-Path $packageBackendRoot $packageSource) -Destination (Join-Path $packageBackend $packageSource)
}
Copy-DemoPackageTree -Source (Join-Path $packageBackendRoot 'data/db/migrations') -Destination (Join-Path $packageBackend 'data/db/migrations') -Include {
    param($entry)
    return $entry.Extension -eq '.sql'
}
Copy-DemoPackageTree -Source (Join-Path $packageBackendRoot 'data/db/operator') -Destination (Join-Path $packageBackend 'data/db/operator') -Include {
    param($entry)
    return $entry.Extension -eq '.sql'
}

$packageArchive = Join-Path $packageRelease 'go-zero-admin-demo-linux-amd64.tar.gz'
& tar -czf $packageArchive -C $packagePayload backend frontend
if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $packageArchive -PathType Leaf)) {
    throw 'Creating the deployment archive failed; the partial release directory was retained for inspection.'
}
$packageArchiveEntries = @(& tar -tzf $packageArchive)
if ($LASTEXITCODE -ne 0 -or $packageArchiveEntries.Count -eq 0) { throw 'The deployment archive could not be inspected.' }
foreach ($packageArchiveEntry in $packageArchiveEntries) {
    if ($packageArchiveEntry -notmatch '^(backend|frontend)/' -or
        $packageArchiveEntry -match '(^|/)(\.\.|\.env[^/]*|\.git|node_modules)(/|$)' -or
        $packageArchiveEntry -match '(^|/)dump-[^/]*\.sql$|\.(log|bak)$') {
        throw 'The deployment archive contained an unexpected path. Do not upload this release.'
    }
}
$packageHash = (Get-FileHash -LiteralPath $packageArchive -Algorithm SHA256).Hash.ToLowerInvariant()
$packageFiles = @(Get-ChildItem -LiteralPath $packagePayload -File -Recurse)
$packageSummary = [ordered]@{
    release = $packageName
    target = 'linux/amd64'
    publicDemo = $true
    modelKeysIncluded = $false
    files = $packageFiles.Count
    payloadBytes = [long](($packageFiles | Measure-Object -Property Length -Sum).Sum)
    archive = [IO.Path]::GetFileName($packageArchive)
    archiveBytes = (Get-Item -LiteralPath $packageArchive).Length
    sha256 = $packageHash
}
$packageSummary | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $packageRelease 'package-summary.json') -Encoding UTF8
Write-Host ('Payload: ' + $packagePayload)
Write-Host ('Archive: ' + $packageArchive)
Write-Host ('SHA256: ' + $packageHash)
Write-Host ('Files: ' + $packageFiles.Count + '; bytes: ' + $packageSummary.payloadBytes)
Write-Host 'Package complete. Actual environment files, model keys, source caches and database backups were excluded.'
