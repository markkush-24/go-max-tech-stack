param()

$ErrorActionPreference = "Stop"
. "$PSScriptRoot\\common.ps1"

Invoke-Native "staticcheck" @("./...")
