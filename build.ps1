# Builds EasyMoveResize.exe (no console window, stripped).
param([string]$Version = 'dev')
$ErrorActionPreference = 'Stop'
Push-Location $PSScriptRoot
try {
    go build -trimpath -ldflags "-H windowsgui -s -w -X main.version=$Version" -o EasyMoveResize.exe .
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    Get-Item EasyMoveResize.exe | Select-Object Name, Length
} finally {
    Pop-Location
}
