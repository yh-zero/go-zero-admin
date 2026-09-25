#requires -Version 5.1
[CmdletBinding()]
param(
    [ValidateSet('Init', 'Migrate', 'Backup', 'Status')][string]$Action = 'Status',
    [ValidateSet('Development', 'Deploy')][string]$Environment = 'Development',
    [ValidatePattern('^[A-Za-z0-9_-]*$')][string]$Database = ''
)
$ErrorActionPreference = 'Stop'
$projectRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
$utf8 = New-Object Text.UTF8Encoding($false)
$composeArgs = @('compose', '-f', (Join-Path $projectRoot 'docker-compose.yml'))
if ($Environment -eq 'Deploy') {
    $composeArgs = @('compose', '--env-file', (Join-Path $projectRoot 'docker/.env.deploy'), '-f', (Join-Path $projectRoot 'docker/deploy-compose.yml'))
}
function Invoke-Docker([string[]]$DockerArgs) {
    $output = & docker @DockerArgs
    if ($LASTEXITCODE -ne 0) { throw ('Docker command failed: ' + ($DockerArgs -join ' ')) }
    return $output
}
function Invoke-Compose([string[]]$CommandArgs) { Invoke-Docker ($composeArgs + $CommandArgs) }
function Invoke-Client([string[]]$ClientArgs) {
    $execArgs = @('exec')
    if ($Database) { $execArgs += @('-e', ('MYSQL_DATABASE=' + $Database)) }
    Invoke-Docker ($execArgs + @($container, 'sh', $helper) + $ClientArgs)
}
function Invoke-Query([string]$Sql) { Invoke-Client @('query', $Sql) }
function Get-MigrationHash([string]$Path) {
    $bytes = $utf8.GetBytes([IO.File]::ReadAllText($Path).Replace("`r`n", "`n"))
    $algorithm = [Security.Cryptography.SHA256]::Create()
    try { return ([BitConverter]::ToString($algorithm.ComputeHash($bytes))).Replace('-', '').ToLowerInvariant() }
    finally { $algorithm.Dispose() }
}
function Save-Backup {
    $backupDir = Join-Path $projectRoot 'bin/db-backups'
    New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
    $name = 'gozero-' + $Environment.ToLowerInvariant() + '-' + (Get-Date -Format 'yyyyMMdd-HHmmss-fff') + '.sql'
    $path = Join-Path $backupDir $name
    $temporary = '/tmp/' + $name
    if (Test-Path -LiteralPath $path) { throw 'Backup already exists.' }
    Invoke-Client @('backup', $temporary) | Out-Null
    try {
        Invoke-Docker @('cp', ($container + ':' + $temporary), ($path + '.partial')) | Out-Null
        $body = [IO.File]::ReadAllText($path + '.partial', $utf8)
        if ($body -notmatch '(?m)^-- Dump completed on ' -or $body -notmatch '(?m)^CREATE TABLE ') { throw 'Incomplete backup; partial file kept for inspection.' }
        $remoteHash = ((Invoke-Docker @('exec', $container, 'sha256sum', $temporary)) -split '\s+')[0]
        $hash = (Get-FileHash -LiteralPath ($path + '.partial') -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($hash -cne $remoteHash) { throw 'Backup checksum mismatch.' }
        Rename-Item -LiteralPath ($path + '.partial') -NewName $name
        [IO.File]::WriteAllText($path + '.sha256', "$hash  $name`n", $utf8)
        Write-Host "Backup saved: $path"
    } finally {
        Invoke-Docker @('exec', $container, 'rm', '-f', '--', $temporary) | Out-Null
    }
}
Push-Location $projectRoot
$locked = $false
$container = ''
$helper = '/tmp/gozero-db-client-' + [guid]::NewGuid().ToString('N') + '.sh'
$helperLocal = Join-Path ([IO.Path]::GetTempPath()) ([IO.Path]::GetFileName($helper))
try {
    if ($Action -eq 'Init') {
        $services = @('mysql', 'redis', 'etcd')
        if ($Environment -eq 'Development') { $services += 'swagger-ui' }
        Invoke-Compose (@('up', '-d', '--wait', '--wait-timeout', '180') + $services) | Out-Host
    }
    $container = (Invoke-Compose @('ps', '-q', 'mysql') | Out-String).Trim()
    if (-not $container) { throw 'Start the MySQL service first, or use -Action Init.' }
    # Normalize checkout CRLF before copying this POSIX script into the container.
    [IO.File]::WriteAllText($helperLocal, ([IO.File]::ReadAllText((Join-Path $PSScriptRoot 'mysql-client.sh')).Replace("`r`n", "`n")), $utf8)
    Invoke-Docker @('cp', $helperLocal, ($container + ':' + $helper)) | Out-Null
    Invoke-Query 'SELECT 1;' | Out-Null
    Write-Host ('Database: ' + ((Invoke-Query 'SELECT DATABASE();') | Out-String).Trim())
    if ($Action -eq 'Backup') { Save-Backup; return }
    $tableExists = (Invoke-Query "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='schema_migrations';" | Out-String).Trim() -eq '1'
    $applied = @{}
    if ($tableExists) {
        foreach ($line in @(Invoke-Query 'SELECT filename, checksum FROM schema_migrations ORDER BY filename;')) {
            $values = $line -split "`t"
            if ($values.Length -eq 2) { $applied[$values[0]] = $values[1] }
        }
    }
    $files = @(Get-ChildItem -LiteralPath (Join-Path $projectRoot 'data/db/migrations') -Filter '*.sql' | Sort-Object Name)
    $pending = @()
    foreach ($file in $files) {
        if ($file.Name -notmatch '^[A-Za-z0-9_-]+\.sql$') { throw 'Unsupported migration filename.' }
        $hash = Get-MigrationHash $file.FullName
        if ($applied.ContainsKey($file.Name)) {
            if ($applied[$file.Name] -cne $hash) { throw ('Applied migration was modified: ' + $file.Name + '. Add a new migration instead.') }
            Write-Host ('APPLIED ' + $file.Name)
        } else {
            Write-Host ('PENDING ' + $file.Name)
            $pending += @{ File = $file; Hash = $hash }
        }
    }
    if ($Action -eq 'Status' -or $pending.Count -eq 0) { return }
    # Shared by the PowerShell and shell entry points, including separate hosts.
    Invoke-Docker @('exec', $container, 'mkdir', '/tmp/gozero-db-migrate.lock') | Out-Null
    $locked = $true
    Save-Backup
    Invoke-Query 'CREATE TABLE IF NOT EXISTS schema_migrations (filename VARCHAR(191) NOT NULL PRIMARY KEY, checksum CHAR(64) NOT NULL, applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP) ENGINE=InnoDB;' | Out-Null
    foreach ($item in $pending) {
        $name = $item.File.Name
        # Recheck after locking so concurrent starts do not apply a migration twice.
        $existing = (Invoke-Query "SELECT checksum FROM schema_migrations WHERE filename='$name';" | Out-String).Trim()
        if ($existing) {
            if ($existing -cne $item.Hash) { throw ('Applied migration checksum mismatch: ' + $name) }
            continue
        }
        $remote = '/tmp/gozero-migration-' + $name
        Invoke-Docker @('cp', $item.File.FullName, ($container + ':' + $remote)) | Out-Null
        try {
            Invoke-Client @('run', $remote) | Out-Host
            Invoke-Query "INSERT INTO schema_migrations(filename,checksum) VALUES('$name','$($item.Hash)');" | Out-Null
            Write-Host ('APPLIED ' + $name)
        } finally {
            Invoke-Docker @('exec', $container, 'rm', '-f', '--', $remote) | Out-Null
        }
    }
    Write-Host 'Migration complete. Restart RPC/API and reload frontend permissions.'
} finally {
    if ($locked) { & docker exec $container rmdir /tmp/gozero-db-migrate.lock | Out-Null }
    if ($container) { & docker exec $container rm -f -- $helper | Out-Null }
    if (Test-Path -LiteralPath $helperLocal) { Remove-Item -LiteralPath $helperLocal -Force }
    Pop-Location
}
