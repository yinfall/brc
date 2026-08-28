$ErrorActionPreference = 'Stop'

# Ensure TLS 1.2 / TLS 1.3 is enabled for older PowerShell 5.1
try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 -bor [Net.SecurityProtocolType]::Tls13
} catch {}

$Repo = "yinfall/blender-remote-console"
$BrcHome = "$HOME\.brc"
$InstallDir = "$BrcHome\bin"
$ReleaseApi = "https://api.github.com/repos/$Repo/releases/latest"

Write-Host ">>> Checking latest release info for $Repo..." -ForegroundColor Cyan
$Tag = $null
try {
    $ReleaseInfo = Invoke-RestMethod -Uri $ReleaseApi -Headers @{"User-Agent" = "PowerShell"} -TimeoutSec 10
    $Tag = $ReleaseInfo.tag_name
    Write-Host "    Found version: $Tag" -ForegroundColor Green
} catch {
    Write-Host "    (Unable to reach GitHub API, falling back to direct asset downloads)" -ForegroundColor Yellow
}

if ($Tag) {
    $ExeUrl = "https://github.com/$Repo/releases/download/$Tag/brc-windows-amd64.exe"
    $ZipUrl = "https://github.com/$Repo/releases/download/$Tag/blender-remote-console.zip"
} else {
    $ExeUrl = "https://github.com/$Repo/releases/latest/download/brc-windows-amd64.exe"
    $ZipUrl = "https://github.com/$Repo/releases/latest/download/blender-remote-console.zip"
}

# 1. Ensure directories exist
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$ExeTarget = Join-Path $InstallDir "brc.exe"
$ZipTarget = Join-Path $BrcHome "blender-remote-console.zip"

# 2. Download files
Write-Host ">>> Downloading brc.exe..." -ForegroundColor Cyan
Invoke-WebRequest -Uri $ExeUrl -OutFile $ExeTarget -UseBasicParsing

Write-Host ">>> Downloading Blender addon package..." -ForegroundColor Cyan
Invoke-WebRequest -Uri $ZipUrl -OutFile $ZipTarget -UseBasicParsing

# 3. Append ~/.brc/bin to User PATH if not present
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Host ">>> Adding $InstallDir to User PATH..." -ForegroundColor Cyan
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:PATH = "$env:PATH;$InstallDir"
}

# 4. Run brc install-addon to detect and install Blender Addon
Write-Host ">>> Configuring Blender addon..." -ForegroundColor Cyan
try {
    & "$ExeTarget" install-addon --all
} catch {
    Write-Host "    [Warning] Could not run 'brc install-addon' automatically. Run 'brc install-addon' manually after installation." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "✓ Installation completed successfully!" -ForegroundColor Green
Write-Host "1. Restart your terminal to use 'brc'"
Write-Host "2. Open Blender -> Edit -> Preferences -> Add-ons, and enable 'Blender Remote Console'"
