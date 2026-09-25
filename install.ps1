# GoClean Windows installer for PowerShell
# Usage: irm https://raw.githubusercontent.com/jalioba/GoClean/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$repo = "jalioba/GoClean"
$binaryName = "goclean.exe"

Write-Host "🔍 Detecting platform architecture..." -ForegroundColor Cyan
$arch = if ([System.Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else {
    "386"
}

Write-Host "📡 Querying latest release from GitHub..." -ForegroundColor Cyan
$releaseUrl = "https://api.github.com/repos/$repo/releases/latest"

try {
    $latestRelease = Invoke-RestMethod -Uri $releaseUrl -Headers @{ "User-Agent" = "GoClean-Installer" }
    $tag = $latestRelease.tag_name
} catch {
    Write-Warning "Could not fetch latest release via API, defaulting to v1.0.0"
    $tag = "v1.0.0"
}

$version = $tag.TrimStart("v")
$zipFileName = "goclean_${version}_windows_${arch}.zip"
$downloadUrl = "https://github.com/$repo/releases/download/$tag/$zipFileName"

$destDir = Join-Path $env:LOCALAPPDATA "Programs\goclean"
if (-not (Test-Path $destDir)) {
    New-Item -ItemType Directory -Force -Path $destDir | Out-Null
}

$tempZip = Join-Path $env:TEMP $zipFileName

Write-Host "⬇️  Downloading GoClean ($tag)..." -ForegroundColor Cyan
try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $tempZip -UseBasicParsing
} catch {
    Write-Error "Failed to download $downloadUrl. Please verify the release asset exists."
    exit 1
}

Write-Host "📦 Extracting files to $destDir..." -ForegroundColor Cyan
Expand-Archive -Path $tempZip -DestinationPath $destDir -Force
Remove-Item -Path $tempZip -Force

# Ensure directory is in User PATH
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$destDir*") {
    Write-Host "⚙️  Adding $destDir to User PATH..." -ForegroundColor Cyan
    $newPath = "$userPath;$destDir"
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    $env:Path = "$env:Path;$destDir"
}

Write-Host ""
Write-Host "✅ GoClean $tag installed successfully!" -ForegroundColor Green
Write-Host "🚀 Run 'goclean --help' or 'goclean -i' to get started." -ForegroundColor Green
