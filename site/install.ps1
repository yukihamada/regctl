# regctl installer for Windows

$ErrorActionPreference = "Stop"

$Version = if ($env:VERSION) { $env:VERSION } else { "latest" }
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { "$env:LOCALAPPDATA\Programs\regctl" }

Write-Host "Installing regctl..." -ForegroundColor Green
Write-Host "Version: $Version"

# Create install directory
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# Construct download URL
if ($Version -eq "latest") {
    $DownloadUrl = "https://github.com/yukihamada/regctl/releases/latest/download/regctl-windows-amd64.exe"
} else {
    $DownloadUrl = "https://github.com/yukihamada/regctl/releases/download/$Version/regctl-windows-amd64.exe"
}

$OutputFile = Join-Path $InstallDir "regctl.exe"

Write-Host "Downloading from: $DownloadUrl"

# Download file
try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $OutputFile -UseBasicParsing
} catch {
    Write-Host "Error downloading regctl: $_" -ForegroundColor Red
    exit 1
}

# Add to PATH if not already present
$Path = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
if ($Path -notlike "*$InstallDir*") {
    Write-Host "Adding $InstallDir to PATH..."
    [Environment]::SetEnvironmentVariable(
        "Path",
        "$Path;$InstallDir",
        [EnvironmentVariableTarget]::User
    )
    $env:Path = "$env:Path;$InstallDir"
}

# Verify installation
Write-Host ""
& $OutputFile version

if ($?) {
    Write-Host ""
    Write-Host "✅ regctl installed successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "You may need to restart your terminal for PATH changes to take effect."
    Write-Host ""
    Write-Host "Get started with: regctl login"
} else {
    Write-Host "⚠️  Installation completed but verification failed" -ForegroundColor Yellow
    Write-Host "Try running: $OutputFile"
}