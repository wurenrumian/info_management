param(
    [string]$UploadDir = "",
    [string]$UploadUid = "",
    [string]$UploadGid = "",
    [string]$UploadMode = "",
    [string]$UploadSubdirs = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$rootDir = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path

if ([string]::IsNullOrWhiteSpace($UploadDir)) {
    $UploadDir = $env:UPLOAD_DIR
}
if ([string]::IsNullOrWhiteSpace($UploadDir)) {
    $UploadDir = Join-Path $rootDir "data/uploads"
}

if ([string]::IsNullOrWhiteSpace($UploadUid)) {
    $UploadUid = $env:UPLOAD_UID
}
if ([string]::IsNullOrWhiteSpace($UploadUid)) {
    $UploadUid = "10001"
}

if ([string]::IsNullOrWhiteSpace($UploadGid)) {
    $UploadGid = $env:UPLOAD_GID
}
if ([string]::IsNullOrWhiteSpace($UploadGid)) {
    $UploadGid = "10001"
}

if ([string]::IsNullOrWhiteSpace($UploadMode)) {
    $UploadMode = $env:UPLOAD_MODE
}
if ([string]::IsNullOrWhiteSpace($UploadMode)) {
    $UploadMode = "0775"
}

if ([string]::IsNullOrWhiteSpace($UploadSubdirs)) {
    $UploadSubdirs = $env:UPLOAD_SUBDIRS
}
if ([string]::IsNullOrWhiteSpace($UploadSubdirs)) {
    $UploadSubdirs = "images avatars knowledge"
}

$subdirList = @(
    $UploadSubdirs -split "\s+" |
        Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
)

function Ensure-Directory {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Grant-DirectoryAccess {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    $currentUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
    $grantTargets = @($currentUser, "Users")

    foreach ($target in $grantTargets) {
        & icacls $Path /grant "${target}:(OI)(CI)M" /T /C | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "icacls grant failed for $target"
        }
    }

    & icacls $Path /inheritance:e /T /C | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "icacls inheritance enable failed"
    }

    & cmd /c attrib -R "$Path\*" /S /D 2>$null | Out-Null
}

Write-Host "Preparing upload directory..."
Write-Host "  UPLOAD_DIR=$UploadDir"
Write-Host "  UPLOAD_UID=$UploadUid"
Write-Host "  UPLOAD_GID=$UploadGid"
Write-Host "  UPLOAD_MODE=$UploadMode"
Write-Host "  UPLOAD_SUBDIRS=$UploadSubdirs"

try {
    Ensure-Directory -Path $UploadDir
} catch {
    Write-Host "[ERROR] failed to create upload dir: $UploadDir"
    Write-Host $_.Exception.Message
    exit 1
}

foreach ($subdir in $subdirList) {
    $subdirPath = Join-Path $UploadDir $subdir
    try {
        Ensure-Directory -Path $subdirPath
    } catch {
        Write-Host "[ERROR] failed to create upload subdir: $subdirPath"
        Write-Host $_.Exception.Message
        exit 1
    }
}

try {
    Grant-DirectoryAccess -Path $UploadDir
} catch {
    Write-Host "[WARN] failed to adjust ACL for $UploadDir."
    Write-Host "[WARN] On Windows, UID/GID and chmod mode cannot be applied directly."
    Write-Host "[WARN] Please make sure the current user and Docker Desktop have write access."
}

Write-Host ""
Write-Host "Prepared successfully:"
Get-Item -LiteralPath $UploadDir | Format-List FullName, Attributes, Mode
