param()

$ErrorActionPreference = "Stop"
. "$PSScriptRoot\\common.ps1"

$toolsDir = Join-Path $PSScriptRoot "..\\tools"

function ConvertTo-NormalizedVersionToken {
    param([Parameter(Mandatory = $true)][string]$Token)

    return ($Token.Trim() -replace '^[\s"''`()\[\]{}<>,;]+|[\s"''`()\[\]{}<>,;]+$', '')
}

function Test-VersionOutputToken {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Output,

        [Parameter(Mandatory = $true)]
        [string]$Expected
    )

    $expectedToken = ConvertTo-NormalizedVersionToken $Expected
    foreach ($token in ($Output -split '\s+')) {
        if ((ConvertTo-NormalizedVersionToken $token) -eq $expectedToken) {
            return $true
        }
    }

    return $false
}

function Test-InstalledVersion {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Command,

        [string[]]$Arguments = @(),

        [Parameter(Mandatory = $true)]
        [string]$Expected
    )

    if (-not (Get-Command $Command -ErrorAction SilentlyContinue)) {
        return $false
    }

    $output = & $Command @Arguments 2>&1 | Out-String
    if ($LASTEXITCODE -ne 0) {
        return $false
    }

    return Test-VersionOutputToken -Output $output -Expected $Expected
}

function Install-GoTool {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Module,

        [Parameter(Mandatory = $true)]
        [string]$Version,

        [Parameter(Mandatory = $true)]
        [string]$Command,

        [string[]]$VersionArguments = @("--version"),

        [Parameter(Mandatory = $true)]
        [string]$ExpectedVersion
    )

    if (Test-InstalledVersion -Command $Command -Arguments $VersionArguments -Expected $ExpectedVersion) {
        Write-Host ("Using pinned {0} {1}" -f $Command, $Version)
        return
    }

    Invoke-Native "go" @("install", "$Module@$Version")
    if (-not (Test-InstalledVersion -Command $Command -Arguments $VersionArguments -Expected $ExpectedVersion)) {
        throw ("Installed {0}@{1}, but {2} did not report exact version token {3}" -f $Module, $Version, $Command, $ExpectedVersion)
    }

    Write-Host ("Installed pinned {0} {1}" -f $Command, $Version)
}

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

    $protocGenGoModuleVersion = go list -m -f "{{.Version}}" google.golang.org/protobuf
    if (-not $protocGenGoModuleVersion) {
        throw "Failed to resolve google.golang.org/protobuf version from tools/go.mod"
    }

    $protocGenGoGRPCModuleVersion = go list -m -f "{{.Version}}" google.golang.org/grpc/cmd/protoc-gen-go-grpc
    if (-not $protocGenGoGRPCModuleVersion) {
        throw "Failed to resolve google.golang.org/grpc/cmd/protoc-gen-go-grpc version from tools/go.mod"
    }

    Install-GoTool -Module "golang.org/x/vuln/cmd/govulncheck" -Version $govulnModuleVersion -Command "govulncheck" -VersionArguments @("-version") -ExpectedVersion "govulncheck@$govulnModuleVersion"
    Install-GoTool -Module "honnef.co/go/tools/cmd/staticcheck" -Version $staticcheckModuleVersion -Command "staticcheck" -VersionArguments @("-version") -ExpectedVersion $staticcheckModuleVersion
    Install-GoTool -Module "google.golang.org/protobuf/cmd/protoc-gen-go" -Version $protocGenGoModuleVersion -Command "protoc-gen-go" -VersionArguments @("--version") -ExpectedVersion $protocGenGoModuleVersion
    $protocGenGoGRPCVersionOutput = $protocGenGoGRPCModuleVersion -replace '^v', ''
    Install-GoTool -Module "google.golang.org/grpc/cmd/protoc-gen-go-grpc" -Version $protocGenGoGRPCModuleVersion -Command "protoc-gen-go-grpc" -VersionArguments @("--version") -ExpectedVersion $protocGenGoGRPCVersionOutput
}
finally {
    Pop-Location
}
