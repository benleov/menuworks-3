param(
    [string[]]$Packages = @('./config', './menu')
)

$ErrorActionPreference = 'Stop'

$go = Join-Path $PSScriptRoot 'bin\go\bin\go.exe'
if (-not (Test-Path $go)) {
    $go = Join-Path $PSScriptRoot 'bin\go\bin\go'
}
if (-not (Test-Path $go)) {
    Write-Error "Go binary not found at $go"
    exit 1
}

$failed = $false
foreach ($pkg in $Packages) {
    Write-Host "Running tests for $pkg"
    & $go test $pkg
    if ($LASTEXITCODE -ne 0) {
        $failed = $true
    }
}

if ($failed) {
    exit 1
}
