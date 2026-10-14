# TestGuard Windows PowerShell Installer
$ErrorActionPreference = 'Stop'

$Repo = "HoangNguyen0403/testguard"
$Project = "testguard"

Write-Host "🛡️ Installing TestGuard for Windows..." -ForegroundColor Cyan

$Arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else {
    Write-Error "TestGuard currently requires a 64-bit Windows operating system."
    exit 1
}

$ZipName = "$Project-windows-$Arch.zip"
$DownloadUrl = "https://github.com/$Repo/releases/latest/download/$ZipName"

$InstallDir = "$env:LOCALAPPDATA\Programs\$Project"
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$TempZip = Join-Path $env:TEMP $ZipName

Write-Host "⬇️ Downloading $ZipName from GitHub Releases..." -ForegroundColor Gray
Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempZip

Write-Host "📦 Extracting binary..." -ForegroundColor Gray
Expand-Archive -Path $TempZip -DestinationPath $InstallDir -Force
Remove-Item $TempZip -Force

# Ensure PATH contains install directory
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
    Write-Host "✅ Added $InstallDir to User PATH." -ForegroundColor Green
}

Write-Host "✅ TestGuard successfully installed to $InstallDir\testguard.exe!" -ForegroundColor Green
Write-Host "👉 Open a new PowerShell terminal and run: testguard --version" -ForegroundColor Cyan
