param()

$ErrorActionPreference = "Stop"

$files = gofmt -l cmd internal
if ($files) {
    Write-Error "gofmt required for:`n$files"
}
