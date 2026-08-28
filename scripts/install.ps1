$ErrorActionPreference = 'Stop'

# Set your GitHub repository name here
$Repo = "your-username/blender-remote-console"
$InstallDir = "$HOME\.brc\bin"
$ReleaseApi = "https://api.github.com/repos/$Repo/releases/latest"

Write-Host ">>> Fetching latest release info for $Repo..." -ForegroundColor Cyan
try {
    $ReleaseInfo = Invoke-RestMethod -Uri $ReleaseApi -Headers @{"User-Agent" = "PowerShell"}
    $Tag = $ReleaseInfo.tag_name
} catch {
    Write-Host "Failed to fetch latest release. Please ensure releases are published on GitHub." -ForegroundColor Red
    exit 1
}

# 1. Download and configure brc.exe
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$ExeUrl = "https://github.com/$Repo/releases/download/$Tag/brc-windows-amd64.exe"
$ExeTarget = Join-Path $InstallDir "brc.exe"

Write-Host ">>> Downloading brc.exe ($Tag)..." -ForegroundColor Cyan
Invoke-WebRequest -Uri $ExeUrl -OutFile $ExeTarget

# Append ~/.brc/bin to User PATH if not present
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Host ">>> Adding $InstallDir to User PATH..." -ForegroundColor Cyan
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:PATH = "$env:PATH;$InstallDir"
}

# 2. Detect local Blender installations and install Addon
Write-Host ">>> Installing Blender Addon..." -ForegroundColor Cyan
$BlenderBase = "$env:APPDATA\Blender Foundation\Blender"
$AddonZipUrl = "https://github.com/$Repo/releases/download/$Tag/blender-remote-console.zip"
$TempZip = Join-Path $env:TEMP "blender-remote-console.zip"

Invoke-WebRequest -Uri $AddonZipUrl -OutFile $TempZip

if (Test-Path $BlenderBase) {
    $Versions = Get-ChildItem $BlenderBase -Directory | Where-Object { $_.Name -match '^\d+\.\d+' }
    if ($Versions.Count -eq 0) {
        Write-Host "    No specific version folders found in $BlenderBase." -ForegroundColor Yellow
    }
    foreach ($Ver in $Versions) {
        $AddonPath = Join-Path $Ver.FullName "scripts\addons\blender-remote-console"
        if (-not (Test-Path $AddonPath)) {
            New-Item -ItemType Directory -Path $AddonPath -Force | Out-Null
        }
        Expand-Archive -Path $TempZip -DestinationPath $AddonPath -Force
        Write-Host "    Installed to Blender $($Ver.Name)" -ForegroundColor Green
    }
} else {
    Write-Host "    [Warning] Blender AppData directory not found. Please manually extract $TempZip to your Blender scripts/addons directory." -ForegroundColor Yellow
}
Remove-Item $TempZip -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "✓ Installation completed successfully!" -ForegroundColor Green
Write-Host "1. Restart your terminal to use 'brc'"
Write-Host "2. Open Blender -> Edit -> Preferences -> Add-ons, and enable 'Blender Remote Console'"
