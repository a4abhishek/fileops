# FileOps Installation Script for Windows
# Run this script in PowerShell as Administrator

param(
    [string]$InstallDir = "$env:ProgramFiles\FileOps",
    [switch]$AddToPath = $true
)

# Colors for output
function Write-Status {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# Detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default {
            Write-Error "Unsupported architecture: $arch"
            exit 1
        }
    }
}

# Get latest release version
function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/a4abhishek/fileops/releases/latest"
        return $response.tag_name
    }
    catch {
        Write-Error "Failed to get latest version: $_"
        exit 1
    }
}

# Download and install FileOps
function Install-FileOps {
    Write-Status "FileOps Installation Script"
    Write-Status "============================"
    
    # Check if running as administrator
    $isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    if (-not $isAdmin) {
        Write-Warning "Running without administrator privileges. Installation may fail."
    }
    
    # Detect architecture
    Write-Status "Detecting architecture..."
    $arch = Get-Architecture
    Write-Status "Detected architecture: $arch"
    
    # Get latest version
    Write-Status "Getting latest version..."
    $version = Get-LatestVersion
    Write-Status "Latest version: $version"
    
    # Create temporary directory
    $tempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
    
    try {
        # Download binary
        $downloadUrl = "https://github.com/a4abhishek/fileops/releases/download/$version/fileops_${version}_windows_${arch}.zip"
        $zipPath = Join-Path $tempDir "fileops.zip"
        
        Write-Status "Downloading from: $downloadUrl"
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath
        
        # Extract binary
        Write-Status "Extracting binary..."
        Expand-Archive -Path $zipPath -DestinationPath $tempDir -Force
        
        # Create installation directory
        Write-Status "Creating installation directory: $InstallDir"
        if (-not (Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }
        
        # Copy binary
        $binaryPath = Join-Path $tempDir "fileops.exe"
        $targetPath = Join-Path $InstallDir "fileops.exe"
        
        Write-Status "Installing to: $targetPath"
        Copy-Item $binaryPath $targetPath -Force
        
        # Add to PATH
        if ($AddToPath) {
            Write-Status "Adding to PATH..."
            $currentPath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
            if ($currentPath -notlike "*$InstallDir*") {
                $newPath = "$currentPath;$InstallDir"
                [Environment]::SetEnvironmentVariable("PATH", $newPath, "Machine")
                Write-Status "Added $InstallDir to system PATH"
                Write-Status "Please restart your terminal to use 'fileops' command"
            } else {
                Write-Status "$InstallDir is already in PATH"
            }
        }
        
        Write-Status ""
        Write-Status "🎉 Installation complete!"
        Write-Status ""
        Write-Status "Next steps:"
        Write-Status "  1. Restart your terminal if PATH was modified"
        Write-Status "  2. Run 'fileops version' to verify installation"
        Write-Status "  3. Run 'fileops --help' to see available commands"
        Write-Status "  4. Check out examples at: https://github.com/a4abhishek/fileops/wiki"
        
    }
    finally {
        # Cleanup
        Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# Run installation
Install-FileOps
