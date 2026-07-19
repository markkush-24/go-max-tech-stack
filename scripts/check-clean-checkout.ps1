param()

$ErrorActionPreference = "Stop"
. "$PSScriptRoot\\common.ps1"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$binaryPath = Join-Path ([System.IO.Path]::GetTempPath()) "pet-study-checkout-gate"
if ($IsWindows) {
    $binaryPath = "$binaryPath.exe"
}

Push-Location $repoRoot
try {
    Invoke-Native "go" @("mod", "tidy")
    Invoke-Native "go" @("generate", "./...")
    Invoke-Native "gofmt" @("-w", "cmd", "internal", "scripts")
    Invoke-Native "go" @("build", "-o", $binaryPath, "./cmd/api")
    Invoke-Native "go" @("test", "./...")
    Invoke-Native "git" @("diff", "--exit-code")

    $status = git status --porcelain
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: git status --porcelain"
    }
    if ($status) {
        Write-Error "Working tree is not clean after checkout gate:`n$status"
    }
}
finally {
    if (Test-Path $binaryPath) {
        Remove-Item $binaryPath
    }
    Pop-Location
}
