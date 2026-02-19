# MenuWorks 3.0 Build Script (PowerShell)
# Builds cross-platform binaries using local Go installation

param(
    [string]$Target = "all"  # all, windows, linux, macos
)

# Read version from VERSION file
$Version = (Get-Content "VERSION" -Raw).Trim()
if (-not $Version) {
    Write-Error "VERSION file is empty"
    exit 1
}

# Set local Go path
$localGo = Join-Path (Get-Location) "bin\go\bin\go.exe"
if (-not (Test-Path $localGo)) {
    Write-Error "Go not found at $localGo. Please install Go in bin/go first."
    exit 1
}

$env:PATH = "$(Join-Path (Get-Location) 'bin\go\bin');$env:PATH"

Write-Host "MenuWorks 3.0 Build System" -ForegroundColor Cyan
Write-Host "Go: $localGo" -ForegroundColor Gray
Write-Host "Version: $Version" -ForegroundColor Gray
Write-Host ""

# Ensure dist directory exists
if (-not (Test-Path "dist")) {
    New-Item -ItemType Directory -Path "dist" | Out-Null
}

# Build matrix: @{ OutputName = "OS/ARCH" }
$targets = @{
    "menuworks-windows.exe"       = "windows/amd64"
    "menuworks-linux"             = "linux/amd64"
    "menuworks-macos"             = "darwin/amd64"
    "menuworks-macos-arm64"       = "darwin/arm64"
}

# Filter by target if specified
if ($Target -ne "all") {
    $filtered = @{}
    $targets.GetEnumerator() | ForEach-Object {
        if ($_.Name -match $Target) {
            $filtered[$_.Name] = $_.Value
        }
    }
    if ($filtered.Count -eq 0) {
        Write-Error "No targets matched: $Target"
        Write-Host "Available targets: all, windows, linux, macos" -ForegroundColor Yellow
        exit 1
    }
    $targets = $filtered
}

# Build each target
$successCount = 0
$targets.GetEnumerator() | ForEach-Object {
    $outputFile = $_.Name
    $osArch = $_.Value -split "/"
    $os, $arch = $osArch[0], $osArch[1]

    Write-Host "Building $outputFile ($os/$arch)..." -ForegroundColor Green
    
    $env:GOOS = $os
    $env:GOARCH = $arch
    
    $ldFlags = "-X main.version=$Version"
    $outputPath = "dist/$outputFile"
    
    & $localGo build -ldflags $ldFlags -o $outputPath cmd/menuworks/main.go
    
    if ($LASTEXITCODE -eq 0) {
        $size = (Get-Item $outputPath).Length / 1MB
        Write-Host "  ✓ $outputFile ($([math]::Round($size, 2)) MB)" -ForegroundColor Green
        $successCount++
    } else {
        Write-Host "  ✗ Failed to build $outputFile" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "Build complete: $successCount/$($targets.Count) targets succeeded" -ForegroundColor Cyan

# Clean environment
$env:GOOS = ""
$env:GOARCH = ""

if ($successCount -eq $targets.Count) {
    Write-Host "All builds successful!" -ForegroundColor Green
    exit 0
} else {
    exit 1
}