param()

$ErrorActionPreference = "Stop"

& "$PSScriptRoot\\check-format.ps1"
. "$PSScriptRoot\\common.ps1"
Invoke-Native "go" @("test", "./...")
Invoke-Native "go" @("vet", "./...")
& "$PSScriptRoot\\run-race.ps1"
& "$PSScriptRoot\\run-staticcheck.ps1"
& "$PSScriptRoot\\run-govulncheck.ps1"
