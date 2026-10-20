#requires -Version 5.1
<#
.SYNOPSIS
    Regenerates data/db/gozero-admin.sql from baseline.sql plus the ordered tail
    migration/operator files listed in $TailEntries.
.DESCRIPTION
    Single source of truth: baseline.sql and the files under data/db/migrations
    and data/db/operator. Never edit gozero-admin.sql by hand. To include a new
    migration in fresh installations, append its entries to $TailEntries and run
    this script. schema_migrations checksums (SHA-256 over UTF-8 without BOM,
    CRLF normalized to LF) are computed automatically. -Check compares the
    committed gozero-admin.sql with a fresh build and fails on any difference
    without rewriting the file.
#>
[CmdletBinding()]
param([switch]$Check)

$ErrorActionPreference = 'Stop'
$dbRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\db'))
$target = Join-Path $dbRoot 'gozero-admin.sql'
$utf8 = New-Object Text.UTF8Encoding($false, $true)

# Ordered tail appended after baseline.sql. A migration's schema_migrations
# record is deferred until after any following operator entries, matching the
# documented upgrade order (migration, then explicit operator bootstrap).
$TailEntries = @(
    @{ Path = 'migrations/20261008_permission_revision_history.sql' }
    @{ Path = 'operator/20261008_permission_admin_recovery_bootstrap.sql' }
    @{ Path = 'migrations/20261019_app_members.sql'; Marker = '-- App module: fresh installation includes administrator role 1 bootstrap.' }
    @{ Path = 'operator/20261019_app_member_admin_bootstrap.sql' }
    @{ Path = 'migrations/20261019_app_registration_ip.sql' }
)

function Read-SqlText([string]$relativePath) {
    $text = $utf8.GetString([IO.File]::ReadAllBytes((Join-Path $dbRoot $relativePath)))
    return $text.TrimStart([char]0xFEFF).Replace("`r`n", "`n")
}

function Get-SqlChecksum([string]$text) {
    $sha = [Security.Cryptography.SHA256]::Create()
    try {
        return ([BitConverter]::ToString($sha.ComputeHash($utf8.GetBytes($text)))).Replace('-', '').ToLowerInvariant()
    } finally {
        $sha.Dispose()
    }
}

function Add-MigrationRecord([Text.StringBuilder]$builder, [string]$filename, [string]$suffix) {
    $checksum = Get-SqlChecksum (Read-SqlText ('migrations/' + $filename))
    [void]$builder.Append("INSERT INTO schema_migrations(filename,checksum,applied_at) VALUES('$filename','$checksum',NOW());" + $suffix)
}

$builder = New-Object Text.StringBuilder
[void]$builder.Append((Read-SqlText 'baseline.sql'))
$pendingMigration = $null
foreach ($entry in $TailEntries) {
    $isMigration = $entry.Path.StartsWith('migrations/')
    if ($isMigration -and $null -ne $pendingMigration) {
        Add-MigrationRecord $builder $pendingMigration "`n`n"
    }
    if ($isMigration -and $entry.ContainsKey('Marker')) {
        [void]$builder.Append($entry.Marker + "`n")
    }
    [void]$builder.Append((Read-SqlText $entry.Path) + "`n")
    if ($isMigration) { $pendingMigration = [IO.Path]::GetFileName($entry.Path) }
}
if ($null -ne $pendingMigration) {
    Add-MigrationRecord $builder $pendingMigration "`n"
}
$built = $builder.ToString()

if ($Check) {
    $current = $utf8.GetString([IO.File]::ReadAllBytes($target)).TrimStart([char]0xFEFF).Replace("`r`n", "`n")
    if ([string]::Equals($current, $built, [StringComparison]::Ordinal)) {
        Write-Host 'data/db/gozero-admin.sql is up to date.'
        exit 0
    }
    Write-Host 'data/db/gozero-admin.sql is stale. Run data/scripts/build-fresh-sql.ps1 and commit the result.'
    exit 1
}

[IO.File]::WriteAllBytes($target, $utf8.GetBytes($built.Replace("`n", "`r`n")))
Write-Host ('Regenerated ' + $target)
