param()

$ErrorActionPreference = "Stop"
. "$PSScriptRoot\\common.ps1"

$toolsDir = Join-Path $PSScriptRoot "..\\tools"

Push-Location $toolsDir
try {
    $govulnModuleVersion = go list -m -f "{{.Version}}" golang.org/x/vuln
    if (-not $govulnModuleVersion) {
        throw "Failed to resolve golang.org/x/vuln version from tools/go.mod"
    }

    $staticcheckModuleVersion = go list -m -f "{{.Version}}" honnef.co/go/tools
    if (-not $staticcheckModuleVersion) {
        throw "Failed to resolve honnef.co/go/tools version from tools/go.mod"
    }

    Invoke-Native "go" @("install", "golang.org/x/vuln/cmd/govulncheck@$govulnModuleVersion")
    Invoke-Native "go" @("install", "honnef.co/go/tools/cmd/staticcheck@$staticcheckModuleVersion")
}
finally {
    Pop-Location
}
