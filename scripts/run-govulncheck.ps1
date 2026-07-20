param()

$ErrorActionPreference = "Stop"
. "$PSScriptRoot\\common.ps1"

$artifactDir = Join-Path $PSScriptRoot "..\\.artifacts"
$binaryName = "pet-study"
if ($IsWindows) {
    $binaryName = "pet-study.exe"
}
$binaryPath = Join-Path $artifactDir $binaryName

New-Item -ItemType Directory -Force -Path $artifactDir | Out-Null

try {
    Invoke-Native "go" @("version")
    Invoke-Native "go" @("build", "-o", $binaryPath, "./cmd/api")
    Invoke-Native "govulncheck" @("-mode", "binary", $binaryPath)
}
finally {
    if (Test-Path $binaryPath) {
        Remove-Item $binaryPath
    }
}
