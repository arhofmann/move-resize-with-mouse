# Builds MoveResizeWithMouse.exe (no console window, stripped).
param([string]$Version = 'dev')
$ErrorActionPreference = 'Stop'
Push-Location $PSScriptRoot
try {
    go build -trimpath -ldflags "-H windowsgui -s -w -X main.version=$Version" -o MoveResizeWithMouse.exe .
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    Get-Item MoveResizeWithMouse.exe | Select-Object Name, Length
} finally {
    Pop-Location
}
