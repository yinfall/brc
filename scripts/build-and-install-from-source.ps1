$ErrorActionPreference = 'Stop'

# Project root directory
$ProjectRoot = (Get-Item $PSScriptRoot).Parent.FullName
$BrcHome = "$HOME\.brc"
$InstallDir = "$BrcHome\bin"

Write-Host "=== Installing Blender Remote Console locally from source ===" -ForegroundColor Cyan
Write-Host "Project Root: $ProjectRoot"

# 1. Ensure ~/.brc/bin directory exists
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# 2. Build Go CLI directly to ~/.brc/bin/brc.exe
Write-Host "`n>>> [1/3] Building brc.exe locally from source..." -ForegroundColor Cyan
$CliDir = Join-Path $ProjectRoot "cli"
$ExeTarget = Join-Path $InstallDir "brc.exe"

Push-Location $CliDir
try {
    go build -ldflags="-s -w" -o $ExeTarget .
    Write-Host "✓ Successfully built: $ExeTarget" -ForegroundColor Green
} finally {
    Pop-Location
}

# 3. Package local blender-addon directory to ~/.brc/blender-remote-console.zip
Write-Host "`n>>> [2/3] Packaging local addon files into zip..." -ForegroundColor Cyan
$AddonDir = Join-Path $ProjectRoot "blender-addon"
$ZipTarget = Join-Path $BrcHome "blender-remote-console.zip"

if (Test-Path $ZipTarget) {
    Remove-Item -Force $ZipTarget
}
Compress-Archive -Path "$AddonDir\*" -DestinationPath $ZipTarget -Force
Write-Host "✓ Successfully created: $ZipTarget" -ForegroundColor Green

# 4. Add ~/.brc/bin to User PATH if not present
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Host "`n>>> Adding $InstallDir to User PATH..." -ForegroundColor Cyan
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:PATH = "$env:PATH;$InstallDir"
}

# 5. Run brc install-addon
Write-Host "`n>>> [3/3] Running 'brc install-addon'..." -ForegroundColor Cyan
& "$ExeTarget" install-addon

Write-Host "`n✓ Local installation & setup completed!" -ForegroundColor Green
