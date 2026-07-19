param()

$ErrorActionPreference = "Stop"
. "$PSScriptRoot\\common.ps1"

$cgoEnabled = (go env CGO_ENABLED).Trim()
if ($cgoEnabled -ne "1") {
    Write-Host "Skipping race tests locally because CGO_ENABLED=$cgoEnabled. CI still runs go test -race ./...."
    exit 0
}

Invoke-Native "go" @("test", "-race", "./...")
