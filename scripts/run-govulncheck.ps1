param()

$ErrorActionPreference = "Stop"
. "$PSScriptRoot\\common.ps1"

$artifactDir = Join-Path $PSScriptRoot "..\\.artifacts"
$binaryPath = Join-Path $artifactDir "pet-study.exe"
$previousToolchain = $env:GOTOOLCHAIN

New-Item -ItemType Directory -Force -Path $artifactDir | Out-Null

try {
    $env:GOTOOLCHAIN = "go1.25.8+auto"
    Invoke-Native "go" @("build", "-o", $binaryPath, ".\\cmd\\api")
    Invoke-Native "govulncheck" @("-mode", "binary", $binaryPath)
}
finally {
    $env:GOTOOLCHAIN = $previousToolchain
    if (Test-Path $binaryPath) {
        Remove-Item $binaryPath
    }
}
